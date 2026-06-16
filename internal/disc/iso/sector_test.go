package iso_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/iso"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/iso/isotest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// sliceSource is a [iso.SectorSource] backed by a slice of sectors, reporting
// [io.EOF] for any index past the end.
type sliceSource [][]byte

func (s sliceSource) ReadSector(n int) ([]byte, error) {
	if n < 0 || n >= len(s) {
		return nil, io.EOF
	}
	return s[n], nil
}

// errSource is a [iso.SectorSource] whose every read fails with err.
type errSource struct {
	err error
}

func (e errSource) ReadSector(int) ([]byte, error) {
	return nil, e.err
}

func TestSectorReaderReadAt(t *testing.T) {
	t.Parallel()

	const block = 2048
	formOne := bytes.Repeat([]byte{'A'}, block)
	formTwo := bytes.Repeat([]byte{'B'}, 2324)
	next := bytes.Repeat([]byte{'C'}, block)
	sentinel := errors.New("sector failure")

	testCases := []struct {
		name     string
		source   iso.SectorSource
		offset   int64
		size     int
		wantData []byte
		wantN    int
		wantErr  error
	}{
		{
			name:     "WithinSector",
			source:   sliceSource{formOne, next},
			offset:   0,
			size:     4,
			wantData: []byte("AAAA"),
			wantN:    4,
			wantErr:  nil,
		},
		{
			name:     "MidSectorOffset",
			source:   sliceSource{formOne, next},
			offset:   block,
			size:     4,
			wantData: []byte("CCCC"),
			wantN:    4,
			wantErr:  nil,
		},
		{
			name:     "StrideSkipsFormTwoTail",
			source:   sliceSource{formTwo, next},
			offset:   0,
			size:     2 * block,
			wantData: append(bytes.Repeat([]byte{'A' + 1}, block), bytes.Repeat([]byte{'C'}, block)...),
			wantN:    2 * block,
			wantErr:  nil,
		},
		{
			name:     "PastEnd",
			source:   sliceSource{formOne},
			offset:   block - 2,
			size:     8,
			wantData: []byte("AA"),
			wantN:    2,
			wantErr:  io.EOF,
		},
		{
			name:     "ShortSector",
			source:   sliceSource{formOne, []byte("Z")},
			offset:   block,
			size:     8,
			wantData: []byte("Z"),
			wantN:    1,
			wantErr:  io.EOF,
		},
		{
			name:     "SourceError",
			source:   errSource{err: sentinel},
			offset:   0,
			size:     8,
			wantData: []byte{},
			wantN:    0,
			wantErr:  sentinel,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := iso.SectorReader{Source: tc.source}
			buf := make([]byte, tc.size)

			// Act
			read, err := sut.ReadAt(buf, tc.offset)

			// Assert
			if got, want := buf[:read], tc.wantData; !cmp.Equal(got, want) {
				t.Errorf("SectorReader.ReadAt(...) = data %q, want %q", got, want)
			}
			if got, want := read, tc.wantN; !cmp.Equal(got, want) {
				t.Errorf("SectorReader.ReadAt(...) = %d bytes, want %d", got, want)
			}
			opts := cmpopts.EquateErrors()
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, opts) {
				t.Errorf("SectorReader.ReadAt(...) = error %v, want %v", got, want)
			}
		})
	}
}

// paddedSectorSource serves a 2048-byte-per-block image as sectors, padding one
// chosen sector with extra bytes so it reads back oversized, like an XA Form 2
// sector. A [iso.SectorReader] must trim it back to 2048 bytes to keep the image
// aligned.
type paddedSectorSource struct {
	image     []byte
	padSector int
}

func (p paddedSectorSource) ReadSector(n int) ([]byte, error) {
	const block = 2048
	start := n * block
	if start >= len(p.image) {
		return nil, io.EOF
	}
	end := min(start+block, len(p.image))
	data := append([]byte{}, p.image[start:end]...)
	if n == p.padSector {
		data = append(data, bytes.Repeat([]byte{0xFF}, 276)...)
	}
	return data, nil
}

func TestSectorReaderVisits(t *testing.T) {
	t.Parallel()

	// Arrange
	image := isotest.New().AddFile("/A.TXT", []byte("hi")).Build()
	sut := iso.New(iso.SectorReader{Source: paddedSectorSource{image: image, padSector: 5}})
	r := &recorder{}

	// Act
	if err := sut.Visit(r); err != nil {
		t.Fatalf("Visit(...) = unexpected error %v", err)
	}

	// Assert
	files := []fileData{}
	for _, e := range r.events {
		if e.Kind == "file" {
			files = append(files, fileData{Path: e.Path, Name: e.Name, Data: e.Data})
		}
	}
	want := []fileData{{Path: "/A.TXT", Name: "A.TXT", Data: "hi"}}
	if got := files; !cmp.Equal(got, want) {
		t.Errorf("Visit(...) files = mismatch (-want +got):\n%s", cmp.Diff(want, got))
	}
}
