package tracktest_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/cue"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track/tracktest"
)

var builderMSF = cue.MSF{Minute: 12, Second: 34, Frame: 56}

// payload returns n deterministic, mostly non-zero bytes.
func payload(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i + 1)
	}
	return b
}

func TestBuildMode1Sector(t *testing.T) {
	t.Parallel()

	// Arrange
	data := payload(2048)
	raw := track.RawSector{Number: 1, Data: tracktest.BuildMode1Sector(builderMSF, data)}
	decoder := track.Mode1Decoder{SectorSize: 2352, Verifier: track.Mode1Verifier{}}
	wantSector := track.Sector{
		Number:   1,
		Header:   &track.SectorHeader{Minute: 12, Second: 34, Frame: 56, Mode: track.SectorModeMode1},
		UserData: data,
		Raw:      raw.Data,
	}
	opts := cmp.Options{cmpopts.EquateEmpty()}

	// Act
	sector, err := decoder.DecodeSector(raw)

	// Assert
	if got, want := err, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("DecodeSector(BuildMode1Sector(...)) = error %v, want %v", got, want)
	}
	if got, want := sector, wantSector; !cmp.Equal(got, want, opts) {
		t.Errorf("DecodeSector(BuildMode1Sector(...)) = mismatch (-want +got):\n%s", cmp.Diff(want, got, opts))
	}
}

func TestBuildMode2Form1Sector(t *testing.T) {
	t.Parallel()

	// Arrange
	data := payload(2048)
	sub := track.Subheader{File: 1, Channel: 2, SubMode: 4, Coding: 8}
	raw := track.RawSector{Number: 2, Data: tracktest.BuildMode2Form1Sector(sub, builderMSF, data)}
	decoder := track.Mode2Decoder{SectorSize: 2352, Verifier: track.Mode2Verifier{}}
	wantSector := track.Sector{
		Number:    2,
		Header:    &track.SectorHeader{Minute: 12, Second: 34, Frame: 56, Mode: track.SectorModeMode2},
		Subheader: &track.Subheader{File: 1, Channel: 2, SubMode: 4, Coding: 8, Form: track.FormOne},
		UserData:  data,
		Raw:       raw.Data,
	}
	opts := cmp.Options{cmpopts.EquateEmpty()}

	// Act
	sector, err := decoder.DecodeSector(raw)

	// Assert
	if got, want := err, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("DecodeSector(BuildMode2Form1Sector(...)) = error %v, want %v", got, want)
	}
	if got, want := sector, wantSector; !cmp.Equal(got, want, opts) {
		t.Errorf("DecodeSector(BuildMode2Form1Sector(...)) = mismatch (-want +got):\n%s", cmp.Diff(want, got, opts))
	}
}

func TestBuildMode2Form2Sector(t *testing.T) {
	t.Parallel()

	// Arrange
	data := payload(2324)
	sub := track.Subheader{File: 1, Channel: 2, SubMode: 4, Coding: 8}
	raw := track.RawSector{Number: 3, Data: tracktest.BuildMode2Form2Sector(sub, builderMSF, data)}
	decoder := track.Mode2Decoder{SectorSize: 2352, Verifier: track.Mode2Verifier{}}
	wantSector := track.Sector{
		Number:    3,
		Header:    &track.SectorHeader{Minute: 12, Second: 34, Frame: 56, Mode: track.SectorModeMode2},
		Subheader: &track.Subheader{File: 1, Channel: 2, SubMode: 0x24, Coding: 8, Form: track.FormTwo},
		UserData:  data,
		Raw:       raw.Data,
	}
	opts := cmp.Options{cmpopts.EquateEmpty()}

	// Act
	sector, err := decoder.DecodeSector(raw)

	// Assert
	if got, want := err, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("DecodeSector(BuildMode2Form2Sector(...)) = error %v, want %v", got, want)
	}
	if got, want := sector, wantSector; !cmp.Equal(got, want, opts) {
		t.Errorf("DecodeSector(BuildMode2Form2Sector(...)) = mismatch (-want +got):\n%s", cmp.Diff(want, got, opts))
	}
}
