package str_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/subheader"
	"github.com/bitwizeshift/psx-decomp-tools/internal/media/str"
	"github.com/bitwizeshift/psx-decomp-tools/internal/media/str/strtest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestParseSectorHeader(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		data    []byte
		want    str.SectorHeader
		wantErr error
	}{
		{
			name: "MDECHeader",
			data: strtest.New().Sector(strtest.Sector{
				Status: str.StatusMagic, Type: str.TypeMDEC,
				Index: 0, Count: 9, Frame: 1, DataSize: 11412, Width: 320, Height: 220,
			}).Bytes(),
			want: str.SectorHeader{
				Status: str.StatusMagic, Type: str.TypeMDEC,
				SectorIndex: 0, SectorCount: 9, FrameNumber: 1, DataSize: 11412, Width: 320, Height: 220,
			},
			wantErr: nil,
		}, {
			name:    "Short",
			data:    make([]byte, 16),
			want:    str.SectorHeader{},
			wantErr: str.ErrShortSector,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange / Act
			got, err := str.ParseSectorHeader(tc.data)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("ParseSectorHeader() error = %v, want %v", got, want)
			}
			if got, want := got, tc.want; !cmp.Equal(got, want) {
				t.Errorf("ParseSectorHeader() diff (-got +want):\n%s", cmp.Diff(got, want))
			}
		})
	}
}

func TestSectorHeaderMDEC(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		header str.SectorHeader
		want   bool
	}{
		{
			name:   "MDEC",
			header: str.SectorHeader{Status: str.StatusMagic, Type: str.TypeMDEC},
			want:   true,
		}, {
			name:   "WrongStatus",
			header: str.SectorHeader{Status: 0, Type: str.TypeMDEC},
			want:   false,
		}, {
			name:   "WrongType",
			header: str.SectorHeader{Status: str.StatusMagic, Type: 0},
			want:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange / Act
			got := tc.header.MDEC()

			// Assert
			if got, want := got, tc.want; got != want {
				t.Errorf("MDEC() = %v, want %v", got, want)
			}
		})
	}
}

func TestDemuxerNextFrame(t *testing.T) {
	t.Parallel()

	big := repeatByte(0xAB, 5000)

	testCases := []struct {
		name    string
		builder *strtest.Builder
		reader  func([]byte) io.Reader
		want    []*str.Frame
		wantErr error
	}{
		{
			name:    "SingleFrame",
			builder: strtest.New().Video(strtest.Frame{Number: 1, Width: 320, Height: 240, Bitstream: []byte{1, 2, 3, 4}}),
			want: []*str.Frame{
				{Number: 1, Width: 320, Height: 240, Bitstream: []byte{1, 2, 3, 4}},
			},
			wantErr: io.EOF,
		}, {
			name: "SkipsAudio",
			builder: strtest.New().
				Audio(0).
				Video(strtest.Frame{Number: 1, Width: 16, Height: 16, Bitstream: []byte{9}}).
				Audio(1),
			want: []*str.Frame{
				{Number: 1, Width: 16, Height: 16, Bitstream: []byte{9}},
			},
			wantErr: io.EOF,
		}, {
			name: "SkipsNonMDEC",
			builder: strtest.New().
				Sector(strtest.Sector{Status: 0, Type: 0, Payload: []byte{0xDE, 0xAD}}).
				Video(strtest.Frame{Number: 7, Width: 32, Height: 32, Bitstream: []byte{5, 6}}),
			want: []*str.Frame{
				{Number: 7, Width: 32, Height: 32, Bitstream: []byte{5, 6}},
			},
			wantErr: io.EOF,
		}, {
			name: "MultiSectorFrame",
			builder: strtest.New().
				Video(strtest.Frame{Number: 2, Width: 320, Height: 224, Bitstream: big}),
			want: []*str.Frame{
				{Number: 2, Width: 320, Height: 224, Bitstream: big},
			},
			wantErr: io.EOF,
		}, {
			name: "TwoFrames",
			builder: strtest.New().
				Video(strtest.Frame{Number: 1, Width: 16, Height: 16, Bitstream: []byte{1}}).
				Video(strtest.Frame{Number: 2, Width: 16, Height: 16, Bitstream: []byte{2}}),
			want: []*str.Frame{
				{Number: 1, Width: 16, Height: 16, Bitstream: []byte{1}},
				{Number: 2, Width: 16, Height: 16, Bitstream: []byte{2}},
			},
			wantErr: io.EOF,
		}, {
			name: "UntrimmedWhenNoDataSize",
			builder: strtest.New().Sector(strtest.Sector{
				Status: str.StatusMagic, Type: str.TypeMDEC,
				Index: 0, Count: 1, Frame: 3, Width: 16, Height: 16, DataSize: 0, Payload: []byte{7, 8, 9},
			}),
			want: []*str.Frame{
				{Number: 3, Width: 16, Height: 16, Bitstream: untrimmedPayload([]byte{7, 8, 9})},
			},
			wantErr: io.EOF,
		}, {
			name: "BadFrame",
			builder: strtest.New().
				Sector(strtest.Sector{Status: str.StatusMagic, Type: str.TypeMDEC, Index: 0, Count: 2, Frame: 1}).
				Sector(strtest.Sector{Status: str.StatusMagic, Type: str.TypeMDEC, Index: 0, Count: 2, Frame: 1}),
			want:    nil,
			wantErr: str.ErrBadFrame,
		}, {
			name:    "TruncatedStream",
			builder: strtest.New().Video(strtest.Frame{Number: 1, Width: 16, Height: 16, Bitstream: []byte{1}}),
			reader:  func(data []byte) io.Reader { return bytes.NewReader(data[:len(data)-4]) },
			want:    nil,
			wantErr: io.ErrUnexpectedEOF,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			data, subs := tc.builder.Build()
			var reader io.Reader = bytes.NewReader(data)
			if tc.reader != nil {
				reader = tc.reader(data)
			}
			demuxer := str.NewDemuxer(reader, subs)

			// Act
			got, err := collectFrames(demuxer)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("NextFrame() error = %v, want %v", got, want)
			}
			if got, want := got, tc.want; !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
				t.Errorf("NextFrame() diff (-got +want):\n%s", cmp.Diff(got, want, cmpopts.EquateEmpty()))
			}
		})
	}
}

