package iotest_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/testing/iotest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestErrWriter(t *testing.T) {
	t.Parallel()

	// Arrange
	testErr := errors.New("test error")
	sut := iotest.ErrWriter(testErr)

	// Act
	n, err := sut.Write([]byte("hello"))

	// Assert
	if got, want := err, testErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("ErrWriter() err = %v, want %v", got, want)
	}
	if got, want := n, 0; !cmp.Equal(got, want) {
		t.Errorf("ErrWriter() = %d, want %d", got, want)
	}
}

func writePiecewise(w io.Writer, data []byte, chunkSize int) (int, error) {
	var n int
	for i := 0; i < len(data); i += chunkSize {
		end := min(i+chunkSize, len(data))
		written, err := w.Write(data[i:end])
		n += written
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func TestTruncateErrWriter(t *testing.T) {
	t.Parallel()

	testErr := errors.New("test error")
	testCases := []struct {
		name        string
		input       string
		limit       int
		wantWritten int
		wantErr     error
	}{
		{
			name:        "UnderLimit",
			input:       "hello",
			limit:       10,
			wantWritten: 5,
			wantErr:     nil,
		}, {
			name:        "AtLimit",
			input:       "hello",
			limit:       5,
			wantWritten: 5,
			wantErr:     nil,
		}, {
			name:        "OverLimit",
			input:       "hello",
			limit:       3,
			wantWritten: 3,
			wantErr:     testErr,
		}, {
			name:        "WriteAfterLimit",
			input:       "hello world",
			limit:       6,
			wantWritten: 6,
			wantErr:     testErr,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := iotest.TruncateErrWriter(tc.limit, testErr)

			// Act
			n, err := writePiecewise(sut, []byte(tc.input), 2)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("TruncateErrWriter() err = %v, want %v", got, want)
			}
			if got, want := n, tc.wantWritten; !cmp.Equal(got, want) {
				t.Errorf("TruncateErrWriter() = %d, want %d", got, want)
			}
		})
	}
}

type fakeT struct {
	testing.TB
	DidFatal bool
}

func (t *fakeT) Fatalf(string, ...any) {
	t.DidFatal = true
}

func (t *fakeT) Helper() {}

func writerWith(content string) io.Writer {
	var buf bytes.Buffer
	buf.WriteString(content)

	return &buf
}

type readWriterWithErr struct {
	err error
}

func (rw *readWriterWithErr) Write(p []byte) (int, error) {
	return 0, rw.err
}

func (rw *readWriterWithErr) Read(p []byte) (int, error) {
	return 0, rw.err
}

func TestReadAllFromWriter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		writer    io.Writer
		wantBytes []byte
		wantFatal bool
	}{
		{
			name:      "NonReader",
			writer:    iotest.ErrWriter(nil),
			wantBytes: nil,
			wantFatal: false,
		}, {
			name:      "WriterWithContent",
			writer:    writerWith("hello"),
			wantBytes: []byte("hello"),
			wantFatal: false,
		}, {
			name:      "WriterWithReadError",
			writer:    &readWriterWithErr{err: errors.New("read error")},
			wantBytes: nil,
			wantFatal: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			fakeT := &fakeT{}

			// Act
			gotBytes := iotest.ReadAllFromWriter(fakeT, tc.writer)

			// Assert
			if got, want := gotBytes, tc.wantBytes; !cmp.Equal(got, want) {
				t.Errorf("ReadAllFromWriter() = %v, want %v", got, want)
			}
			if got, want := fakeT.DidFatal, tc.wantFatal; !cmp.Equal(got, want) {
				t.Errorf("ReadAllFromWriter() DidFatal = %v, want %v", got, want)
			}
		})
	}
}
