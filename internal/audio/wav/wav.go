package wav

import (
	"encoding/binary"
	"errors"
	"io"
)

// headerSize is the size of the canonical RIFF/WAVE header that precedes the PCM
// samples.
const headerSize = 44

// ErrInvalidFormat indicates a [Format] whose channel count, sample rate, or
// sample width is not positive.
var ErrInvalidFormat = errors.New("wav: invalid format")

// Format describes the layout of a stream of PCM samples.
type Format struct {
	// SampleRate is the number of frames per second, such as 44100.
	SampleRate int

	// Channels is the number of interleaved channels, such as 2 for stereo.
	Channels int

	// BitsPerSample is the width of one sample of one channel, such as 16.
	BitsPerSample int
}

// BlockAlign returns the number of bytes in one frame: one sample for every
// channel.
func (f Format) BlockAlign() int {
	return f.Channels * f.BitsPerSample / 8
}

// ByteRate returns the number of bytes of PCM per second of audio.
func (f Format) ByteRate() int {
	return f.SampleRate * f.BlockAlign()
}

// valid reports whether every field of the format is positive.
func (f Format) valid() bool {
	return f.SampleRate > 0 && f.Channels > 0 && f.BitsPerSample > 0
}

// Encode writes pcm to w as a canonical RIFF/WAVE file described by format. pcm
// holds interleaved little-endian samples. It returns [ErrInvalidFormat] for a
// non-positive format, or any error from writing to w.
func Encode(w io.Writer, format Format, pcm []byte) error {
	if !format.valid() {
		return ErrInvalidFormat
	}

	var h [headerSize]byte
	dataSize := uint32(len(pcm))
	put := binary.LittleEndian.PutUint32
	put16 := binary.LittleEndian.PutUint16

	copy(h[0:4], "RIFF")
	put(h[4:8], 36+dataSize)
	copy(h[8:12], "WAVE")
	copy(h[12:16], "fmt ")
	put(h[16:20], 16)
	put16(h[20:22], 1) // PCM
	put16(h[22:24], uint16(format.Channels))
	put(h[24:28], uint32(format.SampleRate))
	put(h[28:32], uint32(format.ByteRate()))
	put16(h[32:34], uint16(format.BlockAlign()))
	put16(h[34:36], uint16(format.BitsPerSample))
	copy(h[36:40], "data")
	put(h[40:44], dataSize)

	if _, err := w.Write(h[:]); err != nil {
		return err
	}
	_, err := w.Write(pcm)
	return err
}
