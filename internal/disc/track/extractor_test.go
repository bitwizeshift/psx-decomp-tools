package track_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/cue"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track"
)

func TestPayloadExtractorPayload(t *testing.T) {
	t.Parallel()

	userData := []byte{0x01, 0x02, 0x03}
	raw := []byte{0xF0, 0xF1, 0xF2, 0xF3}
	sector := track.Sector{UserData: userData, Raw: raw}

	testCases := []struct {
		name      string
		extractor track.PayloadExtractor
		want      []byte
	}{
		{
			name:      "Mode1",
			extractor: track.Mode1Extractor{},
			want:      userData,
		},
		{
			name:      "Mode2",
			extractor: track.Mode2Extractor{},
			want:      userData,
		},
		{
			name:      "XA",
			extractor: track.XAExtractor{},
			want:      userData,
		},
		{
			name:      "Raw",
			extractor: track.RawExtractor{},
			want:      raw,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange

			// Act
			data, err := tc.extractor.Payload(sector)

			// Assert
			if got, want := err, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("PayloadExtractor.Payload(...) = error %v, want %v", got, want)
			}
			if got, want := data, tc.want; !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
				t.Errorf("PayloadExtractor.Payload(...) = %v, want %v", got, want)
			}
		})
	}
}

func TestNewDispatchExtractorFactoryExtractorFor(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		mode cue.Mode
		want track.PayloadExtractor
	}{
		{
			name: "Audio",
			mode: cue.ModeAudio,
			want: track.RawExtractor{},
		},
		{
			name: "CDG",
			mode: cue.ModeCDG,
			want: track.RawExtractor{},
		},
		{
			name: "Mode1_2048",
			mode: cue.ModeMode1_2048,
			want: track.Mode1Extractor{},
		},
		{
			name: "Mode1_2352",
			mode: cue.ModeMode1_2352,
			want: track.Mode1Extractor{},
		},
		{
			name: "Mode2_2336",
			mode: cue.ModeMode2_2336,
			want: track.Mode2Extractor{},
		},
		{
			name: "Mode2_2352",
			mode: cue.ModeMode2_2352,
			want: track.XAExtractor{},
		},
		{
			name: "CDI_2336",
			mode: cue.ModeCDI_2336,
			want: track.Mode2Extractor{},
		},
		{
			name: "CDI_2352",
			mode: cue.ModeCDI_2352,
			want: track.XAExtractor{},
		},
		{
			name: "UnknownFallsBackToRaw",
			mode: cue.Mode(99),
			want: track.RawExtractor{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			factory := track.NewDispatchExtractorFactory()
			tr := cue.Track{Mode: tc.mode}

			// Act
			extractor := factory.ExtractorFor(tr)

			// Assert
			if got, want := extractor, tc.want; !cmp.Equal(got, want) {
				t.Errorf("DispatchExtractorFactory.ExtractorFor(%v) = %#v, want %#v", tc.mode, got, want)
			}
		})
	}
}
