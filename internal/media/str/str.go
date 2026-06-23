package str

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/subheader"
)

// SectorHeaderSize is the length of the STR header that begins every video
// sector's user data.
const SectorHeaderSize = 32

// Identifiers carried in a video sector's STR header.
const (
	// StatusMagic is the StStatus value that marks a standard STR video sector.
	StatusMagic = 0x0160

	// TypeMDEC is the StType value of an MDEC-compressed video sector.
	TypeMDEC = 0x8001
)

// payloadSize is the number of BS bitstream bytes a video sector carries after
// its STR header.
const payloadSize = subheader.Form1Size - SectorHeaderSize

// Sentinel errors reported while demuxing.
var (
	// ErrShortSector indicates a video sector whose user data is too small to hold
	// an STR header.
	ErrShortSector = errors.New("str: short sector")

	// ErrBadFrame indicates a frame whose sector headers disagree on its layout.
	ErrBadFrame = errors.New("str: bad frame")
)

// SectorHeader is the 32-byte STR header at the start of a video sector's user
// data. A frame's bitstream spans SectorCount sectors, each numbered by
// SectorIndex from zero.
type SectorHeader struct {
	// Status is the StStatus identifier; [StatusMagic] for a standard stream.
	Status uint16

	// Type is the StType identifier; [TypeMDEC] for MDEC video.
	Type uint16

	// SectorIndex is the zero-based position of this sector within its frame.
	SectorIndex int

	// SectorCount is the number of sectors the frame spans.
	SectorCount int

	// FrameNumber is the one-based frame ordinal.
	FrameNumber int

	// DataSize is the length in bytes of the frame's BS bitstream, excluding
	// sector headers and padding.
	DataSize int

	// Width is the frame's pixel width.
	Width int

	// Height is the frame's pixel height.
	Height int
}

// ParseSectorHeader decodes the STR header at the start of data. It reports
// [ErrShortSector] when data is smaller than [SectorHeaderSize].
func ParseSectorHeader(data []byte) (SectorHeader, error) {
	if len(data) < SectorHeaderSize {
		return SectorHeader{}, fmt.Errorf("str: %d bytes: %w", len(data), ErrShortSector)
	}
	return SectorHeader{
		Status:      binary.LittleEndian.Uint16(data[0:2]),
		Type:        binary.LittleEndian.Uint16(data[2:4]),
		SectorIndex: int(binary.LittleEndian.Uint16(data[4:6])),
		SectorCount: int(binary.LittleEndian.Uint16(data[6:8])),
		FrameNumber: int(binary.LittleEndian.Uint32(data[8:12])),
		DataSize:    int(binary.LittleEndian.Uint32(data[12:16])),
		Width:       int(binary.LittleEndian.Uint16(data[16:18])),
		Height:      int(binary.LittleEndian.Uint16(data[18:20])),
	}, nil
}

// MDEC reports whether the header marks a standard MDEC-compressed video sector.
func (h SectorHeader) MDEC() bool {
	return h.Status == StatusMagic && h.Type == TypeMDEC
}

// Frame is one demuxed video frame: its ordinal, pixel dimensions, and the BS
// bitstream that the mdec package decodes into a picture.
type Frame struct {
	// Number is the one-based frame ordinal from its STR header.
	Number int

	// Width is the frame's pixel width.
	Width int

	// Height is the frame's pixel height.
	Height int

	// Bitstream is the frame's reassembled BS bitstream, trimmed to its declared
	// length.
	Bitstream []byte
}

// AudioPacket is one XA audio sector demuxed from a stream: its channel and
// coding from the CD-XA subheader, and the raw Form 2 user data the xa package
// decodes into PCM.
type AudioPacket struct {
	// Channel is the interleaved channel the sector belongs to.
	Channel byte

	// Coding is the subheader coding byte describing the audio parameters.
	Coding byte

	// Data is the sector's user data.
	Data []byte
}

// DataPacket is one demuxed filler sector: a Form 1 sector that carries neither
// audio nor MDEC video. Its payload is the mastering tool's interleave padding,
// not a decodable media format, and is surfaced verbatim for extraction.
type DataPacket struct {
	// Channel is the interleaved channel the sector belongs to.
	Channel byte

	// Data is the sector's user data.
	Data []byte
}

