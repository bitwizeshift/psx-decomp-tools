package vag_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/media/vag"
	"github.com/bitwizeshift/psx-decomp-tools/internal/media/vag/vagtest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestDecode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		data    []byte
		want    []int16
		wantErr error
	}{
		{
			name:    "ConstantNibble",
			data:    vagtest.Body(vagtest.Frame{Filter: 0, Shift: 12, Nibbles: fill(3)}),
			want:    repeat(3, 28),
			wantErr: nil,
		}, {
			name:    "NegativeNibble",
			data:    vagtest.Body(vagtest.Frame{Filter: 0, Shift: 12, Nibbles: fill(0x8)}),
			want:    repeat(-8, 28),
			wantErr: nil,
		}, {
			name:    "FilterOutOfRange",
			data:    vagtest.Body(vagtest.Frame{Filter: 15, Shift: 12, Nibbles: fill(3)}),
			want:    repeat(3, 28),
			wantErr: nil,
		}, {
			name:    "Empty",
			data:    []byte{},
			want:    []int16{},
			wantErr: nil,
		}, {
			name:    "PartialFrame",
			data:    make([]byte, 10),
			want:    nil,
			wantErr: vag.ErrShortFrame,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange / Act
			got, err := vag.Decode(tc.data)

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

func TestDecodeWithLoop(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		data     []byte
		want     []int16
		wantLoop vag.Loop
		wantErr  error
	}{
		{
			name: "Looping",
			data: vagtest.Body(
				vagtest.Frame{Shift: 12, Flags: 0x04, Nibbles: fill(3)},
				vagtest.Frame{Shift: 12, Flags: 0x03, Nibbles: fill(3)},
			),
			want:     repeat(3, 56),
			wantLoop: vag.Loop{Start: 0, End: 56},
			wantErr:  nil,
		}, {
			name: "OneShot",
			data: vagtest.Body(
				vagtest.Frame{Shift: 12, Flags: 0x00, Nibbles: fill(3)},
				vagtest.Frame{Shift: 12, Flags: 0x01, Nibbles: fill(3)},
			),
			want:     repeat(3, 56),
			wantLoop: vag.Loop{Start: -1, End: 56},
			wantErr:  nil,
		}, {
			name: "LoopStartMidStream",
			data: vagtest.Body(
				vagtest.Frame{Shift: 12, Flags: 0x00, Nibbles: fill(3)},
				vagtest.Frame{Shift: 12, Flags: 0x04, Nibbles: fill(3)},
				vagtest.Frame{Shift: 12, Flags: 0x03, Nibbles: fill(3)},
			),
			want:     repeat(3, 84),
			wantLoop: vag.Loop{Start: 28, End: 84},
			wantErr:  nil,
		}, {
			name: "RepeatWithoutExplicitStart",
			data: vagtest.Body(
				vagtest.Frame{Shift: 12, Flags: 0x00, Nibbles: fill(3)},
				vagtest.Frame{Shift: 12, Flags: 0x03, Nibbles: fill(3)},
			),
			want:     repeat(3, 56),
			wantLoop: vag.Loop{Start: 0, End: 56},
			wantErr:  nil,
		}, {
			name: "NoEndFlagDecodesAll",
			data: vagtest.Body(
				vagtest.Frame{Shift: 12, Flags: 0x00, Nibbles: fill(3)},
				vagtest.Frame{Shift: 12, Flags: 0x00, Nibbles: fill(3)},
			),
			want:     repeat(3, 56),
			wantLoop: vag.Loop{Start: -1, End: 56},
			wantErr:  nil,
		}, {
			name:     "PartialFrame",
			data:     make([]byte, 10),
			want:     nil,
			wantLoop: vag.Loop{Start: -1, End: 0},
			wantErr:  vag.ErrShortFrame,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange / Act
			got, loop, err := vag.DecodeWithLoop(tc.data)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("DecodeWithLoop() error = %v, want %v", got, want)
			}
			if got, want := got, tc.want; !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
				t.Errorf("DecodeWithLoop() pcm diff (-got +want):\n%s", cmp.Diff(got, want, cmpopts.EquateEmpty()))
			}
			if got, want := loop, tc.wantLoop; !cmp.Equal(got, want) {
				t.Errorf("DecodeWithLoop() loop = %+v, want %+v", got, want)
			}
		})
	}
}

