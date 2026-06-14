package cue_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/cue"
)

func TestModeString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		mode cue.Mode
		want string
	}{
		{
			name: "Audio",
			mode: cue.ModeAudio,
			want: "AUDIO",
		},
		{
			name: "CDG",
			mode: cue.ModeCDG,
			want: "CDG",
		},
		{
			name: "Mode1_2048",
			mode: cue.ModeMode1_2048,
			want: "MODE1/2048",
		},
		{
			name: "Mode1_2352",
			mode: cue.ModeMode1_2352,
			want: "MODE1/2352",
		},
		{
			name: "Mode2_2336",
			mode: cue.ModeMode2_2336,
			want: "MODE2/2336",
		},
		{
			name: "Mode2_2352",
			mode: cue.ModeMode2_2352,
			want: "MODE2/2352",
		},
		{
			name: "CDI_2336",
			mode: cue.ModeCDI_2336,
			want: "CDI/2336",
		},
		{
			name: "CDI_2352",
			mode: cue.ModeCDI_2352,
			want: "CDI/2352",
		},
		{
			name: "Unknown",
			mode: cue.Mode(99),
			want: "Mode(99)",
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
				t.Errorf("Mode.String() = %q, want %q", got, want)
			}
		})
	}
}

func TestModeMarshalText(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		mode cue.Mode
		want string
	}{
		{
			name: "Audio",
			mode: cue.ModeAudio,
			want: "AUDIO",
		},
		{
			name: "Mode2_2352",
			mode: cue.ModeMode2_2352,
			want: "MODE2/2352",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange

			// Act
			text, err := tc.mode.MarshalText()

			// Assert
			if got, want := err, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Mode.MarshalText() = error %v, want %v", got, want)
			}
			if got, want := string(text), tc.want; !cmp.Equal(got, want) {
				t.Errorf("Mode.MarshalText() = %q, want %q", got, want)
			}
		})
	}
}

func TestModeUnmarshalText(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		text    string
		want    cue.Mode
		wantErr error
	}{
		{
			name: "Audio",
			text: "AUDIO",
			want: cue.ModeAudio,
		},
		{
			name: "CDG",
			text: "CDG",
			want: cue.ModeCDG,
		},
		{
			name: "Mode1_2048",
			text: "MODE1/2048",
			want: cue.ModeMode1_2048,
		},
		{
			name: "Mode1_2352",
			text: "MODE1/2352",
			want: cue.ModeMode1_2352,
		},
		{
			name: "Mode2_2336",
			text: "MODE2/2336",
			want: cue.ModeMode2_2336,
		},
		{
			name: "Mode2_2352",
			text: "MODE2/2352",
			want: cue.ModeMode2_2352,
		},
		{
			name: "CDI_2336",
			text: "CDI/2336",
			want: cue.ModeCDI_2336,
		},
		{
			name: "CDI_2352",
			text: "CDI/2352",
			want: cue.ModeCDI_2352,
		},
		{
			name: "CaseInsensitive",
			text: "mode2/2352",
			want: cue.ModeMode2_2352,
		},
		{
			name: "Invalid",
			text: "NONSENSE",
			want: cue.ModeAudio,
			wantErr: cue.ErrInvalidMode,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			var mode cue.Mode

			// Act
			err := mode.UnmarshalText([]byte(tc.text))

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Mode.UnmarshalText(%q) = error %v, want %v", tc.text, got, want)
			}
			if got, want := mode, tc.want; !cmp.Equal(got, want) {
				t.Errorf("Mode.UnmarshalText(%q) = %v, want %v", tc.text, got, want)
			}
		})
	}
}