func TestDemuxerNext(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		builder *strtest.Builder
		want    []*str.Packet
		wantErr error
	}{
		{
			name: "InterleavedVideoAudioData",
			builder: strtest.New().
				Audio(0).
				Video(strtest.Frame{Number: 1, Width: 16, Height: 16, Bitstream: []byte{9}}).
				Data([]byte{0xDE, 0xAD}).
				Audio(1),
			want: []*str.Packet{
				{Audio: &str.AudioPacket{Channel: 0, Coding: 0, Data: audioData()}},
				{Video: &str.Frame{Number: 1, Width: 16, Height: 16, Bitstream: []byte{9}}},
				{Data: &str.DataPacket{Channel: 0, Data: fillerData([]byte{0xDE, 0xAD})}},
				{Audio: &str.AudioPacket{Channel: 1, Coding: 0, Data: audioData()}},
			},
			wantErr: io.EOF,
		}, {
			name: "AudioOnlyWithFiller",
			builder: strtest.New().
				Audio(2).
				Data([]byte{0xDE}).
				Audio(2),
			want: []*str.Packet{
				{Audio: &str.AudioPacket{Channel: 2, Coding: 0, Data: audioData()}},
				{Data: &str.DataPacket{Channel: 0, Data: fillerData([]byte{0xDE})}},
				{Audio: &str.AudioPacket{Channel: 2, Coding: 0, Data: audioData()}},
			},
			wantErr: io.EOF,
		}, {
			name: "VideoSubModeNonMDECIsData",
			builder: strtest.New().
				Sector(strtest.Sector{Status: 0, Type: 0, Payload: []byte{0xBE, 0xEF}}),
			want: []*str.Packet{
				{Data: &str.DataPacket{Channel: 0, Data: sectorPayload([]byte{0xBE, 0xEF})}},
			},
			wantErr: io.EOF,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			data, subs := tc.builder.Build()
			demuxer := str.NewDemuxer(bytes.NewReader(data), subs)

			// Act
			got, err := collectPackets(demuxer)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Next() error = %v, want %v", got, want)
			}
			if got, want := got, tc.want; !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
				t.Errorf("Next() diff (-got +want):\n%s", cmp.Diff(got, want, cmpopts.EquateEmpty()))
			}
		})
	}
}

// collectPackets drains every packet from d, returning the packets read and the
// terminating error, which is [io.EOF] for a fully consumed stream.
func collectPackets(d *str.Demuxer) ([]*str.Packet, error) {
	var packets []*str.Packet
	for {
		packet, err := d.Next()
		if err != nil {
			return packets, err
		}
		packets = append(packets, packet)
	}
}

// audioData returns the zeroed user data of a synthetic audio sector.
func audioData() []byte {
	return make([]byte, subheader.Form2Size)
}

// fillerData returns a whole filler sector's user data: prefix at its start,
// padded with zeros, as the strtest.Builder.Data sector is laid down.
func fillerData(prefix []byte) []byte {
	out := make([]byte, subheader.Form1Size)
	copy(out, prefix)
	return out
}

// sectorPayload returns a whole sector's user data with prefix written after the
// zeroed STR header slot, as the strtest.Builder.Sector payload is laid down.
func sectorPayload(prefix []byte) []byte {
	out := make([]byte, subheader.Form1Size)
	copy(out[str.SectorHeaderSize:], prefix)
	return out
}

// collectFrames drains every frame from d, returning the frames read and the
// terminating error, which is [io.EOF] for a fully consumed stream.
func collectFrames(d *str.Demuxer) ([]*str.Frame, error) {
	var frames []*str.Frame
	for {
		frame, err := d.NextFrame()
		if err != nil {
			return frames, err
		}
		frames = append(frames, frame)
	}
}

func repeatByte(value byte, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = value
	}
	return out
}

// untrimmedPayload returns a whole sector's bitstream payload, prefix followed by
// the zero padding a demuxed frame retains when its header declares no data size.
func untrimmedPayload(prefix []byte) []byte {
	out := make([]byte, 2048-str.SectorHeaderSize)
	copy(out, prefix)
	return out
}
