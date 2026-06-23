package mdectest

import "encoding/binary"

// fileID is the identifier word a BS header carries.
const fileID = 0x3800

// endOfFrame is the 10-bit DC code that terminates a frame.
const endOfFrame = "0111111111"

// blockEnd is the AC Huffman code for end of block.
const blockEnd = "10"

// Builder assembles a BS frame: an 8-byte header and a bitstream packed from the
// appended bits. Build one with [New].
type Builder struct {
	quant   int
	version int
	bits    []byte
}

// New returns a [Builder] for a version 2 frame with the given quantization
// scale.
func New(quant int) *Builder {
	return &Builder{quant: quant, version: 2}
}

// Version sets the BS version written into the header.
func (b *Builder) Version(version int) *Builder {
	b.version = version
	return b
}

// Bits appends the literal bits of code, a string of '0' and '1'.
func (b *Builder) Bits(code string) *Builder {
	for _, c := range code {
		b.bits = append(b.bits, byte(c-'0'))
	}
	return b
}

// DC appends a 10-bit DC code, most-significant bit first.
func (b *Builder) DC(value int) *Builder {
	for i := 9; i >= 0; i-- {
		b.bits = append(b.bits, byte(value>>i&1))
	}
	return b
}

// Block appends a DC-only block: its DC code followed by end of block.
func (b *Builder) Block(dc int) *Builder {
	return b.DC(dc).Bits(blockEnd)
}

// EndOfFrame appends the frame terminator.
func (b *Builder) EndOfFrame() *Builder {
	return b.Bits(endOfFrame)
}

// Build returns the header followed by the packed bitstream, the bits grouped
// most-significant first into little-endian halfwords and zero-padded to a whole
// halfword.
func (b *Builder) Build() []byte {
	out := make([]byte, 8)
	binary.LittleEndian.PutUint16(out[2:4], fileID)
	binary.LittleEndian.PutUint16(out[4:6], uint16(b.quant))
	binary.LittleEndian.PutUint16(out[6:8], uint16(b.version))

	for i := 0; i < len(b.bits); i += 16 {
		var word uint16
		for j := range 16 {
			word <<= 1
			if i+j < len(b.bits) {
				word |= uint16(b.bits[i+j])
			}
		}
		out = binary.LittleEndian.AppendUint16(out, word)
	}
	return out
}
