package sqv_test

import (
	"bytes"
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/media/midi"
	"github.com/bitwizeshift/psx-decomp-tools/internal/media/midi/miditest"
	"github.com/bitwizeshift/psx-decomp-tools/internal/media/sqv"
	"github.com/bitwizeshift/psx-decomp-tools/internal/media/sqv/sqvtest"
	"github.com/bitwizeshift/psx-decomp-tools/internal/media/vab"
	"github.com/bitwizeshift/psx-decomp-tools/internal/media/vab/vabtest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestDecodeConfig(t *testing.T) {
	t.Parallel()

	bank := vabtest.Bank(nil, nil)
	twoSeq := sqvtest.File([][]byte{miditest.Sequence(48), miditest.Sequence(48)}, bank)

	testCases := []struct {
		name    string
		data    []byte
		want    sqv.Header
		wantErr error
	}{
		{
			name:    "SingleSequence",
			data:    sqvtest.File([][]byte{miditest.Sequence(48)}, bank),
			want:    sqv.Header{NextSequence: 0},
			wantErr: nil,
		}, {
			name:    "TwoSequences",
			data:    twoSeq,
			want:    sqv.Header{NextSequence: 12 + len(miditest.Sequence(48))},
			wantErr: nil,
		}, {
			name:    "BadMagic",
			data:    bytes.Repeat([]byte{0x00}, 16),
			want:    sqv.Header{},
			wantErr: sqv.ErrInvalidHeader,
		}, {
			name:    "Short",
			data:    []byte(".sqv"),
			want:    sqv.Header{},
			wantErr: sqv.ErrInvalidHeader,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			reader := bytes.NewReader(tc.data)

			// Act
			got, err := sqv.DecodeConfig(reader)

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
	bank := vabtest.Bank(
		[]vabtest.Program{{Tones: []vabtest.Tone{{CenterNote: 60, Waveform: 0}}}},
		[][]byte{waveform},
	)
	sequence := miditest.Sequence(48,
		miditest.Event{Delta: 0, Message: miditest.NoteOn(0, 60, 100)},
		miditest.Event{Delta: 4, Message: miditest.NoteOff(0, 60)},
		miditest.Event{Delta: 0, Message: miditest.EndOfTrack()},
	)
	wantSequence := &midi.Sequence{
		Header: midi.Header{Format: 0, NumTracks: 1, Division: 48},
		Tracks: []midi.Track{{Events: []midi.Event{
			{Delta: 0, Message: midi.NoteOn{Channel: 0, Note: 60, Velocity: 100}},
			{Delta: 4, Message: midi.NoteOff{Channel: 0, Note: 60}},
			{Delta: 0, Message: midi.EndOfTrack{}},
		}}},
	}

	testCases := []struct {
		name          string
		data          []byte
		wantSequences []*midi.Sequence
		wantWaveforms int
		wantErr       error
	}{
		{
			name:          "Single",
			data:          sqvtest.File([][]byte{sequence}, bank),
			wantSequences: []*midi.Sequence{wantSequence},
			wantWaveforms: 1,
			wantErr:       nil,
		}, {
			name:          "TwoSequences",
			data:          sqvtest.File([][]byte{sequence, sequence}, bank),
			wantSequences: []*midi.Sequence{wantSequence, wantSequence},
			wantWaveforms: 1,
			wantErr:       nil,
		}, {
			name:    "BadMagic",
			data:    bytes.Repeat([]byte{0x00}, 16),
			wantErr: sqv.ErrInvalidHeader,
		}, {
			name:    "MissingBank",
			data:    sqvtest.File([][]byte{sequence}, nil),
			wantErr: sqv.ErrMissingBank,
		}, {
			name:    "MalformedBank",
			data:    sqvtest.File([][]byte{sequence}, []byte("pBAVbroken")),
			wantErr: vab.ErrInvalidHeader,
		}, {
			name:          "SkipsMalformedSequence",
			data:          sqvtest.File([][]byte{append([]byte(midi.HeaderMagic), 0, 0, 0, 1), sequence}, bank),
			wantSequences: []*midi.Sequence{wantSequence},
			wantWaveforms: 1,
			wantErr:       nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			reader := bytes.NewReader(tc.data)

			// Act
			file, err := sqv.Decode(reader)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Decode() error = %v, want %v", got, want)
			}
			if tc.wantErr != nil {
				return
			}
			if got, want := file.Sequences, tc.wantSequences; !cmp.Equal(got, want) {
				t.Errorf("Decode() sequences diff (-got +want):\n%s", cmp.Diff(got, want))
			}
			if got, want := file.Bank.NumWaveforms(), tc.wantWaveforms; got != want {
				t.Errorf("Decode() waveforms = %d, want %d", got, want)
			}
		})
	}
}
