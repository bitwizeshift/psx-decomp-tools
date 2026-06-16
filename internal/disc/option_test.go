package disc_test

import (
	"io"
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/cue"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/disctest"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestWithSectorCallback_ToleratesChecksumError(t *testing.T) {
	t.Parallel()

	// Arrange
	var recorded []error
	callback := func(err error) { recorded = append(recorded, err) }
	corrupt := corruptMode1Raw()
	sut := disctest.MustOpenTrack(cue.ModeMode1_2352, corrupt, disc.WithSectorCallback(callback))
	opts := cmp.Options{cmpopts.EquateErrors()}

	// Act
	payload, err := io.ReadAll(sut)

	// Assert
	if got, want := err, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("io.ReadAll(track) = error %v, want %v", got, want)
	}
	if got, want := recorded, []error{track.ErrChecksum}; !cmp.Equal(got, want, opts) {
		t.Errorf("WithSectorCallback recorded = %v, want %v", got, want)
	}
	if got, want := payload, corrupt[16:2064]; !cmp.Equal(got, want) {
		t.Errorf("io.ReadAll(track) = mismatch (-want +got):\n%s", cmp.Diff(want, got))
	}
}
