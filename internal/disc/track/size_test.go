package track_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/cue"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track"
)

func TestSectorSize(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		mode    cue.Mode
		want    int
		wantErr error
	}{
		{
			name: "Audio",
			mode: cue.ModeAudio,
			want: 2352,
		},
		{
			name: "CDG",
			mode: cue.ModeCDG,
			want: 2352,
		},
		{
			name: "Mode1_2048",
			mode: cue.ModeMode1_2048,
			want: 2048,
		},
		{
			name: "Mode1_2352",
			mode: cue.ModeMode1_2352,
			want: 2352,
		},
		{
			name: "Mode2_2336",
			mode: cue.ModeMode2_2336,
			want: 2336,
		},
		{
			name: "Mode2_2352",
			mode: cue.ModeMode2_2352,
			want: 2352,
		},
		{
			name: "CDI_2336",
			mode: cue.ModeCDI_2336,
			want: 2336,
		},
		{
			name: "CDI_2352",
			mode: cue.ModeCDI_2352,
			want: 2352,
		},
		{
			name:    "Unsupported",
			mode:    cue.Mode(99),
			want:    0,
			wantErr: track.ErrUnsupportedMode,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange

			// Act
			size, err := track.SectorSize(tc.mode)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("SectorSize(%v) = error %v, want %v", tc.mode, got, want)
			}
			if got, want := size, tc.want; !cmp.Equal(got, want) {
				t.Errorf("SectorSize(%v) = %d, want %d", tc.mode, got, want)
			}
		})
	}
}
