package timtest

import (
	"encoding/binary"

	"github.com/bitwizeshift/psx-decomp-tools/internal/image/tim"
)

// block is a CLUT or pixel block under construction. width is in 16-bit words.
type block struct {
	x, y   int
	width  int
	height int
	data   []byte
	bnum   *uint32
}

// builder accumulates the pieces of a TIM stream before they are encoded.
type builder struct {
	id    uint32
	mode  tim.PixelMode
	flag  *uint32
	clut  *block
	image block
	cut   *int
}

// Option configures the TIM stream built by [New].
type Option interface {
	apply(*builder)
}

// option adapts a plain function to the [Option] interface.
type option func(*builder)

func (o option) apply(b *builder) { o(b) }

// Mode selects the pixel mode written into the flag word.
func Mode(mode tim.PixelMode) Option {
	return option(func(b *builder) { b.mode = mode })
}

// Pixels sets the pixel block. width is the block width in 16-bit words, height
// is its row count, and data is its raw bytes.
func Pixels(width, height int, data []byte) Option {
	return option(func(b *builder) {
		b.image.width = width
		b.image.height = height
		b.image.data = data
	})
}

// ImageCoords sets the framebuffer coordinates of the pixel block.
func ImageCoords(x, y int) Option {
	return option(func(b *builder) {
		b.image.x, b.image.y = x, y
	})
}

// Palette adds a CLUT block and turns on the CLUT flag. width is the number of
// 16-bit color entries per palette, height is the palette count, and data is its
// raw bytes.
func Palette(width, height int, data []byte) Option {
	return option(func(b *builder) {
		b.ensureCLUT().width = width
		b.clut.height = height
		b.clut.data = data
	})
}

// PaletteCoords sets the framebuffer coordinates of the CLUT block, adding the
// block if it is absent.
func PaletteCoords(x, y int) Option {
	return option(func(b *builder) {
		b.ensureCLUT().x = x
		b.clut.y = y
	})
}

// ID overrides the file's id word, which a valid TIM leaves at 0x10.
func ID(id uint32) Option {
	return option(func(b *builder) { b.id = id })
}

// Flag overrides the whole flag word, bypassing [Mode] and the CLUT bit.
func Flag(flag uint32) Option {
	return option(func(b *builder) { b.flag = &flag })
}

// PaletteByteCount overrides the CLUT block's declared byte count, adding the
// block if it is absent.
func PaletteByteCount(bnum uint32) Option {
	return option(func(b *builder) {
		b.ensureCLUT().bnum = &bnum
	})
}

// PixelByteCount overrides the pixel block's declared byte count.
func PixelByteCount(bnum uint32) Option {
	return option(func(b *builder) { b.image.bnum = &bnum })
}

// Truncate keeps only the first n bytes of the encoded stream.
func Truncate(n int) Option {
	return option(func(b *builder) { b.cut = &n })
}

// U16s packs values as little-endian 16-bit words, a convenience for building
// color and pixel data.
func U16s(values ...uint16) []byte {
	data := make([]byte, len(values)*2)
	for i, v := range values {
		binary.LittleEndian.PutUint16(data[i*2:], v)
	}
	return data
}

// New assembles a TIM byte stream from opts. With no options it builds a minimal
// valid 16-bit, single-pixel image.
func New(opts ...Option) []byte {
	b := &builder{
		id:    0x10,
		mode:  tim.Mode16bpp,
		image: block{width: 1, height: 1, data: []byte{0x00, 0x00}},
	}
	for _, opt := range opts {
		opt.apply(b)
	}
	return b.bytes()
}

// ensureCLUT returns the CLUT block, creating an empty one when none has been
// configured yet.
func (b *builder) ensureCLUT() *block {
	if b.clut == nil {
		b.clut = &block{}
	}
	return b.clut
}

// bytes encodes the accumulated builder into a TIM stream.
func (b *builder) bytes() []byte {
	out := appendU32(nil, b.id)

	flag := uint32(b.mode)
	if b.clut != nil {
		flag |= 0x08
	}
	if b.flag != nil {
		flag = *b.flag
	}
	out = appendU32(out, flag)

	if b.clut != nil {
		out = appendBlock(out, *b.clut)
	}
	out = appendBlock(out, b.image)

	if b.cut != nil && *b.cut < len(out) {
		out = out[:*b.cut]
	}
	return out
}

// appendBlock encodes a block header and its data onto out.
func appendBlock(out []byte, b block) []byte {
	bnum := uint32(12 + b.width*b.height*2)
	if b.bnum != nil {
		bnum = *b.bnum
	}
	out = appendU32(out, bnum)
	out = appendU16(out, uint16(b.x))
	out = appendU16(out, uint16(b.y))
	out = appendU16(out, uint16(b.width))
	out = appendU16(out, uint16(b.height))
	return append(out, b.data...)
}

// appendU32 appends v as a little-endian 32-bit word.
func appendU32(out []byte, v uint32) []byte {
	return binary.LittleEndian.AppendUint32(out, v)
}

// appendU16 appends v as a little-endian 16-bit word.
func appendU16(out []byte, v uint16) []byte {
	return binary.LittleEndian.AppendUint16(out, v)
}