func TestDecodePredictor(t *testing.T) {
	t.Parallel()

	// Filter 1 (K0 = 60/64) at shift 0 with a single impulse decays: 4096, then
	// each next sample is round(prev * 60 / 64).
	body := vagtest.Body(vagtest.Frame{Filter: 1, Shift: 0, Nibbles: impulse()})
	want := []int16{4096, 3840, 3600}

	// Arrange / Act
	got, err := vag.Decode(body)

	// Assert
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got, want := got[:3], want; !cmp.Equal(got, want) {
		t.Errorf("Decode() head = %v, want %v", got, want)
	}
}

func TestDecodeClamps(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		nibble byte
		want   int16
	}{
		{
			name:   "PositiveSaturates",
			nibble: 0x7,
			want:   32767,
		}, {
			name:   "NegativeSaturates",
			nibble: 0x8,
			want:   -32768,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			body := vagtest.Body(vagtest.Frame{Filter: 1, Shift: 0, Nibbles: fill(tc.nibble)})

			// Act
			got, err := vag.Decode(body)

			// Assert
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if got, want := got[1], tc.want; got != want {
				t.Errorf("Decode() sample[1] = %d, want %d", got, want)
			}
		})
	}
}

func TestDecodeConfig(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		data    []byte
		want    vag.Header
		wantErr error
	}{
		{
			name:    "Valid",
			data:    vagtest.File(22050, "kick", nil),
			want:    vag.Header{Version: 0x20, SampleRate: 22050, Channels: 1, DataSize: 0, Name: "kick"},
			wantErr: nil,
		}, {
			name:    "BadMagic",
			data:    bytes.Repeat([]byte{0x00}, 48),
			want:    vag.Header{},
			wantErr: vag.ErrInvalidHeader,
		}, {
			name:    "Short",
			data:    []byte("VAGp"),
			want:    vag.Header{},
			wantErr: vag.ErrInvalidHeader,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			reader := bytes.NewReader(tc.data)

			// Act
			got, err := vag.DecodeConfig(reader)

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

func TestDecodeSample(t *testing.T) {
	t.Parallel()

	body := vagtest.Body(vagtest.Frame{Filter: 0, Shift: 12, Nibbles: fill(2)})

	twoFrames := vagtest.File(44100, "loop", vagtest.Body(
		vagtest.Frame{Filter: 0, Shift: 12, Nibbles: fill(2)},
		vagtest.Frame{Filter: 0, Shift: 12, Nibbles: fill(5)},
	))
	binary.BigEndian.PutUint32(twoFrames[12:16], 16)

	testCases := []struct {
		name    string
		data    []byte
		want    *vag.Sample
		wantErr error
	}{
		{
			name: "Valid",
			data: vagtest.File(44100, "snare", body),
			want: &vag.Sample{
				Header: vag.Header{Version: 0x20, SampleRate: 44100, Channels: 1, DataSize: 16, Name: "snare"},
				PCM:    repeat(2, 28),
			},
			wantErr: nil,
		}, {
			name: "BodyExceedsDataSize",
			data: twoFrames,
			want: &vag.Sample{
				Header: vag.Header{Version: 0x20, SampleRate: 44100, Channels: 1, DataSize: 16, Name: "loop"},
				PCM:    repeat(2, 28),
			},
			wantErr: nil,
		}, {
			name:    "BadHeader",
			data:    bytes.Repeat([]byte{0x00}, 48),
			want:    nil,
			wantErr: vag.ErrInvalidHeader,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			reader := bytes.NewReader(tc.data)

			// Act
			got, err := vag.DecodeSample(reader)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("DecodeSample() error = %v, want %v", got, want)
			}
			if got, want := got, tc.want; !cmp.Equal(got, want) {
				t.Errorf("DecodeSample() diff (-got +want):\n%s", cmp.Diff(got, want))
			}
		})
	}
}

// fill returns a frame's nibbles all set to the given value.
func fill(nibble byte) [28]byte {
	var nibbles [28]byte
	for i := range nibbles {
		nibbles[i] = nibble
	}
	return nibbles
}

// impulse returns a frame's nibbles with a single leading one.
func impulse() [28]byte {
	var nibbles [28]byte
	nibbles[0] = 1
	return nibbles
}

// repeat returns a slice of n samples all set to value.
func repeat(value int16, n int) []int16 {
	out := make([]int16, n)
	for i := range out {
		out[i] = value
	}
	return out
}
