package cue_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/cue"
)

func TestMSFFrameCount(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		msf  cue.MSF
		want int
	}{
		{
			name: "Zero",
			msf:  cue.MSF{},
			want: 0,
		},
		{
			name: "FramesOnly",
			msf:  cue.MSF{Frame: 74},
			want: 74,
		},
		{
			name: "OneSecond",
			msf:  cue.MSF{Second: 1},
			want: 75,
		},
		{
			name: "OneMinute",
			msf:  cue.MSF{Minute: 1},
			want: 4500,
		},
		{
			name: "Combined",
			msf:  cue.MSF{Minute: 2, Second: 32, Frame: 12},
			want: 11412,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange

			// Act
			frames := tc.msf.FrameCount()

			// Assert
			if got, want := frames, tc.want; !cmp.Equal(got, want) {
				t.Errorf("MSF.FrameCount() = %d, want %d", got, want)
			}
		})
	}
}

func TestMSFString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		msf  cue.MSF
		want string
	}{
		{
			name: "Zero",
			msf:  cue.MSF{},
			want: "00:00:00",
		},
		{
			name: "Padded",
			msf:  cue.MSF{Minute: 2, Second: 5, Frame: 9},
			want: "02:05:09",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange

			// Act
			text := tc.msf.String()

			// Assert
			if got, want := text, tc.want; !cmp.Equal(got, want) {
				t.Errorf("MSF.String() = %q, want %q", got, want)
			}
		})
	}
}

func TestMSFMarshalText(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		msf  cue.MSF
		want string
	}{
		{
			name: "Zero",
			msf:  cue.MSF{},
			want: "00:00:00",
		},
		{
			name: "Combined",
			msf:  cue.MSF{Minute: 2, Second: 32, Frame: 12},
			want: "02:32:12",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange

			// Act
			text, err := tc.msf.MarshalText()

			// Assert
			if got, want := err, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("MSF.MarshalText() = error %v, want %v", got, want)
			}
			if got, want := string(text), tc.want; !cmp.Equal(got, want) {
				t.Errorf("MSF.MarshalText() = %q, want %q", got, want)
			}
		})
	}
}

func TestMSFUnmarshalText(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		text    string
		want    cue.MSF
		wantErr error
	}{
		{
			name: "Zero",
			text: "00:00:00",
			want: cue.MSF{},
		},
		{
			name: "MaximumFields",
			text: "99:59:74",
			want: cue.MSF{Minute: 99, Second: 59, Frame: 74},
		},
		{
			name:    "TooFewFields",
			text:    "00:00",
			wantErr: cue.ErrInvalidMSF,
		},
		{
			name:    "TooManyFields",
			text:    "00:00:00:00",
			wantErr: cue.ErrInvalidMSF,
		},
		{
			name:    "NonNumericField",
			text:    "00:xx:00",
			wantErr: cue.ErrInvalidMSF,
		},
		{
			name:    "NegativeField",
			text:    "00:-1:00",
			wantErr: cue.ErrInvalidMSF,
		},
		{
			name:    "SecondsOutOfRange",
			text:    "00:60:00",
			wantErr: cue.ErrInvalidMSF,
		},
		{
			name:    "FramesOutOfRange",
			text:    "00:00:75",
			wantErr: cue.ErrInvalidMSF,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			var msf cue.MSF

			// Act
			err := msf.UnmarshalText([]byte(tc.text))

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("MSF.UnmarshalText(%q) = error %v, want %v", tc.text, got, want)
			}
			if got, want := msf, tc.want; !cmp.Equal(got, want) {
				t.Errorf("MSF.UnmarshalText(%q) = %v, want %v", tc.text, got, want)
			}
		})
	}
}
