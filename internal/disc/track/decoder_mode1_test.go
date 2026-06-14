package track_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track/tracktest"
)

func TestMode1DecoderDecodeSector(t *testing.T) {
	t.Parallel()

	const number = 7
	valid := tracktest.BuildMode1Sector(testMSF, payload(2048))
	header := &track.SectorHeader{Minute: 1, Second: 2, Frame: 3, Mode: track.SectorModeMode1}

	testCases := []struct {
		name       string
		sectorSize int
		verifier   track.SectorVerifier
		raw        []byte
		want       track.Sector
		wantErr    error
	}{
		{
			name:       "FullValidNilVerifier",
			sectorSize: 2352,
			verifier:   nil,
			raw:        valid,
			want: track.Sector{
				Number:   number,
				Header:   header,
				UserData: payload(2048),
				Raw:      valid,
			},
		},
		{
			name:       "ChecksumError",
			sectorSize: 2352,
			verifier:   tracktest.ErrVerifier(track.ErrChecksum),
			raw:        valid,
			want: track.Sector{
				Number:   number,
				Header:   header,
				UserData: payload(2048),
				Raw:      valid,
			},
			wantErr: track.ErrChecksum,
		},
		{
			name:       "Bare",
			sectorSize: 2048,
			raw:        payload(2048),
			want: track.Sector{
				Number:   number,
				UserData: payload(2048),
				Raw:      payload(2048),
			},
		},
		{
			name:       "Short",
			sectorSize: 2352,
			raw:        payload(100),
			want: track.Sector{
				Number: number,
				Raw:    payload(100),
			},
			wantErr: track.ErrShortSector,
		},
		{
			name:       "BadSync",
			sectorSize: 2352,
			raw:        corrupt(valid, 0),
			want: track.Sector{
				Number:   number,
				UserData: payload(2048),
				Raw:      corrupt(valid, 0),
			},
			wantErr: track.ErrBadSync,
		},
		{
			name:       "BadBCDMinute",
			sectorSize: 2352,
			raw:        corrupt(valid, 12),
			want: track.Sector{
				Number:   number,
				UserData: payload(2048),
				Raw:      corrupt(valid, 12),
			},
			wantErr: track.ErrInvalidBCD,
		},
		{
			name:       "BadBCDSecond",
			sectorSize: 2352,
			raw:        corrupt(valid, 13),
			want: track.Sector{
				Number:   number,
				UserData: payload(2048),
				Raw:      corrupt(valid, 13),
			},
			wantErr: track.ErrInvalidBCD,
		},
		{
			name:       "BadBCDFrame",
			sectorSize: 2352,
			raw:        corrupt(valid, 14),
			want: track.Sector{
				Number:   number,
				UserData: payload(2048),
				Raw:      corrupt(valid, 14),
			},
			wantErr: track.ErrInvalidBCD,
		},
		{
			name:       "BadMode",
			sectorSize: 2352,
			raw:        corrupt(valid, 15),
			want: track.Sector{
				Number:   number,
				UserData: payload(2048),
				Raw:      corrupt(valid, 15),
			},
			wantErr: track.ErrInvalidSectorMode,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := track.Mode1Decoder{
				SectorSize: tc.sectorSize,
				Verifier:   tc.verifier,
			}
			raw := track.RawSector{
				Number: number,
				Data:   tc.raw,
			}
			opts := cmp.Options{cmpopts.EquateEmpty()}

			// Act
			sector, err := sut.DecodeSector(raw)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Mode1Decoder.DecodeSector(...) = error %v, want %v", got, want)
			}
			if got, want := sector, tc.want; !cmp.Equal(got, want, opts) {
				t.Errorf("Mode1Decoder.DecodeSector(...) = mismatch (-want +got):\n%s", cmp.Diff(want, got, opts))
			}
		})
	}
}
