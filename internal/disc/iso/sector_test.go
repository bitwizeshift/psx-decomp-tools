package iso_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/iso"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// sliceSource is a [iso.SectorSource] backed by a slice of logical blocks,
// reporting [io.EOF] for any index past the end.
type sliceSource [][]byte

func (s sliceSource) ReadSector(n int) (iso.Sector, error) {
	if n < 0 || n >= len(s) {
		return iso.Sector{}, io.EOF
	}
	return iso.Sector{Index: int64(n), Block: s[n]}, nil
}

// errSource is a [iso.SectorSource] whose every read fails with err.
type errSource struct {
	err error
}

func (e errSource) ReadSector(int) (iso.Sector, error) {
	return iso.Sector{}, e.err
}

func TestSectorReaderReadAt(t *testing.T) {
	t.Parallel()

	const block = 2048
	first := bytes.Repeat([]byte{'A'}, block)
	second := bytes.Repeat([]byte{'B'}, block)
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
			name:     "WithinBlock",
			source:   sliceSource{first, second},
			offset:   0,
			size:     4,
			wantData: []byte("AAAA"),
			wantN:    4,
			wantErr:  nil,
		},
		{
			name:     "SpansBlocks",
			source:   sliceSource{first, second},
			offset:   block - 2,
			size:     4,
			wantData: []byte("AABB"),
			wantN:    4,
			wantErr:  nil,
		},
		{
			name:     "PartialBlock",
			source:   sliceSource{first, []byte("Z")},
			offset:   block,
			size:     8,
			wantData: []byte("Z"),
			wantN:    1,
			wantErr:  io.EOF,
		},
		{
			name:     "PastEnd",
			source:   sliceSource{first},
			offset:   block - 2,
			size:     8,
			wantData: []byte("AA"),
			wantN:    2,
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
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("SectorReader.ReadAt(...) = error %v, want %v", got, want)
			}
			if got, want := buf[:read], tc.wantData; !cmp.Equal(got, want) {
				t.Errorf("SectorReader.ReadAt(...) = data %q, want %q", got, want)
			}
			if got, want := read, tc.wantN; !cmp.Equal(got, want) {
				t.Errorf("SectorReader.ReadAt(...) = %d bytes, want %d", got, want)
			}
		})
	}
}
