package disc_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/disctest"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/iso"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// sectorBytes returns a 2048-byte sector filled with value.
func sectorBytes(value byte) []byte {
	return bytes.Repeat([]byte{value}, 2048)
}

// twoTrackDisc builds a two-track disc whose tracks each hold two distinct
// 2048-byte sectors, so a volume over it spans logical sectors 0..3.
func twoTrackDisc() *disc.Disc {
	cueText := lines(
		`FILE "a.bin" BINARY`,
		`  TRACK 01 MODE1/2048`,
		`    INDEX 01 00:00:00`,
		`FILE "b.bin" BINARY`,
		`  TRACK 02 MODE1/2048`,
		`    INDEX 01 00:00:00`,
	)
	files := map[string][]byte{
		"a.bin": append(sectorBytes(0xA1), sectorBytes(0xA2)...),
		"b.bin": append(sectorBytes(0xB1), sectorBytes(0xB2)...),
	}
	return disctest.MustDisc(cueText, files)
}

// corruptDisc builds a single-track disc whose only full Mode 1 sector is all
// zeros, so decoding it fails with [track.ErrBadSync].
func corruptDisc() *disc.Disc {
	cueText := lines(
		`FILE "track.bin" BINARY`,
		`  TRACK 01 MODE1/2352`,
		`    INDEX 01 00:00:00`,
	)
	return disctest.MustDisc(cueText, map[string][]byte{"track.bin": make([]byte, 2352)})
}

// mustVolume returns d's volume, panicking if it cannot be opened.
func mustVolume(d *disc.Disc) iso.SectorSource {
	v, err := d.Volume()
	if err != nil {
		panic(err)
	}
	return v
}

func TestVolumeReadSector(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		source   iso.SectorSource
		sector   int
		wantData []byte
		wantErr  error
	}{
		{
			name:     "FirstTrackFirstSector",
			source:   mustVolume(twoTrackDisc()),
			sector:   0,
			wantData: sectorBytes(0xA1),
			wantErr:  nil,
		},
		{
			name:     "FirstTrackSecondSector",
			source:   mustVolume(twoTrackDisc()),
			sector:   1,
			wantData: sectorBytes(0xA2),
			wantErr:  nil,
		},
		{
			name:     "SecondTrackAcrossBoundary",
			source:   mustVolume(twoTrackDisc()),
			sector:   2,
			wantData: sectorBytes(0xB1),
			wantErr:  nil,
		},
		{
			name:     "SecondTrackLastSector",
			source:   mustVolume(twoTrackDisc()),
			sector:   3,
			wantData: sectorBytes(0xB2),
			wantErr:  nil,
		},
		{
			name:     "PastEnd",
			source:   mustVolume(twoTrackDisc()),
			sector:   4,
			wantData: nil,
			wantErr:  io.EOF,
		},
		{
			name:     "DecodeError",
			source:   mustVolume(corruptDisc()),
			sector:   0,
			wantData: nil,
			wantErr:  track.ErrBadSync,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := tc.source

			// Act
			data, err := sut.ReadSector(tc.sector)

			// Assert
			if got, want := data, tc.wantData; !cmp.Equal(got, want) {
				t.Errorf("volume.ReadSector(%d) = mismatch (-want +got):\n%s", tc.sector, cmp.Diff(want, got))
			}
			opts := cmpopts.EquateErrors()
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, opts) {
				t.Errorf("volume.ReadSector(%d) = error %v, want %v", tc.sector, got, want)
			}
		})
	}
}

func TestVolumeOpenError(t *testing.T) {
	t.Parallel()

	// Arrange
	sentinel := errors.New("stat failure")
	cueText := lines(
		`FILE "track.bin" BINARY`,
		`  TRACK 01 MODE1/2048`,
		`    INDEX 01 00:00:00`,
	)
	files := map[string][]byte{"track.bin": dataBytes(2048)}
	d := disctest.MustDisc(cueText, files, disc.WithStatFunc(disctest.ErrStatFunc(sentinel)))

	// Act
	source, err := d.Volume()

	// Assert
	if got, want := source, iso.SectorSource(nil); !cmp.Equal(got, want, cmpopts.EquateComparable()) {
		t.Errorf("Disc.Volume() = source %v, want %v", got, want)
	}
	opts := cmpopts.EquateErrors()
	if got, want := err, sentinel; !cmp.Equal(got, want, opts) {
		t.Errorf("Disc.Volume() = error %v, want %v", got, want)
	}
}
