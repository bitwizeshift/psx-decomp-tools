package xa_test

import (
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/media/xa"
	"github.com/google/go-cmp/cmp"
)

func TestCoding(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		coding         xa.Coding
		wantStereo     bool
		wantChannels   int
		wantSampleRate int
	}{
		{
			name:           "MonoFullRate",
			coding:         0x00,
			wantStereo:     false,
			wantChannels:   1,
			wantSampleRate: 37800,
		}, {
			name:           "StereoFullRate",
			coding:         0x01,
			wantStereo:     true,
			wantChannels:   2,
			wantSampleRate: 37800,
		}, {
			name:           "MonoHalfRate",
			coding:         0x04,
			wantStereo:     false,
			wantChannels:   1,
			wantSampleRate: 18900,
		}, {
			name:           "StereoHalfRate",
			coding:         0x05,
			wantStereo:     true,
			wantChannels:   2,
			wantSampleRate: 18900,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			coding := tc.coding

			// Act
			stereo := coding.Stereo()
			channels := coding.Channels()
			sampleRate := coding.SampleRate()

			// Assert
			if got, want := stereo, tc.wantStereo; !cmp.Equal(got, want) {
				t.Errorf("Stereo() = %v, want %v", got, want)
			}
			if got, want := channels, tc.wantChannels; !cmp.Equal(got, want) {
				t.Errorf("Channels() = %v, want %v", got, want)
			}
			if got, want := sampleRate, tc.wantSampleRate; !cmp.Equal(got, want) {
				t.Errorf("SampleRate() = %v, want %v", got, want)
			}
		})
	}
}
