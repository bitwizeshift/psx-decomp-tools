package cue_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/cue"
)

func TestFlagString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		flag cue.Flag
		want string
	}{
		{
			name: "DCP",
			flag: cue.FlagDCP,
			want: "DCP",
		},
		{
			name: "4CH",
			flag: cue.Flag4CH,
			want: "4CH",
		},
		{
			name: "PRE",
			flag: cue.FlagPRE,
			want: "PRE",
		},
		{
			name: "SCMS",
			flag: cue.FlagSCMS,
			want: "SCMS",
		},
		{
			name: "Unknown",
			flag: cue.Flag(99),
			want: "Flag(99)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange

			// Act
			text := tc.flag.String()

			// Assert
			if got, want := text, tc.want; !cmp.Equal(got, want) {
				t.Errorf("Flag.String() = %q, want %q", got, want)
			}
		})
	}
}

func TestFlagMarshalText(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		flag cue.Flag
		want string
	}{
		{
			name: "DCP",
			flag: cue.FlagDCP,
			want: "DCP",
		},
		{
			name: "4CH",
			flag: cue.Flag4CH,
			want: "4CH",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange

			// Act
			text, err := tc.flag.MarshalText()

			// Assert
			if got, want := err, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Flag.MarshalText() = error %v, want %v", got, want)
			}
			if got, want := string(text), tc.want; !cmp.Equal(got, want) {
				t.Errorf("Flag.MarshalText() = %q, want %q", got, want)
			}
		})
	}
}

func TestFlagUnmarshalText(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		text    string
		want    cue.Flag
		wantErr error
	}{
		{
			name: "DCP",
			text: "DCP",
			want: cue.FlagDCP,
		},
		{
			name: "4CH",
			text: "4CH",
			want: cue.Flag4CH,
		},
		{
			name: "PRE",
			text: "PRE",
			want: cue.FlagPRE,
		},
		{
			name: "SCMS",
			text: "SCMS",
			want: cue.FlagSCMS,
		},
		{
			name: "CaseInsensitive",
			text: "scms",
			want: cue.FlagSCMS,
		},
		{
			name:    "Invalid",
			text:    "NONSENSE",
			want:    cue.FlagDCP,
			wantErr: cue.ErrInvalidFlag,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			var flag cue.Flag

			// Act
			err := flag.UnmarshalText([]byte(tc.text))

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Flag.UnmarshalText(%q) = error %v, want %v", tc.text, got, want)
			}
			if got, want := flag, tc.want; !cmp.Equal(got, want) {
				t.Errorf("Flag.UnmarshalText(%q) = %v, want %v", tc.text, got, want)
			}
		})
	}
}
