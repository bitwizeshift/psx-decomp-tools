package track_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track/tracktest"
)

func TestMode2DecoderDecodeSector(t *testing.T) {
	t.Parallel()

	const number = 9
	sub := track.Subheader{File: 0x11, Channel: 0x22, SubMode: 0x08, Coding: 0x44}
	form1 := tracktest.BuildMode2Form1Sector(sub, testMSF, payload(2048))
	form2 := tracktest.BuildMode2Form2Sector(sub, testMSF, payload(2324))

	header := &track.SectorHeader{Minute: 1, Second: 2, Frame: 3, Mode: track.SectorModeMode2}
	form1Sub := &track.Subheader{File: 0x11, Channel: 0x22, SubMode: 0x08, Coding: 0x44, Form: track.FormOne}
	form2Sub := &track.Subheader{File: 0x11, Channel: 0x22, SubMode: 0x28, Coding: 0x44, Form: track.FormTwo}

	testCases := []struct {
		name       string
		sectorSize int
		verifier   track.SectorVerifier
		raw        []byte
		want       track.Sector
		wantErr    error
	}{
		{
			name:       "Form1Valid",
			sectorSize: 2352,
			raw:        form1,
			want: track.Sector{
				Number:    number,
				Header:    header,
				Subheader: form1Sub,
				UserData:  payload(2048),
				Raw:       form1,
			},
		},
		{
			name:       "Form2Valid",
			sectorSize: 2352,
			raw:        form2,
			want: track.Sector{
				Number:    number,
				Header:    header,
				Subheader: form2Sub,
				UserData:  payload(2324),
				Raw:       form2,
			},
		},
		{
			name:       "ChecksumError",
			sectorSize: 2352,
			verifier:   tracktest.ErrVerifier(track.ErrChecksum),
			raw:        form1,
			want: track.Sector{
				Number:    number,
				Header:    header,
				Subheader: form1Sub,
				UserData:  payload(2048),
				Raw:       form1,
			},
			wantErr: track.ErrChecksum,
		},
		{
			name:       "Bare",
			sectorSize: 2336,
			raw:        payload(2336),
			want: track.Sector{
				Number:   number,
				UserData: payload(2336),
				Raw:      payload(2336),
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
			raw:        corrupt(form1, 0),
			want: track.Sector{
				Number:    number,
				Subheader: form1Sub,
				UserData:  payload(2048),
				Raw:       corrupt(form1, 0),
			},
			wantErr: track.ErrBadSync,
		},
		{
			name:       "BadBCD",
			sectorSize: 2352,
			raw:        corrupt(form1, 12),
			want: track.Sector{
				Number:    number,
				Subheader: form1Sub,
				UserData:  payload(2048),
				Raw:       corrupt(form1, 12),
			},
			wantErr: track.ErrInvalidBCD,
		},
		{
			name:       "BadMode",
			sectorSize: 2352,
			raw:        corrupt(form1, 15),
			want: track.Sector{
				Number:    number,
				Subheader: form1Sub,
				UserData:  payload(2048),
				Raw:       corrupt(form1, 15),
			},
			wantErr: track.ErrInvalidSectorMode,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := track.Mode2Decoder{SectorSize: tc.sectorSize, Verifier: tc.verifier}
			raw := track.RawSector{Number: number, Data: tc.raw}
			opts := cmp.Options{cmpopts.EquateEmpty()}

			// Act
			sector, err := sut.DecodeSector(raw)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Mode2Decoder.DecodeSector(...) = error %v, want %v", got, want)
			}
			if got, want := sector, tc.want; !cmp.Equal(got, want, opts) {
				t.Errorf("Mode2Decoder.DecodeSector(...) = mismatch (-want +got):\n%s", cmp.Diff(want, got, opts))
			}
		})
	}
}
