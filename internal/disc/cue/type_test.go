package cue_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/cue"
)

func TestTypeString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		fileType cue.Type
		want     string
	}{
		{
			name: "Binary",
			fileType: cue.TypeBinary,
			want: "BINARY",
		},
		{
			name: "Motorola",
			fileType: cue.TypeMotorola,
			want: "MOTOROLA",
		},
		{
			name: "AIFF",
			fileType: cue.TypeAIFF,
			want: "AIFF",
		},
		{
			name: "Wave",
			fileType: cue.TypeWave,
			want: "WAVE",
		},
		{
			name: "MP3",
			fileType: cue.TypeMP3,
			want: "MP3",
		},
		{
			name: "Unknown",
			fileType: cue.Type(99),
			want: "Type(99)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange

			// Act
			text := tc.fileType.String()

			// Assert
			if got, want := text, tc.want; !cmp.Equal(got, want) {
				t.Errorf("Type.String() = %q, want %q", got, want)
			}
		})
	}
}

func TestTypeMarshalText(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		fileType cue.Type
		want     string
	}{
		{
			name: "Binary",
			fileType: cue.TypeBinary,
			want: "BINARY",
		},
		{
			name: "Wave",
			fileType: cue.TypeWave,
			want: "WAVE",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange

			// Act
			text, err := tc.fileType.MarshalText()

			// Assert
			if got, want := err, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Type.MarshalText() = error %v, want %v", got, want)
			}
			if got, want := string(text), tc.want; !cmp.Equal(got, want) {
				t.Errorf("Type.MarshalText() = %q, want %q", got, want)
			}
		})
	}
}

func TestTypeUnmarshalText(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		text    string
		want    cue.Type
		wantErr error
	}{
		{
			name: "Binary",
			text: "BINARY",
			want: cue.TypeBinary,
		},
		{
			name: "Motorola",
			text: "MOTOROLA",
			want: cue.TypeMotorola,
		},
		{
			name: "AIFF",
			text: "AIFF",
			want: cue.TypeAIFF,
		},
		{
			name: "Wave",
			text: "WAVE",
			want: cue.TypeWave,
		},
		{
			name: "MP3",
			text: "MP3",
			want: cue.TypeMP3,
		},
		{
			name: "CaseInsensitive",
			text: "binary",
			want: cue.TypeBinary,
		},
		{
			name: "Invalid",
			text: "NONSENSE",
			want: cue.TypeBinary,
			wantErr: cue.ErrInvalidType,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			var fileType cue.Type

			// Act
			err := fileType.UnmarshalText([]byte(tc.text))

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Type.UnmarshalText(%q) = error %v, want %v", tc.text, got, want)
			}
			if got, want := fileType, tc.want; !cmp.Equal(got, want) {
				t.Errorf("Type.UnmarshalText(%q) = %v, want %v", tc.text, got, want)
			}
		})
	}
}
