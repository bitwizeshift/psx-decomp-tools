package sqv_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/media/midi"
	"github.com/bitwizeshift/psx-decomp-tools/internal/media/midi/miditest"
	"github.com/bitwizeshift/psx-decomp-tools/internal/media/sqv"
	"github.com/bitwizeshift/psx-decomp-tools/internal/media/vab"
	"github.com/bitwizeshift/psx-decomp-tools/internal/media/vab/vabtest"
	"github.com/bitwizeshift/psx-decomp-tools/internal/media/vag/vagtest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// constantSample is the value every frame of the test waveform decodes to.
const constantSample = 12288

func TestRender(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		note      byte
		pan       byte
		frame     int
		wantLeft  int16
		wantRight int16
	}{
		{
			name:      "SustainPannedLeft",
			note:      60,
			pan:       0,
			frame:     200,
			wantLeft:  constantSample,
			wantRight: 0,
		}, {
			name:      "SustainPannedRight",
			note:      60,
			pan:       127,
			frame:     200,
			wantLeft:  0,
			wantRight: constantSample,
		}, {
			name:      "OctaveUpStillSounding",
			note:      72,
			pan:       0,
			frame:     200,
			wantLeft:  constantSample,
			wantRight: 0,
		}, {
			name:      "OctaveUpSampleConsumed",
			note:      72,
			pan:       0,
			frame:     230,
			wantLeft:  0,
			wantRight: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			bank := constantBank(t, tc.pan)
			sequence := note(tc.note)

			// Act
			pcm, err := sqv.Render(sequence, bank)

			// Assert
			if err != nil {
				t.Fatalf("Render() error = %v, want nil", err)
			}
			left, right := frame(pcm, tc.frame)
			if got, want := left, tc.wantLeft; got != want {
				t.Errorf("Render() frame %d left = %d, want %d", tc.frame, got, want)
			}
			if got, want := right, tc.wantRight; got != want {
				t.Errorf("Render() frame %d right = %d, want %d", tc.frame, got, want)
			}
		})
	}
}

func TestRenderScenarios(t *testing.T) {
	t.Parallel()

	held := decodeSequence(miditest.Sequence(480,
		miditest.Event{Delta: 0, Message: miditest.ProgramChange(0, 0)},
		miditest.Event{Delta: 0, Message: miditest.NoteOn(0, 60, 127)},
		miditest.Event{Delta: 100, Message: miditest.EndOfTrack()},
	))
	tempo := decodeSequence(miditest.Sequence(480,
		miditest.Event{Delta: 0, Message: miditest.SetTempo(250000)},
		miditest.Event{Delta: 0, Message: miditest.ProgramChange(0, 0)},
		miditest.Event{Delta: 0, Message: miditest.NoteOn(0, 60, 127)},
		miditest.Event{Delta: 100, Message: miditest.NoteOff(0, 60)},
		miditest.Event{Delta: 0, Message: miditest.EndOfTrack()},
	))
	unmapped := decodeSequence(miditest.Sequence(480,
		miditest.Event{Delta: 0, Message: miditest.ProgramChange(0, 5)},
		miditest.Event{Delta: 0, Message: miditest.NoteOn(0, 60, 127)},
		miditest.Event{Delta: 100, Message: miditest.NoteOff(0, 60)},
		miditest.Event{Delta: 0, Message: miditest.EndOfTrack()},
	))
	zeroDivision := decodeSequence(miditest.Sequence(0,
		miditest.Event{Delta: 0, Message: miditest.ProgramChange(0, 0)},
		miditest.Event{Delta: 0, Message: miditest.NoteOn(0, 60, 127)},
		miditest.Event{Delta: 1, Message: miditest.NoteOff(0, 60)},
		miditest.Event{Delta: 0, Message: miditest.EndOfTrack()},
	))

	chord := decodeSequence(miditest.Sequence(480,
		miditest.Event{Delta: 0, Message: miditest.ProgramChange(0, 0)},
		miditest.Event{Delta: 0, Message: miditest.NoteOn(0, 60, 127)},
		miditest.Event{Delta: 0, Message: miditest.NoteOn(0, 64, 127)},
		miditest.Event{Delta: 0, Message: miditest.NoteOn(0, 67, 127)},
		miditest.Event{Delta: 100, Message: miditest.NoteOff(0, 60)},
		miditest.Event{Delta: 0, Message: miditest.NoteOff(0, 64)},
		miditest.Event{Delta: 0, Message: miditest.NoteOff(0, 67)},
		miditest.Event{Delta: 0, Message: miditest.EndOfTrack()},
	))

	narrow := vabtest.Tone{Volume: 127, CenterNote: 60, NoteMin: 70, NoteMax: 80, Waveform: 0}
	playable := vabtest.Tone{Volume: 127, CenterNote: 60, NoteMin: 0, NoteMax: 127, Waveform: 0}

	testCases := []struct {
		name     string
		sequence *midi.Sequence
		bank     *vab.Bank
		wantLeft int16
	}{
		{
			name:     "HeldNoteSounds",
			sequence: held,
			bank:     constantBank(t, 0),
			wantLeft: constantSample,
		}, {
			name:     "TempoChangeSounds",
			sequence: tempo,
			bank:     constantBank(t, 0),
			wantLeft: constantSample,
		}, {
			name:     "UnmappedProgramSilent",
			sequence: unmapped,
			bank:     constantBank(t, 0),
			wantLeft: 0,
		}, {
			name:     "FallbackToneSounds",
			sequence: note(60),
			bank:     bankWith(t, narrow, constantWaveform(16)),
			wantLeft: constantSample,
		}, {
			name:     "UndecodableWaveformSilent",
			sequence: note(60),
			bank:     bankWith(t, playable, []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}),
			wantLeft: 0,
		}, {
			name:     "ZeroDivisionSounds",
			sequence: zeroDivision,
			bank:     constantBank(t, 0),
			wantLeft: constantSample,
		}, {
			name:     "OverlappingNotesClampAndReuseSample",
			sequence: chord,
			bank:     constantBank(t, 0),
			wantLeft: 32767,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Act
			pcm, err := sqv.Render(tc.sequence, tc.bank)

			// Assert
			if err != nil {
				t.Fatalf("Render() error = %v, want nil", err)
			}
			left, _ := frame(pcm, 200)
			if got, want := left, tc.wantLeft; got != want {
				t.Errorf("Render() frame 200 left = %d, want %d", got, want)
			}
		})
	}
}

