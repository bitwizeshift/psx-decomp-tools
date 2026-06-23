package subheader_test

import (
	"bytes"
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/subheader"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestRead(t *testing.T) {
	t.Parallel()

	// Arrange
	data := []byte{0x00, 0x01, 0x64, 0x01, 0x00, 0x02, 0x00, 0x00}
	reader := bytes.NewReader(data)

	// Act
	got, err := subheader.Read(reader)

	// Assert
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	want := []subheader.Subheader{
		{File: 0x00, Channel: 0x01, SubMode: 0x64, Coding: 0x01},
		{File: 0x00, Channel: 0x02, SubMode: 0x00, Coding: 0x00},
	}
	if got, want := got, want; !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
		t.Errorf("Read() diff (-got +want):\n%s", cmp.Diff(got, want, cmpopts.EquateEmpty()))
	}
}

func TestParse(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		data []byte
		want []subheader.Subheader
	}{
		{
			name: "Empty",
			data: nil,
			want: []subheader.Subheader{},
		}, {
			name: "TwoSectors",
			data: []byte{0x00, 0x01, 0x64, 0x01, 0x00, 0x02, 0x00, 0x00},
			want: []subheader.Subheader{
				{File: 0x00, Channel: 0x01, SubMode: 0x64, Coding: 0x01},
				{File: 0x00, Channel: 0x02, SubMode: 0x00, Coding: 0x00},
			},
		}, {
			name: "IgnoresTrailingPartial",
			data: []byte{0x00, 0x01, 0x64, 0x01, 0xff},
			want: []subheader.Subheader{
				{File: 0x00, Channel: 0x01, SubMode: 0x64, Coding: 0x01},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange

			// Act
			got := subheader.Parse(tc.data)

			// Assert
			if got, want := got, tc.want; !cmp.Equal(got, want) {
				t.Errorf("Parse() diff (-got +want):\n%s", cmp.Diff(got, want))
			}
		})
	}
}

func TestSubheaderClassification(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		submode      byte
		wantForm2    bool
		wantAudio    bool
		wantVideo    bool
		wantData     bool
		wantRealTime bool
		wantEOF      bool
		wantDataSize int
	}{
		{
			name:         "Form1Filler",
			submode:      0x00,
			wantData:     true,
			wantDataSize: 2048,
		}, {
			name:         "Form1RealTimeVideo",
			submode:      0x48,
			wantVideo:    true,
			wantRealTime: true,
			wantDataSize: 2048,
		}, {
			name:         "Form2AudioRealTime",
			submode:      0x64,
			wantForm2:    true,
			wantAudio:    true,
			wantRealTime: true,
			wantDataSize: 2324,
		}, {
			name:         "Form2AudioEndOfFile",
			submode:      0xe4,
			wantForm2:    true,
			wantAudio:    true,
			wantRealTime: true,
			wantEOF:      true,
			wantDataSize: 2324,
		}, {
			name:         "Form1EndOfFile",
			submode:      0x80,
			wantData:     true,
			wantEOF:      true,
			wantDataSize: 2048,
		}, {
			name:         "Form2Data",
			submode:      0x20,
			wantForm2:    true,
			wantDataSize: 2324,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := subheader.Subheader{
				SubMode: tc.submode,
			}

			// Act
			form2 := sut.Form2()
			audio := sut.Audio()
			video := sut.Video()
			data := sut.Data()
			realTime := sut.RealTime()
			eof := sut.EndOfFile()
			dataSize := sut.UserDataSize()

			// Assert
			if got, want := form2, tc.wantForm2; !cmp.Equal(got, want) {
				t.Errorf("Form2() = %v, want %v", got, want)
			}
			if got, want := audio, tc.wantAudio; !cmp.Equal(got, want) {
				t.Errorf("Audio() = %v, want %v", got, want)
			}
			if got, want := video, tc.wantVideo; !cmp.Equal(got, want) {
				t.Errorf("Video() = %v, want %v", got, want)
			}
			if got, want := data, tc.wantData; !cmp.Equal(got, want) {
				t.Errorf("Data() = %v, want %v", got, want)
			}
			if got, want := realTime, tc.wantRealTime; !cmp.Equal(got, want) {
				t.Errorf("RealTime() = %v, want %v", got, want)
			}
			if got, want := eof, tc.wantEOF; !cmp.Equal(got, want) {
				t.Errorf("EndOfFile() = %v, want %v", got, want)
			}
			if got, want := dataSize, tc.wantDataSize; !cmp.Equal(got, want) {
				t.Errorf("UserDataSize() = %d, want %d", got, want)
			}
		})
	}
}