// Packet is one demuxed unit of a stream: exactly one of Video, Audio, or Data
// is set.
type Packet struct {
	// Video is a completed video frame, or nil otherwise.
	Video *Frame

	// Audio is an audio sector, or nil otherwise.
	Audio *AudioPacket

	// Data is a filler sector, or nil otherwise.
	Data *DataPacket
}

// Demuxer separates the interleaved video frames, audio sectors, and filler data
// sectors of an STR stream, reading sector user data in order. The per-sector
// CD-XA subheaders supply each sector's size and kind; decode them from the
// stream's sidecar with the subheader package.
type Demuxer struct {
	r    io.Reader
	subs []subheader.Subheader
	pos  int

	header SectorHeader
	body   []byte
	next   int
	open   bool
}

// NewDemuxer returns a [Demuxer] reading the stream r whose sectors are described
// by subs, in stream order.
func NewDemuxer(r io.Reader, subs []subheader.Subheader) *Demuxer {
	return &Demuxer{r: r, subs: subs}
}

// Next returns the next packet: an audio or filler data sector as it appears, or
// a video frame once its last sector is read. It reports [io.EOF] when the stream
// is consumed, [ErrBadFrame] when a frame's sector headers are inconsistent, or
// any read error from the stream.
func (d *Demuxer) Next() (*Packet, error) {
	for d.pos < len(d.subs) {
		sub := d.subs[d.pos]
		buf := make([]byte, sub.UserDataSize())
		if _, err := io.ReadFull(d.r, buf); err != nil {
			return nil, fmt.Errorf("str: read sector %d: %w", d.pos, err)
		}
		d.pos++

		if sub.Audio() {
			return &Packet{Audio: &AudioPacket{Channel: sub.Channel, Coding: sub.Coding, Data: buf}}, nil
		}
		h, ok := d.videoHeader(sub, buf)
		if !ok {
			return &Packet{Data: &DataPacket{Channel: sub.Channel, Data: buf}}, nil
		}
		frame, err := d.accumulate(h, buf)
		if err != nil {
			return nil, err
		}
		if frame != nil {
			return &Packet{Video: frame}, nil
		}
	}
	return nil, io.EOF
}

// NextFrame returns the next complete video frame, skipping audio and data
// packets. It reports the same errors as [Demuxer.Next].
func (d *Demuxer) NextFrame() (*Frame, error) {
	for {
		packet, err := d.Next()
		if err != nil {
			return nil, err
		}
		if packet.Video != nil {
			return packet.Video, nil
		}
	}
}

// accumulate folds one video sector into the open frame, returning the completed
// [Frame] once its last sector arrives, or nil while it is still in progress. It
// reports [ErrBadFrame] when the sector does not extend the open frame in order.
func (d *Demuxer) accumulate(h SectorHeader, buf []byte) (*Frame, error) {
	if !d.open {
		if h.SectorIndex != 0 {
			return nil, nil
		}
		d.header, d.body, d.next, d.open = h, make([]byte, 0, h.SectorCount*payloadSize), 0, true
	}
	if h.SectorIndex != d.next || h.FrameNumber != d.header.FrameNumber {
		return nil, fmt.Errorf("str: frame %d sector %d: %w", d.header.FrameNumber, h.SectorIndex, ErrBadFrame)
	}
	d.body = append(d.body, buf[SectorHeaderSize:]...)
	if d.next++; d.next >= d.header.SectorCount {
		d.open = false
		return d.frame(d.header, d.body), nil
	}
	return nil, nil
}

// videoHeader reports the STR header of a sector that carries MDEC video, or
// false for an audio, non-MDEC, or undersized sector.
func (d *Demuxer) videoHeader(sub subheader.Subheader, data []byte) (SectorHeader, bool) {
	if sub.Form2() || sub.Audio() || len(data) < SectorHeaderSize {
		return SectorHeader{}, false
	}
	h, err := ParseSectorHeader(data)
	if err != nil || !h.MDEC() {
		return SectorHeader{}, false
	}
	return h, true
}

// frame builds a [Frame] from a reassembled body, trimming the trailing sector
// padding down to the bitstream length the header declares.
func (d *Demuxer) frame(h SectorHeader, body []byte) *Frame {
	bitstream := body
	if h.DataSize > 0 && h.DataSize <= len(body) {
		bitstream = body[:h.DataSize]
	}
	return &Frame{Number: h.FrameNumber, Width: h.Width, Height: h.Height, Bitstream: bitstream}
}
