package track_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track"
)

func TestRawDecoderDecodeSector(t *testing.T) {
	t.Parallel()

	// Arrange
	sut := track.RawDecoder{}
	data := payload(800)
	raw := track.RawSector{Number: 4, Data: data}
	wantSector := track.Sector{Number: 4, UserData: data, Raw: data}
	opts := cmp.Options{cmpopts.EquateEmpty()}

	// Act
	sector, err := sut.DecodeSector(raw)

	// Assert
	if got, want := err, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("RawDecoder.DecodeSector(...) = error %v, want %v", got, want)
	}
	if got, want := sector, wantSector; !cmp.Equal(got, want, opts) {
		t.Errorf("RawDecoder.DecodeSector(...) = mismatch (-want +got):\n%s", cmp.Diff(want, got, opts))
	}
}
