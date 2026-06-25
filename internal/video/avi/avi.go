package avi

import (
	"encoding/binary"
	"errors"
	"image"
	"io"
)

// Fixed sizes of the AVI header structures, in bytes.
const (
	avihSize    = 56 // MainAVIHeader body
	strhSize    = 56 // AVIStreamHeader body
	vidStrfSize = 40 // BITMAPINFOHEADER
	audStrfSize = 18 // WAVEFORMATEX with zero cbSize
	chunkHeader = 8  // FOURCC + size
	listHeader  = 12 // "LIST" + size + list type
)

// Derived list sizes recorded in the header.
const (
	videoStrlBody = chunkHeader + strhSize + chunkHeader + vidStrfSize
	audioStrlBody = chunkHeader + strhSize + chunkHeader + audStrfSize
	videoStrlSize = 4 + videoStrlBody // "strl" + body
	audioStrlSize = 4 + audioStrlBody
	hdrlBody      = chunkHeader + avihSize + listHeader + videoStrlBody + listHeader + audioStrlBody
	hdrlSize      = 4 + hdrlBody // "hdrl" + body
)

// AVI flag and index bits.
const (
	flagHasIndex      = 0x00000010
	flagInterleaved   = 0x00000100
	indexKeyframe     = 0x00000010
	indexEntrySize    = 16
	bitsPerSampleByte = 8
)

// ErrInvalidFormat indicates a [VideoFormat] or [AudioFormat] whose dimensions,
// frame rate, or audio parameters are not positive.
var ErrInvalidFormat = errors.New("avi: invalid format")

// VideoFormat describes the video stream. The codec tag comes from the
// [FrameEncoder], not this struct.
type VideoFormat struct {
	// Width and Height are the frame dimensions in pixels.
	Width  int
	Height int

	// FrameRateNum and FrameRateDen are the numerator and denominator of the
	// frame rate in frames per second.
	FrameRateNum int
	FrameRateDen int
}

// valid reports whether the format has positive dimensions and frame rate.
func (f VideoFormat) valid() bool {
	return f.Width > 0 && f.Height > 0 && f.FrameRateNum > 0 && f.FrameRateDen > 0
}

// AudioFormat describes the PCM audio stream.
type AudioFormat struct {
	// SampleRate is the number of frames per second.
	SampleRate int

	// Channels is the number of interleaved channels.
	Channels int

	// BitsPerSample is the width of one sample of one channel.
	BitsPerSample int
}

// valid reports whether every field of the format is positive.
func (f AudioFormat) valid() bool {
	return f.SampleRate > 0 && f.Channels > 0 && f.BitsPerSample > 0
}

// blockAlign returns the number of bytes in one audio frame across all channels.
func (f AudioFormat) blockAlign() int {
	return f.Channels * f.BitsPerSample / bitsPerSampleByte
}

// indexEntry records one movi chunk for the trailing idx1.
type indexEntry struct {
	id     string
	offset uint32
	size   uint32
}

// Writer muxes video and audio chunks into an AVI over an [io.WriteSeeker],
// patching the header totals on [Writer.Close].
type Writer struct {
	ws    io.WriteSeeker
	enc   FrameEncoder
	video VideoFormat
	audio AudioFormat

	pos int64
	err error

	riffSizePos int64
	moviSizePos int64
	moviStart   int64
	framesPos   int64
	maxBytesPos int64
	videoLenPos int64
	audioLenPos int64

	frames     int
	audioBytes int64
	index      []indexEntry
}

// NewWriter returns a [Writer] that muxes frames described by video and audio,
// encoding each frame with enc. It writes the AVI header immediately and reports
// [ErrInvalidFormat] for a non-positive format, or any error from ws.
func NewWriter(ws io.WriteSeeker, video VideoFormat, audio AudioFormat, enc FrameEncoder) (*Writer, error) {
	if !video.valid() || !audio.valid() {
		return nil, ErrInvalidFormat
	}
	w := &Writer{ws: ws, enc: enc, video: video, audio: audio}
	w.writeHeader()
	if w.err != nil {
		return nil, w.err
	}
	return w, nil
}

