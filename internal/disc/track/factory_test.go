package track_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/cue"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track"
)

func TestNewDispatchFactoryDecoderFor(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		mode cue.Mode
		want track.SectorDecoder
	}{
		{
			name: "Audio",
			mode: cue.ModeAudio,
			want: track.AudioDecoder{},
		},
		{
			name: "CDG",
			mode: cue.ModeCDG,
			want: track.AudioDecoder{},
		},
		{
			name: "Mode1_2048",
			mode: cue.ModeMode1_2048,
			want: track.Mode1Decoder{SectorSize: 2048, Verifier: track.Mode1Verifier{}},
		},
		{
			name: "Mode1_2352",
			mode: cue.ModeMode1_2352,
			want: track.Mode1Decoder{SectorSize: 2352, Verifier: track.Mode1Verifier{}},
		},
		{
			name: "Mode2_2336",
			mode: cue.ModeMode2_2336,
			want: track.Mode2Decoder{SectorSize: 2336, Verifier: track.Mode2Verifier{}},
		},
		{
			name: "Mode2_2352",
			mode: cue.ModeMode2_2352,
			want: track.Mode2Decoder{SectorSize: 2352, Verifier: track.Mode2Verifier{}},
		},
		{
			name: "CDI_2336",
			mode: cue.ModeCDI_2336,
			want: track.Mode2Decoder{SectorSize: 2336, Verifier: track.Mode2Verifier{}},
		},
		{
			name: "CDI_2352",
			mode: cue.ModeCDI_2352,
			want: track.Mode2Decoder{SectorSize: 2352, Verifier: track.Mode2Verifier{}},
		},
		{
			name: "UnknownFallsBackToRaw",
			mode: cue.Mode(99),
			want: track.RawDecoder{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			factory := track.NewDispatchFactory()
			tr := cue.Track{Mode: tc.mode}

			// Act
			decoder := factory.DecoderFor(tr)

			// Assert
			if got, want := decoder, tc.want; !cmp.Equal(got, want) {
				t.Errorf("DispatchFactory.DecoderFor(%v) = %#v, want %#v", tc.mode, got, want)
			}
		})
	}
}
