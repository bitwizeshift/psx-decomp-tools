package ioerr_test

import (
	"errors"
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/ioerr"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// errCloser is a test-double for [io.Closer] that always returns the given error.
type errCloser struct {
	err error
}

func (e *errCloser) Close() error {
	return e.err
}

func TestCloseAndReport(t *testing.T) {
	t.Parallel()

	testErr := errors.New("close error")
	existingErr := errors.New("existing error")

	testCases := []struct {
		name    string
		closer  *errCloser
		initVal string
		initErr error
		want    string
		wantErr error
	}{
		{
			name:    "close succeeds: outVal and outErr unchanged",
			closer:  &errCloser{err: nil},
			initVal: "original",
			initErr: nil,
			want:    "original",
			wantErr: nil,
		},
		{
			name:    "close fails: outErr set and outVal zeroed",
			closer:  &errCloser{err: testErr},
			initVal: "original",
			initErr: nil,
			want:    "",
			wantErr: testErr,
		},
		{
			name:    "close succeeds: pre-existing error is preserved",
			closer:  &errCloser{err: nil},
			initVal: "original",
			initErr: existingErr,
			want:    "original",
			wantErr: existingErr,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			outVal := tc.initVal
			outErr := tc.initErr

			// Act
			ioerr.CloseAndReport(tc.closer, &outVal, &outErr)

			// Assert
			if got, want := outErr, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("CloseAndReport() outErr = %v, want %v", got, want)
			}
			if got, want := outVal, tc.want; !cmp.Equal(got, want) {
				t.Errorf("CloseAndReport() outVal = %q, want %q", got, want)
			}
		})
	}
}
