package strtest

import (
	"encoding/binary"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/subheader"
	"github.com/bitwizeshift/psx-decomp-tools/internal/media/str"
)

// Submode bytes of the sector kinds an STR stream interleaves.
const (
	// videoSubMode marks a Form 1 real-time video sector, as STR video sectors are.
	videoSubMode = 0x48

	// audioSubMode marks a Form 2 real-time audio sector.
	audioSubMode = 0x64

	// dataSubMode marks a Form 1 filler sector that carries neither audio nor video.
	dataSubMode = 0x00
)

// Frame describes a video frame to lay down across one or more STR sectors.
type Frame struct {
	Number    int
	Width     int
	Height    int
	Bitstream []byte
}

// Sector describes one Form 1 sector: its STR header fields and the bitstream
// payload that follows. Tests use it directly to build malformed or non-MDEC
// sectors.
type Sector struct {
	Status   uint16
	Type     uint16
	Index    int
	Count    int
	Frame    int
	DataSize int
	Width    int
	Height   int
	Payload  []byte
}

// Builder assembles a synthetic STR stream and the subheaders that describe it,
// in sector order. Build one with [New].
type Builder struct {
	data []byte
	subs []subheader.Subheader
}

// New returns an empty [Builder].
func New() *Builder {
	return &Builder{}
}

// Sector writes one Form 1 video sector with an explicit STR header.
func (b *Builder) Sector(s Sector) *Builder {
	sector := make([]byte, subheader.Form1Size)
	binary.LittleEndian.PutUint16(sector[0:2], s.Status)
	binary.LittleEndian.PutUint16(sector[2:4], s.Type)
	binary.LittleEndian.PutUint16(sector[4:6], uint16(s.Index))
	binary.LittleEndian.PutUint16(sector[6:8], uint16(s.Count))
	binary.LittleEndian.PutUint32(sector[8:12], uint32(s.Frame))
	binary.LittleEndian.PutUint32(sector[12:16], uint32(s.DataSize))
	binary.LittleEndian.PutUint16(sector[16:18], uint16(s.Width))
	binary.LittleEndian.PutUint16(sector[18:20], uint16(s.Height))
	copy(sector[str.SectorHeaderSize:], s.Payload)

	b.data = append(b.data, sector...)
	b.subs = append(b.subs, subheader.Subheader{File: 1, SubMode: videoSubMode})
	return b
}

// Video lays frame down across the sectors needed to hold its bitstream, writing
// a well-formed STR header into each.
func (b *Builder) Video(frame Frame) *Builder {
	count := sectorsFor(len(frame.Bitstream))
	for i := range count {
		b.Sector(Sector{
			Status:   str.StatusMagic,
			Type:     str.TypeMDEC,
			Index:    i,
			Count:    count,
			Frame:    frame.Number,
			DataSize: len(frame.Bitstream),
			Width:    frame.Width,
			Height:   frame.Height,
			Payload:  chunk(frame.Bitstream, i),
		})
	}
	return b
}

// Audio appends one XA audio sector on the given channel, its user data zeroed.
func (b *Builder) Audio(channel byte) *Builder {
	b.data = append(b.data, make([]byte, subheader.Form2Size)...)
	b.subs = append(b.subs, subheader.Subheader{File: 1, Channel: channel, SubMode: audioSubMode})
	return b
}

// Data appends one Form 1 filler sector whose user data holds payload, padded to
// the sector size. Its STR header slot is left zeroed, as mastering filler is.
func (b *Builder) Data(payload []byte) *Builder {
	sector := make([]byte, subheader.Form1Size)
	copy(sector, payload)

	b.data = append(b.data, sector...)
	b.subs = append(b.subs, subheader.Subheader{File: 1, SubMode: dataSubMode})
	return b
}

// Build returns the assembled stream and the subheaders describing it.
func (b *Builder) Build() ([]byte, []subheader.Subheader) {
	return b.data, b.subs
}

// Bytes returns the assembled stream without its subheaders.
func (b *Builder) Bytes() []byte {
	return b.data
}

// sectorsFor returns the number of sectors a bitstream of n bytes spans, at least
// one even when empty.
func sectorsFor(n int) int {
	payload := subheader.Form1Size - str.SectorHeaderSize
	if n <= 0 {
		return 1
	}
	return (n + payload - 1) / payload
}

// chunk returns the bitstream payload of sector index, or nil past the end.
func chunk(bitstream []byte, index int) []byte {
	payload := subheader.Form1Size - str.SectorHeaderSize
	off := index * payload
	if off >= len(bitstream) {
		return nil
	}
	end := min(off+payload, len(bitstream))
	return bitstream[off:end]
}
