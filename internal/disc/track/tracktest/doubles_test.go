package tracktest_test

import (
	"errors"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track/tracktest"
)

var errStub = errors.New("stub")

func TestErrSectorReaderReadSector(t *testing.T) {
	t.Parallel()

	// Arrange
	reader := tracktest.ErrSectorReader(errStub)

	// Act
	sector, err := reader.ReadSector(0)

	// Assert
	if got, want := err, errStub; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("ErrSectorReader(...).ReadSector(0) = error %v, want %v", got, want)
	}
	if got, want := sector, (track.RawSector{}); !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
		t.Errorf("ErrSectorReader(...).ReadSector(0) = %v, want %v", got, want)
	}
}

func TestStaticSectorReaderReadSector(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		sector  int
		want    track.RawSector
		wantErr error
	}{
		{
			name:   "First",
			sector: 0,
			want:   track.RawSector{Number: 0, Data: []byte{0xA0}},
		},
		{
			name:   "Second",
			sector: 1,
			want:   track.RawSector{Number: 1, Data: []byte{0xB1}},
		},
		{
			name:    "PastEnd",
			sector:  2,
			want:    track.RawSector{},
			wantErr: io.EOF,
		},
		{
			name:    "Negative",
			sector:  -1,
			want:    track.RawSector{},
			wantErr: io.EOF,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			reader := tracktest.StaticSectorReader([]byte{0xA0}, []byte{0xB1})

			// Act
			sector, err := reader.ReadSector(tc.sector)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("StaticSectorReader(...).ReadSector(%d) = error %v, want %v", tc.sector, got, want)
			}
			if got, want := sector, tc.want; !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
				t.Errorf("StaticSectorReader(...).ReadSector(%d) = %v, want %v", tc.sector, got, want)
			}
		})
	}
}

func TestErrVerifierVerify(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		err     error
		wantErr error
	}{
		{
			name:    "Error",
			err:     errStub,
			wantErr: errStub,
		},
		{
			name: "NilAccepts",
			err:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			verifier := tracktest.ErrVerifier(tc.err)

			// Act
			err := verifier.Verify(track.Sector{})

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("ErrVerifier(%v).Verify(...) = error %v, want %v", tc.err, got, want)
			}
		})
	}
}
