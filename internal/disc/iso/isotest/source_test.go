package isotest_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/iso"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/iso/isotest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestSectorViewReadSector(t *testing.T) {
	t.Parallel()

	image := bytes.Repeat([]byte{0xAB}, isotest.BlockSize+100)

	testCases := []struct {
		name    string
		tails   map[int]isotest.Tail
		sector  int
		want    iso.Sector
		wantErr error
	}{
		{
			name:    "FullBlock",
			tails:   nil,
			sector:  0,
			want:    iso.Sector{Index: 0, Block: bytes.Repeat([]byte{0xAB}, isotest.BlockSize)},
			wantErr: nil,
		},
		{
			name:    "PartialLastBlock",
			tails:   nil,
			sector:  1,
			want:    iso.Sector{Index: 1, Block: bytes.Repeat([]byte{0xAB}, 100)},
			wantErr: nil,
		},
		{
			name:   "WithStreamTail",
			tails:  map[int]isotest.Tail{0: {Bytes: []byte{1, 2, 3}, IsStream: true, Subheader: &iso.Subheader{File: 9, Channel: 1}}},
			sector: 0,
			want: iso.Sector{
				Index:        0,
				Block:        bytes.Repeat([]byte{0xAB}, isotest.BlockSize),
				Tail:         []byte{1, 2, 3},
				TailIsStream: true,
				Subheader:    &iso.Subheader{File: 9, Channel: 1},
			},
			wantErr: nil,
		},
		{
			name:    "PastEnd",
			tails:   nil,
			sector:  2,
			want:    iso.Sector{},
			wantErr: io.EOF,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := isotest.SectorView{Image: image, Tails: tc.tails}

			// Act
			sector, err := sut.ReadSector(tc.sector)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("SectorView.ReadSector(%d) = error %v, want %v", tc.sector, got, want)
			}
			if got, want := sector, tc.want; !cmp.Equal(got, want) {
				t.Errorf("SectorView.ReadSector(%d) = mismatch (-want +got):\n%s", tc.sector, cmp.Diff(want, got))
			}
		})
	}
}
