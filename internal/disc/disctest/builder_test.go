package disctest_test

import (
	"io"
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/cue"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/disctest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

const oneTrackCue = `FILE "track.bin" BINARY
  TRACK 01 MODE1/2048
    INDEX 01 00:00:00`

func sectorData() []byte {
	data := make([]byte, 2048)
	for i := range data {
		data[i] = byte(i)
	}
	return data
}

func TestDisc(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		cueText string
		wantErr error
	}{
		{
			name:    "ValidSheet",
			cueText: oneTrackCue,
			wantErr: nil,
		},
		{
			name:    "InvalidSheet",
			cueText: "BOGUS",
			wantErr: cue.ErrSyntax,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			files := map[string][]byte{"track.bin": sectorData()}

			// Act
			_, err := disctest.Disc(tc.cueText, files)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Disc(...) = error %v, want %v", got, want)
			}
		})
	}
}

func TestDisc_ParsesTracks(t *testing.T) {
	t.Parallel()

	// Arrange
	want := []cue.Track{{
		Number:  1,
		Mode:    cue.ModeMode1_2048,
		Type:    cue.TypeBinary,
		File:    "track.bin",
		Indices: []cue.Index{{Number: 1, Offset: cue.MSF{}}},
	}}

	// Act
	built, err := disctest.Disc(oneTrackCue, map[string][]byte{"track.bin": sectorData()})

	// Assert
	if got, want := err, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("Disc(...) = error %v, want %v", got, want)
	}
	if got, want := built.Tracks(), want; !cmp.Equal(got, want) {
		t.Errorf("Disc(...).Tracks() = mismatch (-want +got):\n%s", cmp.Diff(want, got))
	}
}

func TestMustDisc_InvalidSheet_Panics(t *testing.T) {
	t.Parallel()

	defer func() {
		// Assert
		if got := recover(); got == nil {
			t.Errorf("MustDisc(...) recover() = nil, want non-nil")
		}
	}()

	// Act
	disctest.MustDisc("BOGUS", nil)
}

func TestOpenTrackReadsRaw(t *testing.T) {
	t.Parallel()

	// Arrange
	want := sectorData()

	// Act
	track, openErr := disctest.OpenTrack(cue.ModeMode1_2048, want)
	payload, readErr := io.ReadAll(track)

	// Assert
	if got, want := openErr, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("OpenTrack(...) = error %v, want %v", got, want)
	}
	if got, want := readErr, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("io.ReadAll(track) = error %v, want %v", got, want)
	}
	if got, want := track.Number(), 1; !cmp.Equal(got, want) {
		t.Errorf("Track.Number() = %d, want %d", got, want)
	}
	if got, want := payload, want; !cmp.Equal(got, want) {
		t.Errorf("io.ReadAll(track) = mismatch (-want +got):\n%s", cmp.Diff(want, got))
	}
}

func TestMustOpenTrackReadsRaw(t *testing.T) {
	t.Parallel()

	// Arrange
	want := sectorData()

	// Act
	track := disctest.MustOpenTrack(cue.ModeMode1_2048, want)
	payload, readErr := io.ReadAll(track)

	// Assert
	if got, want := readErr, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("io.ReadAll(track) = error %v, want %v", got, want)
	}
	if got, want := payload, want; !cmp.Equal(got, want) {
		t.Errorf("io.ReadAll(track) = mismatch (-want +got):\n%s", cmp.Diff(want, got))
	}
}

func TestOpenTrack_OpenFailure_ReturnsError(t *testing.T) {
	t.Parallel()

	// Arrange
	want := io.ErrUnexpectedEOF

	// Act
	track, err := disctest.OpenTrack(
		cue.ModeMode1_2048,
		sectorData(),
		disc.WithOpenFunc(disctest.ErrOpenFunc(want)),
	)

	// Assert
	if got, want := err, want; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("OpenTrack(...) = error %v, want %v", got, want)
	}
	if got, want := track, (*disc.Track)(nil); !cmp.Equal(got, want) {
		t.Errorf("OpenTrack(...) = %v, want %v", got, want)
	}
}

func TestMustOpenTrack_OpenFailure_Panics(t *testing.T) {
	t.Parallel()

	defer func() {
		// Assert
		if got := recover(); got == nil {
			t.Errorf("MustOpenTrack(...) recover() = nil, want non-nil")
		}
	}()

	// Act
	disctest.MustOpenTrack(
		cue.ModeMode1_2048,
		sectorData(),
		disc.WithOpenFunc(disctest.ErrOpenFunc(io.ErrUnexpectedEOF)),
	)
}