// WriteVideo encodes img with the writer's [FrameEncoder] and writes it as the
// next video frame. It reports any encoding or write error.
func (w *Writer) WriteVideo(img image.Image) error {
	if w.err != nil {
		return w.err
	}
	data, err := w.enc.Encode(img)
	if err != nil {
		w.err = err
		return err
	}
	w.chunk("00dc", data)
	w.frames++
	return w.err
}

// WriteAudio writes pcm as the next audio chunk. pcm holds interleaved
// little-endian samples.
func (w *Writer) WriteAudio(pcm []byte) error {
	if w.err != nil {
		return w.err
	}
	w.chunk("01wb", pcm)
	w.audioBytes += int64(len(pcm))
	return w.err
}

// Close writes the index and patches the header totals. The [Writer] must not be
// used afterward.
func (w *Writer) Close() error {
	if w.err != nil {
		return w.err
	}
	indexStart := w.pos
	w.tag("idx1")
	w.u32(uint32(len(w.index) * indexEntrySize))
	for _, e := range w.index {
		w.tag(e.id)
		w.u32(indexKeyframe)
		w.u32(e.offset)
		w.u32(e.size)
	}
	end := w.pos

	w.patch(w.riffSizePos, uint32(end-chunkHeader))
	w.patch(w.moviSizePos, uint32(indexStart-(w.moviSizePos+4)))
	w.patch(w.framesPos, uint32(w.frames))
	w.patch(w.videoLenPos, uint32(w.frames))
	w.patch(w.audioLenPos, uint32(w.audioBytes/int64(w.audio.blockAlign())))
	w.patch(w.maxBytesPos, w.maxBytesPerSec())
	return w.err
}

// maxBytesPerSec returns the stream's average byte rate for the avih header.
func (w *Writer) maxBytesPerSec() uint32 {
	if w.frames == 0 {
		return 0
	}
	total := w.audioBytes + w.videoBytes()
	seconds := float64(w.frames) * float64(w.video.FrameRateDen) / float64(w.video.FrameRateNum)
	if seconds <= 0 {
		return 0
	}
	return uint32(float64(total) / seconds)
}

// videoBytes returns the total size of the written video chunks.
func (w *Writer) videoBytes() int64 {
	var total int64
	for _, e := range w.index {
		if e.id == "00dc" {
			total += int64(e.size)
		}
	}
	return total
}

// chunk writes one movi chunk and records its index entry, padding odd-length
// data to an even boundary.
func (w *Writer) chunk(id string, data []byte) {
	offset := uint32(w.pos - w.moviStart)
	w.tag(id)
	w.u32(uint32(len(data)))
	w.write(data)
	if len(data)%2 == 1 {
		w.write([]byte{0})
	}
	w.index = append(w.index, indexEntry{id: id, offset: offset, size: uint32(len(data))})
}

// writeHeader writes the RIFF header up to the open movi list, recording the
// offsets patched on close.
func (w *Writer) writeHeader() {
	w.tag("RIFF")
	w.riffSizePos = w.pos
	w.u32(0)
	w.tag("AVI ")

	w.tag("LIST")
	w.u32(hdrlSize)
	w.tag("hdrl")
	w.writeAVIH()
	w.writeVideoStream()
	w.writeAudioStream()

	w.tag("LIST")
	w.moviSizePos = w.pos
	w.u32(0)
	w.moviStart = w.pos
	w.tag("movi")
}