func TestRenderSampleRate(t *testing.T) {
	t.Parallel()

	// Arrange
	bank := constantBank(t, 0)
	sequence := note(60)

	// Act
	pcm, err := sqv.Render(sequence, bank, sqv.WithSampleRate(22050))

	// Assert
	if err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}
	left, right := frame(pcm, 100)
	if got, want := left, int16(constantSample); got != want {
		t.Errorf("Render() frame 100 left = %d, want %d", got, want)
	}
	if got, want := right, int16(0); got != want {
		t.Errorf("Render() frame 100 right = %d, want %d", got, want)
	}
}

func TestRenderNoInput(t *testing.T) {
	t.Parallel()

	bank := constantBank(t, 0)
	sequence := note(60)

	testCases := []struct {
		name     string
		sequence *midi.Sequence
		bank     *vab.Bank
	}{
		{name: "NoSequence", sequence: nil, bank: bank},
		{name: "NoBank", sequence: sequence, bank: nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Act
			pcm, err := sqv.Render(tc.sequence, tc.bank)

			// Assert
			if got, want := err, sqv.ErrNoInput; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Render() error = %v, want %v", got, want)
			}
			if got, want := pcm, []byte(nil); !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
				t.Errorf("Render() pcm = %v, want nil", got)
			}
		})
	}
}

// constantBank returns a one-program, one-tone bank whose sample decodes to a
// constant, centered on note 60 and panned at pan.
func constantBank(t *testing.T, pan byte) *vab.Bank {
	t.Helper()
	tone := vabtest.Tone{Volume: 127, Pan: pan, CenterNote: 60, NoteMin: 0, NoteMax: 127, Waveform: 0}
	return bankWith(t, tone, constantWaveform(16))
}

// bankWith returns a one-program bank holding tone and a single waveform, at full
// program and master volume. The tone is given an envelope with an instant
// attack and a full sustain level so a held note holds at full volume.
func bankWith(t *testing.T, tone vabtest.Tone, waveform []byte) *vab.Bank {
	t.Helper()
	tone.ADSR1, tone.ADSR2 = 0x000f, 0x0000
	data := vabtest.Bank(
		[]vabtest.Program{{Volume: 127, Tones: []vabtest.Tone{tone}}},
		[][]byte{waveform},
		vabtest.WithMasterVolume(127),
	)
	bank, err := vab.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("vab.Decode() error = %v", err)
	}
	return bank
}

// constantWaveform returns the ADPCM body of frames that each decode to a
// constant sample.
func constantWaveform(frames int) []byte {
	nibbles := [28]byte{}
	for i := range nibbles {
		nibbles[i] = 3
	}
	frame := vagtest.Frame{Filter: 0, Shift: 0, Nibbles: nibbles}
	body := make([]vagtest.Frame, frames)
	for i := range body {
		body[i] = frame
	}
	return vagtest.Body(body...)
}

// note builds a sequence that plays one full-velocity note long enough to reach
// its sustain region.
func note(n byte) *midi.Sequence {
	return decodeSequence(miditest.Sequence(480,
		miditest.Event{Delta: 0, Message: miditest.ProgramChange(0, 0)},
		miditest.Event{Delta: 0, Message: miditest.NoteOn(0, n, 127)},
		miditest.Event{Delta: 100, Message: miditest.NoteOff(0, n)},
		miditest.Event{Delta: 0, Message: miditest.EndOfTrack()},
	))
}

func decodeSequence(data []byte) *midi.Sequence {
	seq, _ := midi.Decode(bytes.NewReader(data))
	return seq
}

// frame reads the left and right samples of one stereo frame from pcm, treating
// a frame past the end as silence.
func frame(pcm []byte, index int) (left, right int16) {
	off := index * 4
	if off+4 > len(pcm) {
		return 0, 0
	}
	left = int16(binary.LittleEndian.Uint16(pcm[off:]))
	right = int16(binary.LittleEndian.Uint16(pcm[off+2:]))
	return left, right
}
