package track_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track/tracktest"
)

// clearForm2EDC returns a copy of an XA Form 2 sector with its optional EDC
// field zeroed.
func clearForm2EDC(sector []byte) []byte {
	out := append([]byte(nil), sector...)
	for i := 2348; i < 2352; i++ {
		out[i] = 0
	}
	return out
}

func TestMode1VerifierVerify(t *testing.T) {
	t.Parallel()

	valid := tracktest.BuildMode1Sector(testMSF, payload(2048))

	testCases := []struct {
		name    string
		raw     []byte
		wantErr error
	}{
		{
			name: "Valid",
			raw:  valid,
		},
		{
			name:    "Corrupt",
			raw:     corrupt(valid, 100),
			wantErr: track.ErrChecksum,
		},
		{
			name:    "Short",
			raw:     payload(100),
			wantErr: track.ErrShortSector,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sector := track.Sector{Raw: tc.raw}

			// Act
			err := track.Mode1Verifier{}.Verify(sector)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Mode1Verifier.Verify(...) = error %v, want %v", got, want)
			}
		})
	}
}

func TestMode2VerifierVerify(t *testing.T) {
	t.Parallel()

	form1 := tracktest.BuildMode2Form1Sector(track.Subheader{}, testMSF, payload(2048))
	form2 := tracktest.BuildMode2Form2Sector(track.Subheader{}, testMSF, payload(2324))

	testCases := []struct {
		name      string
		raw       []byte
		subheader *track.Subheader
		wantErr   error
	}{
		{
			name:      "Form1Valid",
			raw:       form1,
			subheader: &track.Subheader{Form: track.FormOne},
		},
		{
			name:      "Form1Corrupt",
			raw:       corrupt(form1, 100),
			subheader: &track.Subheader{Form: track.FormOne},
			wantErr:   track.ErrChecksum,
		},
		{
			name:      "Form2Valid",
			raw:       form2,
			subheader: &track.Subheader{Form: track.FormTwo},
		},
		{
			name:      "Form2ZeroEDC",
			raw:       clearForm2EDC(form2),
			subheader: &track.Subheader{Form: track.FormTwo},
		},
		{
			name:      "Form2Corrupt",
			raw:       corrupt(form2, 100),
			subheader: &track.Subheader{Form: track.FormTwo},
			wantErr:   track.ErrChecksum,
		},
		{
			name:      "NilSubheaderUsesForm1",
			raw:       form1,
			subheader: nil,
		},
		{
			name:      "Short",
			raw:       payload(100),
			subheader: &track.Subheader{Form: track.FormOne},
			wantErr:   track.ErrShortSector,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sector := track.Sector{Raw: tc.raw, Subheader: tc.subheader}

			// Act
			err := track.Mode2Verifier{}.Verify(sector)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Mode2Verifier.Verify(...) = error %v, want %v", got, want)
			}
		})
	}
}

func TestNopVerifierVerify(t *testing.T) {
	t.Parallel()

	// Arrange
	sector := track.Sector{Raw: payload(10)}

	// Act
	err := track.NopVerifier{}.Verify(sector)

	// Assert
	if got, want := err, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("NopVerifier.Verify(...) = error %v, want %v", got, want)
	}
}
