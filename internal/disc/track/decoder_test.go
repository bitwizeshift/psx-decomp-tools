package track_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track"
)

// stubDecoder is a SectorDecoder double that returns a fixed sector and error.
type stubDecoder struct {
	sector track.Sector
	err    error
}

func (d stubDecoder) DecodeSector(track.RawSector) (track.Sector, error) {
	return d.sector, d.err
}

func TestLenientSectorDecoderDecodeSector(t *testing.T) {
	t.Parallel()

	sample := track.Sector{Number: 1, UserData: []byte{0xAB}, Raw: []byte{0xAB}}

	testCases := []struct {
		name         string
		decodeErr    error
		allowed      []error
		want         track.Sector
		wantErr      error
		wantRecorded []error
	}{
		{
			name:         "AllowedSwallowed",
			decodeErr:    track.ErrChecksum,
			allowed:      []error{track.ErrChecksum},
			want:         sample,
			wantRecorded: []error{track.ErrChecksum},
		},
		{
			name:      "NotAllowedPassThrough",
			decodeErr: track.ErrBadSync,
			allowed:   []error{track.ErrChecksum},
			want:      sample,
			wantErr:   track.ErrBadSync,
		},
		{
			name:      "NoError",
			decodeErr: nil,
			allowed:   []error{track.ErrChecksum},
			want:      sample,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			var recorded []error
			sut := track.LenientSectorDecoder{
				Decoder:  stubDecoder{sector: sample, err: tc.decodeErr},
				Callback: func(err error) { recorded = append(recorded, err) },
				Allowed:  tc.allowed,
			}
			opts := cmp.Options{cmpopts.EquateErrors(), cmpopts.EquateEmpty()}

			// Act
			sector, err := sut.DecodeSector(track.RawSector{})

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("LenientSectorDecoder.DecodeSector(...) = error %v, want %v", got, want)
			}
			if got, want := recorded, tc.wantRecorded; !cmp.Equal(got, want, opts) {
				t.Errorf("LenientSectorDecoder.DecodeSector(...) recorded = %v, want %v", got, want)
			}
			if got, want := sector, tc.want; !cmp.Equal(got, want, opts) {
				t.Errorf("LenientSectorDecoder.DecodeSector(...) = mismatch (-want +got):\n%s", cmp.Diff(want, got, opts))
			}
		})
	}
}

func TestLenientSectorDecoderNilCallback(t *testing.T) {
	t.Parallel()

	// Arrange
	sample := track.Sector{Number: 2, Raw: []byte{0x01}}
	sut := track.LenientSectorDecoder{
		Decoder: stubDecoder{sector: sample, err: track.ErrChecksum},
		Allowed: []error{track.ErrChecksum},
	}
	opts := cmp.Options{cmpopts.EquateEmpty()}

	// Act
	sector, err := sut.DecodeSector(track.RawSector{})

	// Assert
	if got, want := err, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("LenientSectorDecoder.DecodeSector(...) = error %v, want %v", got, want)
	}
	if got, want := sector, sample; !cmp.Equal(got, want, opts) {
		t.Errorf("LenientSectorDecoder.DecodeSector(...) = mismatch (-want +got):\n%s", cmp.Diff(want, got, opts))
	}
}
