package vab_test

import (
	"bytes"
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/media/vab"
	"github.com/bitwizeshift/psx-decomp-tools/internal/media/vab/vabtest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestDecodeConfig(t *testing.T) {
	t.Parallel()

	valid := vabtest.Bank(
		[]vabtest.Program{{Tones: []vabtest.Tone{{CenterNote: 60, Waveform: 0}}}},
		[][]byte{bytes.Repeat([]byte{0x11}, 16)},
	)
	badReserved := bytes.Clone(valid)
	badReserved[16], badReserved[17] = 0, 0

	testCases := []struct {
		name    string
		data    []byte
		want    vab.Header
		wantErr error
	}{
		{
			name:    "Valid",
			data:    valid,
			want:    vab.Header{Version: 7, Size: len(valid), Programs: 1, Tones: 1, Waveforms: 1},
			wantErr: nil,
		}, {
			name:    "BadMagic",
			data:    bytes.Repeat([]byte{0x00}, 32),
			want:    vab.Header{},
			wantErr: vab.ErrInvalidHeader,
		}, {
			name:    "BadReserved",
			data:    badReserved,
			want:    vab.Header{},
			wantErr: vab.ErrInvalidHeader,
		}, {
			name:    "Short",
			data:    []byte("pBAV"),
			want:    vab.Header{},
			wantErr: vab.ErrInvalidHeader,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			reader := bytes.NewReader(tc.data)

			// Act
			got, err := vab.DecodeConfig(reader)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("DecodeConfig() error = %v, want %v", got, want)
			}
			if got, want := got, tc.want; !cmp.Equal(got, want) {
				t.Errorf("DecodeConfig() diff (-got +want):\n%s", cmp.Diff(got, want))
			}
		})
	}
}

func TestDecode(t *testing.T) {
	t.Parallel()

	waveform := bytes.Repeat([]byte{0x11}, 16)
	single := vabtest.Bank(
		[]vabtest.Program{{Tones: []vabtest.Tone{{CenterNote: 60, Waveform: 0}}}},
		[][]byte{waveform},
	)
	gapped := vabtest.Bank(
		[]vabtest.Program{{}, {Tones: []vabtest.Tone{{CenterNote: 48, Waveform: 0}}}},
		[][]byte{waveform},
	)

	testCases := []struct {
		name          string
		data          []byte
		wantHeader    vab.Header
		wantPrograms  []vab.Program
		wantWaveforms [][]byte
		wantErr       error
	}{
		{
			name:          "Single",
			data:          single,
			wantHeader:    vab.Header{Version: 7, Size: len(single), Programs: 1, Tones: 1, Waveforms: 1},
			wantPrograms:  []vab.Program{{Tones: []vab.Tone{{CenterNote: 60, Program: 0, Waveform: 0}}}},
			wantWaveforms: [][]byte{waveform},
			wantErr:       nil,
		}, {
			name:          "EmptyLeadingSlot",
			data:          gapped,
			wantHeader:    vab.Header{Version: 7, Size: len(gapped), Programs: 1, Tones: 1, Waveforms: 1},
			wantPrograms:  []vab.Program{{Tones: []vab.Tone{{CenterNote: 48, Program: 1, Waveform: 0}}}},
			wantWaveforms: [][]byte{waveform},
			wantErr:       nil,
		}, {
			name:    "BadMagic",
			data:    bytes.Repeat([]byte{0x00}, 32),
			wantErr: vab.ErrInvalidHeader,
		}, {
			name:    "TruncatedHeader",
			data:    single[:100],
			wantErr: vab.ErrTruncated,
		}, {
			name:    "TruncatedWaveform",
			data:    single[:len(single)-8],
			wantErr: vab.ErrTruncated,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			reader := bytes.NewReader(tc.data)

			// Act
			bank, err := vab.Decode(reader)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Decode() error = %v, want %v", got, want)
			}
			if tc.wantErr != nil {
				return
			}
			if got, want := bank.Header, tc.wantHeader; !cmp.Equal(got, want) {
				t.Errorf("Decode() header diff (-got +want):\n%s", cmp.Diff(got, want))
			}
			if got, want := bank.Programs, tc.wantPrograms; !cmp.Equal(got, want) {
				t.Errorf("Decode() programs diff (-got +want):\n%s", cmp.Diff(got, want))
			}
			if got, want := waveforms(bank), tc.wantWaveforms; !cmp.Equal(got, want) {
				t.Errorf("Decode() waveforms diff (-got +want):\n%s", cmp.Diff(got, want))
			}
		})
	}
}

// waveforms collects every sample of a bank into a slice for comparison.
func waveforms(bank *vab.Bank) [][]byte {
	out := make([][]byte, bank.NumWaveforms())
	for i := range out {
		out[i] = bank.Waveform(i)
	}
	return out
}