// writeAVIH writes the main AVI header.
func (w *Writer) writeAVIH() {
	w.tag("avih")
	w.u32(avihSize)
	w.u32(uint32(1_000_000 * int64(w.video.FrameRateDen) / int64(w.video.FrameRateNum)))
	w.maxBytesPos = w.pos
	w.u32(0)
	w.u32(0)
	w.u32(flagHasIndex | flagInterleaved)
	w.framesPos = w.pos
	w.u32(0)
	w.u32(0)
	w.u32(2)
	w.u32(0)
	w.u32(uint32(w.video.Width))
	w.u32(uint32(w.video.Height))
	w.u32(0)
	w.u32(0)
	w.u32(0)
	w.u32(0)
}

// writeVideoStream writes the video stream's header list.
func (w *Writer) writeVideoStream() {
	w.tag("LIST")
	w.u32(videoStrlSize)
	w.tag("strl")

	w.tag("strh")
	w.u32(strhSize)
	w.tag("vids")
	w.tag(w.enc.FourCC())
	w.u32(0)
	w.u16(0)
	w.u16(0)
	w.u32(0)
	w.u32(uint32(w.video.FrameRateDen))
	w.u32(uint32(w.video.FrameRateNum))
	w.u32(0)
	w.videoLenPos = w.pos
	w.u32(0)
	w.u32(0)
	w.u32(0xFFFFFFFF)
	w.u32(0)
	w.u16(0)
	w.u16(0)
	w.u16(uint16(w.video.Width))
	w.u16(uint16(w.video.Height))

	w.tag("strf")
	w.u32(vidStrfSize)
	w.u32(vidStrfSize)
	w.u32(uint32(w.video.Width))
	w.u32(uint32(w.video.Height))
	w.u16(1)
	w.u16(24)
	w.tag(w.enc.FourCC())
	w.u32(uint32(w.video.Width * w.video.Height * 3))
	w.u32(0)
	w.u32(0)
	w.u32(0)
	w.u32(0)
}

// writeAudioStream writes the audio stream's header list.
func (w *Writer) writeAudioStream() {
	align := w.audio.blockAlign()
	rate := w.audio.SampleRate * align

	w.tag("LIST")
	w.u32(audioStrlSize)
	w.tag("strl")

	w.tag("strh")
	w.u32(strhSize)
	w.tag("auds")
	w.u32(0)
	w.u32(0)
	w.u16(0)
	w.u16(0)
	w.u32(0)
	w.u32(uint32(align))
	w.u32(uint32(rate))
	w.u32(0)
	w.audioLenPos = w.pos
	w.u32(0)
	w.u32(0)
	w.u32(0xFFFFFFFF)
	w.u32(uint32(align))
	w.u16(0)
	w.u16(0)
	w.u16(0)
	w.u16(0)

	w.tag("strf")
	w.u32(audStrfSize)
	w.u16(1)
	w.u16(uint16(w.audio.Channels))
	w.u32(uint32(w.audio.SampleRate))
	w.u32(uint32(rate))
	w.u16(uint16(align))
	w.u16(uint16(w.audio.BitsPerSample))
	w.u16(0)
}

// write appends b to the stream, recording the first error.
func (w *Writer) write(b []byte) {
	if w.err != nil {
		return
	}
	n, err := w.ws.Write(b)
	w.pos += int64(n)
	if err != nil {
		w.err = err
	}
}

// tag writes a four-character code.
func (w *Writer) tag(s string) {
	w.write([]byte(s))
}

// u16 writes a little-endian 16-bit value.
func (w *Writer) u16(v uint16) {
	var b [2]byte
	binary.LittleEndian.PutUint16(b[:], v)
	w.write(b[:])
}

// u32 writes a little-endian 32-bit value.
func (w *Writer) u32(v uint32) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	w.write(b[:])
}

// patch overwrites the 32-bit value at off without disturbing the append
// position the writer tracks.
func (w *Writer) patch(off int64, v uint32) {
	if w.err != nil {
		return
	}
	if _, err := w.ws.Seek(off, io.SeekStart); err != nil {
		w.err = err
		return
	}
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	if _, err := w.ws.Write(b[:]); err != nil {
		w.err = err
	}
}
