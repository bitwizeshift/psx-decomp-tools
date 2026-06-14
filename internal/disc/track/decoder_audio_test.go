package track_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track"
)

func TestAudioDecoderDecodeSector(t *testing.T) {
	t.Parallel()

	// Arrange
	sut := track.AudioDecoder{}
	data := payload(2352)
	raw := track.RawSector{Number: 3, Data: data}
	wantSector := track.Sector{Number: 3, UserData: data, Raw: data}
	opts := cmp.Options{cmpopts.EquateEmpty()}

	// Act
	sector, err := sut.DecodeSector(raw)

	// Assert
	if got, want := err, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("AudioDecoder.DecodeSector(...) = error %v, want %v", got, want)
	}
	if got, want := sector, wantSector; !cmp.Equal(got, want, opts) {
		t.Errorf("AudioDecoder.DecodeSector(...) = mismatch (-want +got):\n%s", cmp.Diff(want, got, opts))
	}
}
