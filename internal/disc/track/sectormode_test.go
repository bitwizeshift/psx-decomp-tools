package track_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track"
)

func TestSectorModeString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		mode track.SectorMode
		want string
	}{
		{
			name: "Blank",
			mode: track.SectorModeBlank,
			want: "Blank",
		},
		{
			name: "Mode1",
			mode: track.SectorModeMode1,
			want: "Mode1",
		},
		{
			name: "Mode2",
			mode: track.SectorModeMode2,
			want: "Mode2",
		},
		{
			name: "Unknown",
			mode: track.SectorMode(99),
			want: "SectorMode(99)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange

			// Act
			text := tc.mode.String()

			// Assert
			if got, want := text, tc.want; !cmp.Equal(got, want) {
				t.Errorf("SectorMode.String() = %q, want %q", got, want)
			}
		})
	}
}
