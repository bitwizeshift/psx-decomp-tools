package track_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track/tracktest"
)

var errBoom = errors.New("boom")

// errReaderAt is an io.ReaderAt that always fails with err.
type errReaderAt struct {
	err error
}

func (r errReaderAt) ReadAt([]byte, int64) (int, error) {
	return 0, r.err
}

func TestBinSectorReaderReadSector(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		reader     io.ReaderAt
		sectorSize int
		sector     int
		want       track.RawSector
		wantErr    error
	}{
		{
			name:       "FirstSector",
			reader:     bytes.NewReader([]byte{0x10, 0x11, 0x12, 0x13, 0x20, 0x21, 0x22, 0x23}),
			sectorSize: 4,
			sector:     0,
			want:       track.RawSector{Number: 0, Data: []byte{0x10, 0x11, 0x12, 0x13}},
		},
		{
			name:       "SecondSector",
			reader:     bytes.NewReader([]byte{0x10, 0x11, 0x12, 0x13, 0x20, 0x21, 0x22, 0x23}),
			sectorSize: 4,
			sector:     1,
			want:       track.RawSector{Number: 1, Data: []byte{0x20, 0x21, 0x22, 0x23}},
		},
		{
			name:       "PastEnd",
			reader:     bytes.NewReader([]byte{0x10, 0x11, 0x12, 0x13}),
			sectorSize: 4,
			sector:     1,
			want:       track.RawSector{},
			wantErr:    io.EOF,
		},
		{
			name:       "PartialSector",
			reader:     bytes.NewReader([]byte{0x10, 0x11, 0x12, 0x13, 0x20, 0x21}),
			sectorSize: 4,
			sector:     1,
			want:       track.RawSector{},
			wantErr:    track.ErrShortSector,
		},
		{
			name:       "ReaderError",
			reader:     errReaderAt{err: errBoom},
			sectorSize: 4,
			sector:     0,
			want:       track.RawSector{},
			wantErr:    errBoom,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			reader := track.BinSectorReader{Reader: tc.reader, SectorSize: tc.sectorSize}

			// Act
			sector, err := reader.ReadSector(tc.sector)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("BinSectorReader.ReadSector(%d) = error %v, want %v", tc.sector, got, want)
			}
			if got, want := sector, tc.want; !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
				t.Errorf("BinSectorReader.ReadSector(%d) = %v, want %v", tc.sector, got, want)
			}
		})
	}
}

func TestReadAllSectors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		reader  track.SectorReader
		want    []track.RawSector
		wantErr error
	}{
		{
			name:   "MultipleSectors",
			reader: tracktest.StaticSectorReader([]byte{0x01}, []byte{0x02}, []byte{0x03}),
			want: []track.RawSector{
				{Number: 0, Data: []byte{0x01}},
				{Number: 1, Data: []byte{0x02}},
				{Number: 2, Data: []byte{0x03}},
			},
		},
		{
			name:   "Empty",
			reader: tracktest.StaticSectorReader(),
			want:   nil,
		},
		{
			name:    "ReaderError",
			reader:  tracktest.ErrSectorReader(errBoom),
			want:    nil,
			wantErr: errBoom,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			opts := cmp.Options{cmpopts.EquateEmpty()}

			// Act
			sectors, err := track.ReadAllSectors(tc.reader)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("ReadAllSectors(...) = error %v, want %v", got, want)
			}
			if got, want := sectors, tc.want; !cmp.Equal(got, want, opts) {
				t.Errorf("ReadAllSectors(...) = mismatch (-want +got):\n%s", cmp.Diff(want, got, opts))
			}
		})
	}
}
