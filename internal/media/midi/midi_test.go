package midi_test

import (
	"bytes"
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/media/midi"
	"github.com/bitwizeshift/psx-decomp-tools/internal/media/midi/miditest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestDecodeConfig(t *testing.T) {
	t.Parallel()

	twoTracks := miditest.File(96, miditest.Track(), miditest.Track())

	testCases := []struct {
		name    string
		data    []byte
		want    midi.Header
		wantErr error
	}{
		{
			name:    "SingleTrack",
			data:    miditest.Sequence(48),
			want:    midi.Header{Format: 0, NumTracks: 1, Division: 48},
			wantErr: nil,
		}, {
			name:    "MultiTrack",
			data:    twoTracks,
			want:    midi.Header{Format: 1, NumTracks: 2, Division: 96},
			wantErr: nil,
		}, {
			name:    "BadMagic",
			data:    bytes.Repeat([]byte{0x00}, 14),
			want:    midi.Header{},
			wantErr: midi.ErrInvalidHeader,
		}, {
			name:    "ShortHeader",
			data:    []byte("MThd"),
			want:    midi.Header{},
			wantErr: midi.ErrInvalidHeader,
		}, {
			name:    "BadLength",
			data:    append([]byte("MThd"), 0x00, 0x00, 0x00, 0x02),
			want:    midi.Header{},
			wantErr: midi.ErrInvalidHeader,
		}, {
			name:    "ShortBody",
			data:    append([]byte("MThd"), 0x00, 0x00, 0x00, 0x06, 0x00, 0x01),
			want:    midi.Header{},
			wantErr: midi.ErrInvalidHeader,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			reader := bytes.NewReader(tc.data)

			// Act
			got, err := midi.DecodeConfig(reader)

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

	messages := miditest.Sequence(48,
		miditest.Event{Delta: 0, Message: miditest.SetTempo(500000)},
		miditest.Event{Delta: 0, Message: miditest.ProgramChange(0, 5)},
		miditest.Event{Delta: 0, Message: miditest.NoteOn(0, 60, 100)},
		miditest.Event{Delta: 10, Message: []byte{62, 100}},
		miditest.Event{Delta: 5, Message: miditest.NoteOff(0, 60)},
		miditest.Event{Delta: 1, Message: []byte{62}},
		miditest.Event{Delta: 0, Message: miditest.NoteOn(0, 60, 0)},
		miditest.Event{Delta: 0, Message: miditest.ControlChange(0, 7, 127)},
		miditest.Event{Delta: 0, Message: []byte{0xe0, 0x40}},
		miditest.Event{Delta: 0, Message: []byte{0xa0, 0x40, 0x50}},
		miditest.Event{Delta: 0, Message: []byte{0xd0, 0x33}},
		miditest.Event{Delta: 0, Message: []byte{0xf0, 0x02, 0x0a, 0x0b}},
		miditest.Event{Delta: 0, Message: []byte{0xff, 0x51, 0x02, 0x01, 0x02}},
		miditest.Event{Delta: 0, Message: miditest.EndOfTrack()},
	)
	unknownMeta := miditest.Sequence(48,
		miditest.Event{Delta: 0, Message: []byte{0xff, 0x03, 0x02, 'h', 'i'}},
		miditest.Event{Delta: 0, Message: miditest.EndOfTrack()},
	)
	noEndPadded := miditest.Sequence(48,
		miditest.Event{Delta: 0, Message: miditest.NoteOn(0, 60, 100)},
		miditest.Event{Delta: 5, Message: miditest.NoteOff(0, 60)},
		miditest.Event{Delta: 0, Message: []byte{0, 0, 0, 0}},
	)
	truncatedTail := miditest.Sequence(48,
		miditest.Event{Delta: 0, Message: miditest.NoteOn(0, 60, 100)},
		miditest.Event{Delta: 4, Message: []byte{0x90, 62}},
	)
	noTrackChunk := append(miditest.File(48), []byte("junk!!!!")...)
	noRunningStatus := miditest.Sequence(48,
		miditest.Event{Delta: 0, Message: []byte{0x40}},
	)
	truncatedDelta := miditest.Sequence(48,
		miditest.Event{Delta: 200, Message: nil},
	)
	truncatedSysex := miditest.Sequence(48,
		miditest.Event{Delta: 0, Message: []byte{0xf0}},
	)
	truncatedMeta := miditest.Sequence(48,
		miditest.Event{Delta: 0, Message: []byte{0xff}},
	)
	truncatedMetaLength := miditest.Sequence(48,
		miditest.Event{Delta: 0, Message: []byte{0xff, 0x51}},
	)

	testCases := []struct {
		name    string
		data    []byte
		want    *midi.Sequence
		wantErr error
	}{
		{
			name: "Messages",
			data: messages,
			want: &midi.Sequence{
				Header: midi.Header{Format: 0, NumTracks: 1, Division: 48},
				Tracks: []midi.Track{{Events: []midi.Event{
					{Delta: 0, Message: midi.SetTempo{MicrosPerQuarter: 500000}},
					{Delta: 0, Message: midi.ProgramChange{Channel: 0, Program: 5}},
					{Delta: 0, Message: midi.NoteOn{Channel: 0, Note: 60, Velocity: 100}},
					{Delta: 10, Message: midi.NoteOn{Channel: 0, Note: 62, Velocity: 100}},
					{Delta: 5, Message: midi.NoteOff{Channel: 0, Note: 60}},
					{Delta: 1, Message: midi.NoteOff{Channel: 0, Note: 62}},
					{Delta: 0, Message: midi.NoteOff{Channel: 0, Note: 60}},
					{Delta: 0, Message: midi.ControlChange{Channel: 0, Controller: 7, Value: 127}},
					{Delta: 0, Message: midi.Unknown{Status: 0xe0, Data: []byte{0x40}}},
					{Delta: 0, Message: midi.Unknown{Status: 0xa0, Data: []byte{0x40, 0x50}}},
					{Delta: 0, Message: midi.Unknown{Status: 0xd0, Data: []byte{0x33}}},
					{Delta: 0, Message: midi.Unknown{Status: 0xf0, Data: []byte{0x0a, 0x0b}}},
					{Delta: 0, Message: midi.Unknown{Status: 0xff, Data: []byte{0x01, 0x02}}},
					{Delta: 0, Message: midi.EndOfTrack{}},
				}}},
			},
			wantErr: nil,
		}, {
			name: "NoEndOfTrackWithPadding",
			data: noEndPadded,
			want: &midi.Sequence{
				Header: midi.Header{Format: 0, NumTracks: 1, Division: 48},
				Tracks: []midi.Track{{Events: []midi.Event{
					{Delta: 0, Message: midi.NoteOn{Channel: 0, Note: 60, Velocity: 100}},
					{Delta: 5, Message: midi.NoteOff{Channel: 0, Note: 60}},
				}}},
			},
			wantErr: nil,
		}, {
			name: "UnknownMeta",
			data: unknownMeta,
			want: &midi.Sequence{
				Header: midi.Header{Format: 0, NumTracks: 1, Division: 48},
				Tracks: []midi.Track{{Events: []midi.Event{
					{Delta: 0, Message: midi.Unknown{Status: 0xff, Data: []byte("hi")}},
					{Delta: 0, Message: midi.EndOfTrack{}},
				}}},
			},
			wantErr: nil,
		}, {
			name: "TruncatedTail",
			data: truncatedTail,
			want: &midi.Sequence{
				Header: midi.Header{Format: 0, NumTracks: 1, Division: 48},
				Tracks: []midi.Track{{Events: []midi.Event{
					{Delta: 0, Message: midi.NoteOn{Channel: 0, Note: 60, Velocity: 100}},
				}}},
			},
			wantErr: midi.ErrTruncated,
		}, {
			name: "NoTrackChunk",
			data: noTrackChunk,
			want: &midi.Sequence{
				Header: midi.Header{Format: 0, NumTracks: 0, Division: 48},
			},
			wantErr: nil,
		}, {
			name: "NoRunningStatus",
			data: noRunningStatus,
			want: &midi.Sequence{
				Header: midi.Header{Format: 0, NumTracks: 1, Division: 48},
				Tracks: []midi.Track{{}},
			},
			wantErr: midi.ErrTruncated,
		}, {
			name: "TruncatedDelta",
			data: truncatedDelta,
			want: &midi.Sequence{
				Header: midi.Header{Format: 0, NumTracks: 1, Division: 48},
				Tracks: []midi.Track{{}},
			},
			wantErr: midi.ErrTruncated,
		}, {
			name: "TruncatedSysex",
			data: truncatedSysex,
			want: &midi.Sequence{
				Header: midi.Header{Format: 0, NumTracks: 1, Division: 48},
				Tracks: []midi.Track{{}},
			},
			wantErr: midi.ErrTruncated,
		}, {
			name: "TruncatedMeta",
			data: truncatedMeta,
			want: &midi.Sequence{
				Header: midi.Header{Format: 0, NumTracks: 1, Division: 48},
				Tracks: []midi.Track{{}},
			},
			wantErr: midi.ErrTruncated,
		}, {
			name: "TruncatedMetaLength",
			data: truncatedMetaLength,
			want: &midi.Sequence{
				Header: midi.Header{Format: 0, NumTracks: 1, Division: 48},
				Tracks: []midi.Track{{}},
			},
			wantErr: midi.ErrTruncated,
		}, {
			name:    "BadMagic",
			data:    bytes.Repeat([]byte{0x00}, 14),
			want:    nil,
			wantErr: midi.ErrInvalidHeader,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			reader := bytes.NewReader(tc.data)

			// Act
			got, err := midi.Decode(reader)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Decode() error = %v, want %v", got, want)
			}
			if got, want := got, tc.want; !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
				t.Errorf("Decode() diff (-got +want):\n%s", cmp.Diff(got, want, cmpopts.EquateEmpty()))
			}
		})
	}
}
