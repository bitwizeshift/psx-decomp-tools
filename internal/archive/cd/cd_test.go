package cd_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"io/fs"
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/archive/cd"
	"github.com/bitwizeshift/psx-decomp-tools/internal/archive/cd/cdtest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestNewReader(t *testing.T) {
	t.Parallel()

	memA := bytes.Repeat([]byte{'A'}, 3000)
	memB := bytes.Repeat([]byte{'B'}, 1000)

	testCases := []struct {
		name    string
		image   []byte
		want    []*cd.File
		wantErr error
	}{
		{
			name:    "Empty",
			image:   cdtest.Build(),
			want:    []*cd.File{},
			wantErr: nil,
		}, {
			name:    "SingleMember",
			image:   cdtest.Build(memA),
			want:    []*cd.File{{StartSector: 1, Size: 3000}},
			wantErr: nil,
		}, {
			name:    "MultipleMembers",
			image:   cdtest.Build(memA, memB),
			want:    []*cd.File{{StartSector: 1, Size: 3000}, {StartSector: 3, Size: 1000}},
			wantErr: nil,
		}, {
			name:    "TruncatedHeader",
			image:   []byte{0x00, 0x00},
			want:    nil,
			wantErr: cd.ErrTruncated,
		}, {
			name:    "TableDoesNotFit",
			image:   headerOnly(1000),
			want:    nil,
			wantErr: cd.ErrTruncated,
		}, {
			name:    "MemberPastEnd",
			image:   withEntrySize(cdtest.Build(memA), 0, 1<<20),
			want:    nil,
			wantErr: cd.ErrCorrupt,
		}, {
			name:    "ChainOutOfBounds",
			image:   withEntryStart(cdtest.Build(memA, memB), 1, 99),
			want:    nil,
			wantErr: cd.ErrCorrupt,
		}, {
			name:    "ChainMismatchInBounds",
			image:   withEntryStart(cdtest.Build(memA, memB), 1, 2),
			want:    nil,
			wantErr: cd.ErrCorrupt,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			reader := bytes.NewReader(tc.image)
			size := int64(len(tc.image))

			// Act
			result, err := cd.NewReader(reader, size)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("NewReader() error = %v, want %v", got, want)
			}
			var files []*cd.File
			if result != nil {
				files = result.File
			}
			if got, want := files, tc.want; !cmp.Equal(got, want, cdtest.CompareFiles(), cmpopts.EquateEmpty()) {
				t.Errorf("NewReader() files diff (-got +want):\n%s", cmp.Diff(got, want, cdtest.CompareFiles(), cmpopts.EquateEmpty()))
			}
		})
	}
}

func TestFileOpen(t *testing.T) {
	t.Parallel()

	memA := bytes.Repeat([]byte{'A'}, 3000)
	memB := bytes.Repeat([]byte{'B'}, 1000)
	image := cdtest.Build(memA, memB)

	testCases := []struct {
		name       string
		index      int
		want       []byte
		wantOffset int64
	}{
		{
			name:       "FirstMember",
			index:      0,
			want:       memA,
			wantOffset: 2048,
		}, {
			name:       "SecondMember",
			index:      1,
			want:       memB,
			wantOffset: 3 * 2048,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			reader, err := cd.NewReader(bytes.NewReader(image), int64(len(image)))
			if err != nil {
				t.Fatalf("NewReader() error = %v", err)
			}
			file := reader.File[tc.index]

			// Act
			stream, err := file.Open()
			if err != nil {
				t.Fatalf("Open() error = %v", err)
			}
			data, err := io.ReadAll(stream)

			// Assert
			if err != nil {
				t.Fatalf("ReadAll() error = %v", err)
			}
			if got, want := data, tc.want; !cmp.Equal(got, want) {
				t.Errorf("Open() data length = %d, want %d", len(got), len(want))
			}
			if got, want := file.Offset(), tc.wantOffset; got != want {
				t.Errorf("Offset() = %d, want %d", got, want)
			}
		})
	}
}

// headerOnly returns an 8-byte ".CD" prologue declaring the given member count
// and nothing else.
func headerOnly(count uint32) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint32(b[0:4], count)
	return b
}

func TestOpenReader(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		path    string
		want    []*cd.File
		wantErr error
	}{
		{
			name:    "Valid",
			path:    "testdata/sample.cd",
			want:    []*cd.File{{StartSector: 1, Size: 11}},
			wantErr: nil,
		}, {
			name:    "Corrupt",
			path:    "testdata/corrupt.cd",
			want:    nil,
			wantErr: cd.ErrTruncated,
		}, {
			name:    "Missing",
			path:    "testdata/does-not-exist.cd",
			want:    nil,
			wantErr: fs.ErrNotExist,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Act
			reader, err := cd.OpenReader(tc.path)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("OpenReader() error = %v, want %v", got, want)
			}
			var files []*cd.File
			if reader != nil {
				files = reader.File
				t.Cleanup(func() { _ = reader.Close() })
			}
			if got, want := files, tc.want; !cmp.Equal(got, want, cdtest.CompareFiles(), cmpopts.EquateEmpty()) {
				t.Errorf("OpenReader() files diff (-got +want):\n%s", cmp.Diff(got, want, cdtest.CompareFiles(), cmpopts.EquateEmpty()))
			}
		})
	}
}

func TestNewReaderTableReadError(t *testing.T) {
	t.Parallel()

	// Arrange: a reader that serves the header but fails the table read.
	header := headerOnly(2)
	reader := &partialReaderAt{header: header}

	// Act
	_, err := cd.NewReader(reader, int64(len(header))+2*8)

	// Assert
	if got, want := err, cd.ErrTruncated; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Errorf("NewReader() error = %v, want %v", got, want)
	}
}

// partialReaderAt serves header at offset 0 and fails every other read, modelling
// a source that is truncated after its table of contents header.
type partialReaderAt struct {
	header []byte
}

func (p *partialReaderAt) ReadAt(b []byte, off int64) (int, error) {
	if off != 0 {
		return 0, errors.New("read past header")
	}
	n := copy(b, p.header)
	if n < len(b) {
		return n, io.EOF
	}
	return n, nil
}

// withEntrySize returns a copy of image with the size of table entry i replaced.
func withEntrySize(image []byte, i int, size uint32) []byte {
	out := bytes.Clone(image)
	binary.LittleEndian.PutUint32(out[8+i*8+4:8+i*8+8], size)
	return out
}

// withEntryStart returns a copy of image with the start sector of table entry i
// replaced.
func withEntryStart(image []byte, i int, start uint32) []byte {
	out := bytes.Clone(image)
	binary.LittleEndian.PutUint32(out[8+i*8:8+i*8+4], start)
	return out
}
