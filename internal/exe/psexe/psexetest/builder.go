package psexetest

import (
	"encoding/binary"

	"github.com/bitwizeshift/psx-decomp-tools/internal/exe/psexe"
)

// builder accumulates the pieces of a PS-X EXE stream before they are encoded.
type builder struct {
	magic     string
	pc0       uint32
	gp0       uint32
	textAddr  uint32
	text      []byte
	dataAddr  uint32
	dataSize  uint32
	bssAddr   uint32
	bssSize   uint32
	stackAddr uint32
	stackSize uint32
	region    string
	cut       *int
}

// Option configures the PS-X EXE stream built by [New].
type Option interface {
	apply(*builder)
}

// option adapts a plain function to the [Option] interface.
type option func(*builder)

func (o option) apply(b *builder) { o(b) }

// PC0 sets the initial program counter recorded in the header.
func PC0(addr uint32) Option {
	return option(func(b *builder) { b.pc0 = addr })
}

// GP0 sets the initial global pointer recorded in the header.
func GP0(addr uint32) Option {
	return option(func(b *builder) { b.gp0 = addr })
}

// TextAddr sets the address the text payload is loaded to.
func TextAddr(addr uint32) Option {
	return option(func(b *builder) { b.textAddr = addr })
}

// Text sets the text payload bytes. Its length becomes the header's text size.
func Text(data []byte) Option {
	return option(func(b *builder) { b.text = data })
}

// Data sets the data region's address and size.
func Data(addr, size uint32) Option {
	return option(func(b *builder) {
		b.dataAddr = addr
		b.dataSize = size
	})
}

// BSS sets the zero-initialized region's address and size.
func BSS(addr, size uint32) Option {
	return option(func(b *builder) {
		b.bssAddr = addr
		b.bssSize = size
	})
}

// Stack sets the stack base address and offset.
func Stack(addr, size uint32) Option {
	return option(func(b *builder) {
		b.stackAddr = addr
		b.stackSize = size
	})
}

// Region sets the ASCII marker string near the end of the header.
func Region(marker string) Option {
	return option(func(b *builder) { b.region = marker })
}

// BadMagic overwrites the magic tag, producing a stream that no longer begins
// with a valid PS-X EXE header.
func BadMagic() Option {
	return option(func(b *builder) { b.magic = "BAD!EXE!" })
}

// Truncate keeps only the first n bytes of the encoded stream.
func Truncate(n int) Option {
	return option(func(b *builder) { b.cut = &n })
}

// New assembles a PS-X EXE byte stream from opts. With no options it builds a
// minimal valid executable with an empty text payload.
func New(opts ...Option) []byte {
	b := &builder{magic: psexe.Magic}
	for _, opt := range opts {
		opt.apply(b)
	}
	return b.bytes()
}

// bytes encodes the accumulated builder into a PS-X EXE stream.
func (b *builder) bytes() []byte {
	out := make([]byte, psexe.HeaderSize)
	copy(out, b.magic)
	binary.LittleEndian.PutUint32(out[0x10:], b.pc0)
	binary.LittleEndian.PutUint32(out[0x14:], b.gp0)
	binary.LittleEndian.PutUint32(out[0x18:], b.textAddr)
	binary.LittleEndian.PutUint32(out[0x1c:], uint32(len(b.text)))
	binary.LittleEndian.PutUint32(out[0x20:], b.dataAddr)
	binary.LittleEndian.PutUint32(out[0x24:], b.dataSize)
	binary.LittleEndian.PutUint32(out[0x28:], b.bssAddr)
	binary.LittleEndian.PutUint32(out[0x2c:], b.bssSize)
	binary.LittleEndian.PutUint32(out[0x30:], b.stackAddr)
	binary.LittleEndian.PutUint32(out[0x34:], b.stackSize)
	copy(out[0x4c:], b.region)

	out = append(out, b.text...)
	if b.cut != nil && *b.cut < len(out) {
		out = out[:*b.cut]
	}
	return out
}
