package disc_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/cue"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/disctest"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/iso"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track/tracktest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// sectorBytes returns a 2048-byte sector filled with value.
func sectorBytes(value byte) []byte {
	return bytes.Repeat([]byte{value}, 2048)
}

// concat returns the concatenation of a and b in a fresh slice.
func concat(a, b []byte) []byte {
	return append(append([]byte{}, a...), b...)
}

// modeVolume builds a single-track disc of the given mode backed by raw and
// returns its volume.
func modeVolume(mode cue.Mode, raw []byte) iso.SectorSource {
	cueText := lines(
		`FILE "track.bin" BINARY`,
		`  TRACK 01 `+mode.String(),
		`    INDEX 01 00:00:00`,
	)
	return mustVolume(disctest.MustDisc(cueText, map[string][]byte{"track.bin": raw}))
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
		"a.bin": concat(sectorBytes(0xA1), sectorBytes(0xA2)),
		"b.bin": concat(sectorBytes(0xB1), sectorBytes(0xB2)),
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

	block := bytes.Repeat([]byte{0x11}, 2048)
	form2Tail := bytes.Repeat([]byte{0x22}, 276)
	audioTail := bytes.Repeat([]byte{0x33}, 304)
	bareTail := bytes.Repeat([]byte{0x44}, 288)
	msf := cue.MSF{Minute: 0, Second: 2, Frame: 0}
	form1Raw := tracktest.BuildMode2Form1Sector(track.Subheader{File: 1, Channel: 2, SubMode: track.SubModeData, Coding: 3}, msf, block)
	form2Raw := tracktest.BuildMode2Form2Sector(track.Subheader{File: 1, Channel: 2, Coding: 3}, msf, concat(block, form2Tail))

	testCases := []struct {
		name    string
		source  iso.SectorSource
		sector  int
		want    iso.Sector
		wantErr error
	}{
		{
			name:    "Mode1Bare",
			source:  modeVolume(cue.ModeMode1_2048, block),
			sector:  0,
			want:    iso.Sector{Index: 0, Block: block},
			wantErr: nil,
		},
		{
			name:    "Mode2Form1",
			source:  modeVolume(cue.ModeMode2_2352, form1Raw),
			sector:  0,
			want:    iso.Sector{Index: 0, Block: block, Subheader: &iso.Subheader{File: 1, Channel: 2, SubMode: track.SubModeData, Coding: 3}},
			wantErr: nil,
		},
		{
			name:    "Mode2Form2Stream",
			source:  modeVolume(cue.ModeMode2_2352, form2Raw),
			sector:  0,
			want:    iso.Sector{Index: 0, Block: block, Tail: form2Tail, TailIsStream: true, Subheader: &iso.Subheader{File: 1, Channel: 2, SubMode: track.SubModeForm2, Coding: 3}},
			wantErr: nil,
		},
		{
			name:    "AudioStream",
			source:  modeVolume(cue.ModeAudio, concat(block, audioTail)),
			sector:  0,
			want:    iso.Sector{Index: 0, Block: block, Tail: audioTail, TailIsStream: true},
			wantErr: nil,
		},
		{
			name:    "BareMode2NonStream",
			source:  modeVolume(cue.ModeMode2_2336, concat(block, bareTail)),
			sector:  0,
			want:    iso.Sector{Index: 0, Block: block, Tail: bareTail},
			wantErr: nil,
		},
		{
			name:    "AcrossTrackBoundary",
			source:  mustVolume(twoTrackDisc()),
			sector:  2,
			want:    iso.Sector{Index: 2, Block: sectorBytes(0xB1)},
			wantErr: nil,
		},
		{
			name:    "PastEnd",
			source:  mustVolume(twoTrackDisc()),
			sector:  4,
			want:    iso.Sector{},
			wantErr: io.EOF,
		},
		{
			name:    "DecodeError",
			source:  mustVolume(corruptDisc()),
			sector:  0,
			want:    iso.Sector{},
			wantErr: track.ErrBadSync,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := tc.source

			// Act
			sector, err := sut.ReadSector(tc.sector)

			// Assert
			if got, want := sector, tc.want; !cmp.Equal(got, want) {
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
