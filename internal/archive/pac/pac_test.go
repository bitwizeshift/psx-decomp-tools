package pac_test

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/archive/pac"
	"github.com/bitwizeshift/psx-decomp-tools/internal/archive/pac/pactest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestNewReader(t *testing.T) {
	t.Parallel()

	// Payloads are whole sectors (less the 16-byte header) so each leaf fills its
	// allotted region without padding, making the extracted size exact.
	leafA := bytes.Repeat([]byte{'A'}, 2032)
	leafB := bytes.Repeat([]byte{'B'}, 6128)
	rawC := bytes.Repeat([]byte{'C'}, 200)
	dataD := bytes.Repeat([]byte{0x7f}, 50)

	testCases := []struct {
		name    string
		image   []byte
		want    []*pac.File
		wantErr error
	}{
		{
			name:    "RootLeaf",
			image:   pactest.Build(pactest.Leaf(0x101, leafA)),
			want:    []*pac.File{{Name: "0", Kind: 0x101, Size: 2032}},
			wantErr: nil,
		}, {
			name:  "DirectoryWithData",
			image: pactest.Build(pactest.DirData(dataD, pactest.Leaf(1, leafA))),
			want: []*pac.File{
				{Name: "data", Kind: 0, Size: 4080},
				{Name: "0", Kind: 1, Size: 2032},
			},
			wantErr: nil,
		}, {
			name:  "DirectoryOfLeaves",
			image: pactest.Build(pactest.Dir(pactest.Leaf(1, leafA), pactest.Leaf(8, leafB))),
			want: []*pac.File{
				{Name: "0", Kind: 1, Size: 2032},
				{Name: "1", Kind: 8, Size: 6128},
			},
			wantErr: nil,
		}, {
			name: "NestedDirectoryTail",
			image: pactest.Build(pactest.Dir(
				pactest.Leaf(1, leafA),
				pactest.Dir(pactest.Leaf(2, leafA), pactest.Leaf(3, leafA)),
			)),
			want: []*pac.File{
				{Name: "0", Kind: 1, Size: 2032},
				{Name: "1/0", Kind: 2, Size: 2032},
				{Name: "1/1", Kind: 3, Size: 2032},
			},
			wantErr: nil,
		}, {
			name:  "RawRemainder",
			image: pactest.Build(pactest.Dir(pactest.Leaf(1, leafA), pactest.Raw(rawC))),
			want: []*pac.File{
				{Name: "0", Kind: 1, Size: 2032},
				{Name: "1", Raw: true, Size: 2048},
			},
			wantErr: nil,
		}, {
			name:    "BadMagic",
			image:   bytes.Repeat([]byte{0x00}, 32),
			want:    nil,
			wantErr: pac.ErrBadMagic,
		}, {
			name:    "Truncated",
			image:   []byte{'P', 'A', 'C'},
			want:    nil,
			wantErr: pac.ErrTruncated,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			reader := bytes.NewReader(tc.image)
			size := int64(len(tc.image))

			// Act
			result, err := pac.NewReader(reader, size)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("NewReader() error = %v, want %v", got, want)
			}
			var files []*pac.File
			if result != nil {
				files = result.File
			}
			if got, want := files, tc.want; !cmp.Equal(got, want, pactest.CompareFiles(), cmpopts.EquateEmpty()) {
				t.Errorf("NewReader() files diff (-got +want):\n%s", cmp.Diff(got, want, pactest.CompareFiles(), cmpopts.EquateEmpty()))
			}
		})
	}
}

func TestFileOpen(t *testing.T) {
	t.Parallel()

	leafA := bytes.Repeat([]byte{'A'}, 2032)
	leafB := bytes.Repeat([]byte{'B'}, 6128)
	image := pactest.Build(pactest.Dir(pactest.Leaf(1, leafA), pactest.Leaf(8, leafB)))

	testCases := []struct {
		name       string
		index      int
		want       []byte
		wantOffset int64
	}{
		{
			name:       "FirstLeaf",
			index:      0,
			want:       leafA,
			wantOffset: 0x1010,
		}, {
			name:       "SecondLeaf",
			index:      1,
			want:       leafB,
			wantOffset: 0x1810,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			reader, err := pac.NewReader(bytes.NewReader(image), int64(len(image)))
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
				t.Errorf("Open() data = %d bytes, want %d", len(got), len(want))
			}
			if got, want := file.Offset(), tc.wantOffset; got != want {
				t.Errorf("Offset() = %#x, want %#x", got, want)
			}
		})
	}
}

func TestNewReaderMalformed(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		image   []byte
		want    []*pac.File
		wantErr error
	}{
		{
			name:    "TruncatedNestedChild",
			image:   nestedDirImage(0x2000, 8),
			want:    nil,
			wantErr: pac.ErrTruncated,
		}, {
			name:    "ZeroSpanChild",
			image:   zeroSpanChildImage(),
			want:    []*pac.File{{Name: "0", Raw: true, Size: 0x800}},
			wantErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			reader := bytes.NewReader(tc.image)

			// Act
			result, err := pac.NewReader(reader, int64(len(tc.image)))

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("NewReader() error = %v, want %v", got, want)
			}
			var files []*pac.File
			if result != nil {
				files = result.File
			}
			if got, want := files, tc.want; !cmp.Equal(got, want, pactest.CompareFiles(), cmpopts.EquateEmpty()) {
				t.Errorf("NewReader() files diff (-got +want):\n%s", cmp.Diff(got, want, pactest.CompareFiles(), cmpopts.EquateEmpty()))
			}
		})
	}
}

// dirHeader writes a directory node header with the given header-sector word at
// off within buf.
func dirHeader(buf []byte, off int, headerSectors uint32) {
	copy(buf[off:off+4], pac.Magic)
	binary.LittleEndian.PutUint32(buf[off+8:off+12], headerSectors)
}

// nestedDirImage builds a directory whose only child is another directory whose
// own children region is truncated to trailing bytes, so parsing the inner
// directory fails. The recursion makes the failure propagate out of the outer
// directory.
func nestedDirImage(innerDataStart, trailing int) []byte {
	buf := make([]byte, innerDataStart+trailing)
	dirHeader(buf, 0, 1)                // outer dir, children start at 0x1000
	dirHeader(buf, pac.SectorSize*2, 1) // inner dir at 0x1000, children at innerDataStart
	return buf
}

// zeroSpanChildImage builds a directory whose first child is a leaf declaring a
// zero total size, so the child spans no sectors and the remainder is reported raw.
func zeroSpanChildImage() []byte {
	buf := make([]byte, pac.SectorSize*3)
	dirHeader(buf, 0, 1) // dir, children start at 0x1000
	copy(buf[pac.SectorSize*2:], pac.Magic)
	// header word 0 (leaf) and total 0 are already zero.
	return buf
}
