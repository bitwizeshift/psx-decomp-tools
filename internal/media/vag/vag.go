package vag

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// Magic is the four-byte tag that begins a standalone VAG file. Its header fields
// are big-endian.
const Magic = "VAGp"

// Frame geometry for SPU ADPCM.
const (
	frameSize       = 16 // one shift/filter byte, one flag byte, 14 sample bytes
	samplesPerFrame = 28
	headerSize      = 48
)

// ADPCM predictor coefficients, scaled by 64, indexed by a frame's filter field.
var (
	filterK0 = [...]int32{0, 60, 115, 98, 122}
	filterK1 = [...]int32{0, 0, -52, -55, -60}
)

// Sentinel errors reported while decoding.
var (
	// ErrInvalidHeader indicates content that does not begin with a valid VAG
	// header.
	ErrInvalidHeader = errors.New("vag: invalid header")

	// ErrShortFrame indicates an ADPCM body whose length is not a whole number of
	// 16-byte frames.
	ErrShortFrame = errors.New("vag: short frame")
)

// Header describes a standalone VAG file.
type Header struct {
	// Version is the format version word from the header.
	Version uint32

	// SampleRate is the playback rate in Hz.
	SampleRate int

	// Channels is the number of channels, always one.
	Channels int

	// DataSize is the length of the ADPCM body in bytes.
	DataSize int

	// Name is the sample name, with trailing padding removed.
	Name string
}

// Sample is a decoded standalone VAG file: its header and its PCM samples.
type Sample struct {
	Header
	PCM []int16
}

// DecodeConfig reads the VAG header from r without decoding the body. It reports
// [ErrInvalidHeader] when the header is short or carries the wrong tag.
func DecodeConfig(r io.Reader) (Header, error) {
	var buf [headerSize]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return Header{}, fmt.Errorf("vag: %w: %w", ErrInvalidHeader, err)
	}
	if string(buf[0:4]) != Magic {
		return Header{}, fmt.Errorf("vag: %q: %w", buf[0:4], ErrInvalidHeader)
	}
	return Header{
		Version:    binary.BigEndian.Uint32(buf[4:8]),
		DataSize:   int(binary.BigEndian.Uint32(buf[12:16])),
		SampleRate: int(binary.BigEndian.Uint32(buf[16:20])),
		Channels:   1,
		Name:       string(bytes.TrimRight(buf[32:48], "\x00")),
	}, nil
}

// DecodeSample reads a standalone VAG file from r and decodes its body to PCM. It
// reports [ErrInvalidHeader] for a malformed header and [ErrShortFrame] for a body
// that is not a whole number of frames.
func DecodeSample(r io.Reader) (*Sample, error) {
	header, err := DecodeConfig(r)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("vag: %w", err)
	}
	n := len(body)
	if header.DataSize > 0 && header.DataSize < n {
		n = header.DataSize
	}
	pcm, err := Decode(body[:n-n%frameSize])
	if err != nil {
		return nil, err
	}
	return &Sample{Header: header, PCM: pcm}, nil
}

// SPU ADPCM frame-flag bits in a frame's second byte.
const (
	flagLoopEnd    = 0x01 // last block of the sample
	flagLoopRepeat = 0x02 // at the end, repeat from the loop start
	flagLoopStart  = 0x04 // marks the loop start block
)

// Loop describes a sample's sustain loop as PCM sample indices. Start is -1 when
// the sample does not repeat; otherwise the samples in [Start, End) play again
// each time playback reaches End while a voice sustains.
type Loop struct {
	Start int
	End   int
}

// Decode decodes a headerless SPU ADPCM body into mono PCM. data must be a whole
// number of 16-byte frames; it reports [ErrShortFrame] otherwise.
func Decode(data []byte) ([]int16, error) {
	if len(data)%frameSize != 0 {
		return nil, fmt.Errorf("vag: %d bytes: %w", len(data), ErrShortFrame)
	}
	out := make([]int16, 0, len(data)/frameSize*samplesPerFrame)
	var p predictor
	for off := 0; off < len(data); off += frameSize {
		out = p.decodeFrame(out, data[off:off+frameSize])
	}
	return out, nil
}

// DecodeWithLoop decodes a headerless SPU ADPCM body into mono PCM and reports
// its loop, stopping at the block flagged as the sample's end. data must be a
// whole number of 16-byte frames; it reports [ErrShortFrame] otherwise. The
// returned [Loop] has Start -1 unless an end block requests a repeat.
func DecodeWithLoop(data []byte) ([]int16, Loop, error) {
	if len(data)%frameSize != 0 {
		return nil, Loop{Start: -1}, fmt.Errorf("vag: %d bytes: %w", len(data), ErrShortFrame)
	}
	out := make([]int16, 0, len(data)/frameSize*samplesPerFrame)
	loop := Loop{Start: -1}
	repeats := false
	var p predictor
	for off := 0; off < len(data); off += frameSize {
		flags := data[off+1]
		if flags&flagLoopStart != 0 {
			loop.Start = len(out)
		}
		out = p.decodeFrame(out, data[off:off+frameSize])
		if flags&flagLoopEnd != 0 {
			repeats = flags&flagLoopRepeat != 0
			break
		}
	}
	loop.End = len(out)
	switch {
	case !repeats:
		loop.Start = -1
	case loop.Start < 0:
		loop.Start = 0
	}
	return out, loop, nil
}

// predictor holds the two previous samples of the ADPCM stream.
type predictor struct {
	old   int32
	older int32
}

// decodeFrame decodes one 16-byte frame, appending its 28 samples to out and
// advancing the predictor.
func (p *predictor) decodeFrame(out []int16, frame []byte) []int16 {
	shift := uint(frame[0] & 0x0f)
	filter := int(frame[0]>>4) & 0x0f
	if filter >= len(filterK0) {
		filter = 0
	}
	for i := range samplesPerFrame {
		raw := frame[2+i/2]
		nibble := raw & 0x0f
		if i&1 == 1 {
			nibble = raw >> 4
		}
		sample := int32(int16(uint16(nibble)<<12)) >> shift
		sample += (p.old*filterK0[filter] + p.older*filterK1[filter] + 32) >> 6
		sample = clamp16(sample)
		p.older = p.old
		p.old = sample
		out = append(out, int16(sample))
	}
	return out
}

// clamp16 limits sample to the range of a signed 16-bit integer.
func clamp16(sample int32) int32 {
	if sample > 32767 {
		return 32767
	}
	if sample < -32768 {
		return -32768
	}
	return sample
}
