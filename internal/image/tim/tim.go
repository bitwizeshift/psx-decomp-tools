package tim

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"io"
)

const (
	// fileHeaderSize is the size in bytes of the file header: the id word and the
	// flag word.
	fileHeaderSize = 8

	// blockHeaderSize is the size in bytes of the header that precedes a CLUT or
	// pixel block: a byte count, two framebuffer coordinates, a width, and a
	// height.
	blockHeaderSize = 12

	// timID is the value of the id word that begins every TIM file.
	timID = 0x10

	// clutFlag is the bit set in the flag word when a CLUT block follows the file
	// header.
	clutFlag = 0x08
)

// timMagic is the four little-endian bytes of the id word that begin every TIM.
var timMagic = []byte{timID, 0x00, 0x00, 0x00}

func init() {
	image.RegisterFormat("tim", string(timMagic), Decode, DecodeConfig)
}

// File is a decoded TIM image: its pixel mode, framebuffer placement, decoded
// dimensions, and any color lookup tables. Build one with [Parse].
type File struct {
	// Mode is how the pixels are stored.
	Mode PixelMode

	// ImgX and ImgY are the framebuffer coordinates of the pixel block.
	ImgX, ImgY int

	// PalX and PalY are the framebuffer coordinates of the CLUT block, zero when
	// the image has no CLUT.
	PalX, PalY int

	// Width and Height are the image's dimensions in pixels.
	Width, Height int

	// Palettes holds one [color.Palette] per CLUT row, or nil for a direct color
	// image.
	Palettes []color.Palette

	pixels []byte
	stride int
	size   int
}

// Parse reads a TIM image from r. It returns [ErrBadMagic] or [ErrBadMode] for a
// header that is not a TIM, [ErrCorrupt] for a block whose size is inconsistent,
// and [ErrTruncated] when the stream ends early.
func Parse(r io.Reader) (*File, error) {
	mode, hasCLUT, err := readFileHeader(r)
	if err != nil {
		return nil, err
	}

	f := &File{Mode: mode, size: fileHeaderSize}
	if hasCLUT {
		head, data, err := readBlock(r)
		if err != nil {
			return nil, err
		}
		f.PalX, f.PalY = head.x, head.y
		f.Palettes = decodePalettes(head, data)
		f.size += blockHeaderSize + len(data)
	}

	head, data, err := readBlock(r)
	if err != nil {
		return nil, err
	}
	f.ImgX, f.ImgY = head.x, head.y
	f.Width = mode.pixelWidth(head.width)
	f.Height = head.height
	f.pixels = data
	f.stride = head.width * 2
	f.size += blockHeaderSize + len(data)
	return f, nil
}

// EncodedSize returns the number of bytes the TIM occupies in its source stream:
// the file header, the optional CLUT block, and the pixel block.
func (f *File) EncodedSize() int {
	return f.size
}

// Located is a TIM found within a buffer, paired with the byte offset at which it
// begins.
type Located struct {
	// Offset is the byte position of the TIM within the buffer passed to
	// [DecodeAll].
	Offset int

	// File is the parsed TIM.
	File *File
}

// DecodeAll reads the run of consecutive TIMs at the front of data. It first skips
// a leading run of zero padding, then parses TIMs back to back for as long as the
// bytes begin with the TIM id word; the first bytes that do not end the run. It
// returns [ErrCorrupt] or [ErrTruncated] when a TIM begins but is malformed, and no
// images with no error when data holds none. Only leading zero padding is skipped;
// a non-zero region that does not begin a TIM ends the run rather than being
// skipped.
func DecodeAll(data []byte) ([]Located, error) {
	offset := 0
	for offset < len(data) && data[offset] == 0 {
		offset++
	}

	var located []Located
	for bytes.HasPrefix(data[offset:], timMagic) {
		file, err := Parse(bytes.NewReader(data[offset:]))
		if err != nil {
			return located, err
		}
		located = append(located, Located{Offset: offset, File: file})
		offset += file.EncodedSize()
	}
	return located, nil
}

// NumPalettes returns the number of CLUTs in the image, zero for a direct color
// image.
func (f *File) NumPalettes() int {
	return len(f.Palettes)
}

// Image renders the image using the CLUT at the given palette index, which is
// ignored for direct color images. CLUT modes yield an [image.Paletted] and
// direct modes an [image.NRGBA]. It returns [ErrBadPalette] for an out-of-range
// index and [ErrUnsupportedMode] for a mode that cannot be rendered.
func (f *File) Image(palette int) (image.Image, error) {
	switch f.Mode {
	case Mode4bpp, Mode8bpp:
		img, err := f.paletted(palette)
		if err != nil {
			return nil, err
		}
		return img, nil
	case Mode16bpp:
		return f.direct16(), nil
	case Mode24bpp:
		return f.direct24(), nil
	default:
		return nil, ErrUnsupportedMode
	}
}

// paletted renders a CLUT image against the palette at index.
func (f *File) paletted(index int) (*image.Paletted, error) {
	if index < 0 || index >= len(f.Palettes) {
		return nil, ErrBadPalette
	}
	img := image.NewPaletted(image.Rect(0, 0, f.Width, f.Height), f.Palettes[index])
	for y := range f.Height {
		row := f.pixels[y*f.stride:]
		for x := range f.Width {
			img.SetColorIndex(x, y, f.indexAt(row, x))
		}
	}
	return img, nil
}

// indexAt returns the CLUT index of pixel x within a pixel row.
func (f *File) indexAt(row []byte, x int) uint8 {
	if f.Mode == Mode4bpp {
		b := row[x/2]
		if x&1 == 0 {
			return b & 0x0f
		}
		return b >> 4
	}
	return row[x]
}

// direct16 renders a 16-bit ABGR1555 image.
func (f *File) direct16() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, f.Width, f.Height))
	for y := range f.Height {
		row := f.pixels[y*f.stride:]
		for x := range f.Width {
			img.SetNRGBA(x, y, abgr1555(binary.LittleEndian.Uint16(row[x*2:])))
		}
	}
	return img
}

// direct24 renders a 24-bit RGB image.
func (f *File) direct24() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, f.Width, f.Height))
	for y := range f.Height {
		row := f.pixels[y*f.stride:]
		for x := range f.Width {
			i := x * 3
			img.SetNRGBA(x, y, color.NRGBA{R: row[i], G: row[i+1], B: row[i+2], A: 0xff})
		}
	}
	return img
}

// Decode reads a TIM image from r and renders it using palette zero. It satisfies
// the decode function of [image.RegisterFormat].
func Decode(r io.Reader) (image.Image, error) {
	f, err := Parse(r)
	if err != nil {
		return nil, err
	}
	return f.Image(0)
}

// DecodeConfig reads a TIM header from r and returns its color model and
// dimensions without rendering the pixels. It satisfies the config function of
// [image.RegisterFormat].
func DecodeConfig(r io.Reader) (image.Config, error) {
	mode, hasCLUT, err := readFileHeader(r)
	if err != nil {
		return image.Config{}, err
	}

	var palette color.Palette
	if hasCLUT {
		head, data, err := readBlock(r)
		if err != nil {
			return image.Config{}, err
		}
		if palettes := decodePalettes(head, data); len(palettes) > 0 {
			palette = palettes[0]
		}
	}

	head, err := readBlockHeader(r)
	if err != nil {
		return image.Config{}, err
	}
	config := image.Config{Width: mode.pixelWidth(head.width), Height: head.height}
	if mode.usesCLUT() {
		config.ColorModel = palette
	} else {
		config.ColorModel = color.NRGBAModel
	}
	return config, nil
}

// readFileHeader reads the file header and reports the pixel mode and whether a
// CLUT block follows.
func readFileHeader(r io.Reader) (PixelMode, bool, error) {
	var head [fileHeaderSize]byte
	if _, err := io.ReadFull(r, head[:]); err != nil {
		return 0, false, truncated(err)
	}
	if binary.LittleEndian.Uint32(head[0:4]) != timID {
		return 0, false, ErrBadMagic
	}
	flag := binary.LittleEndian.Uint32(head[4:8])
	if flag>>4 != 0 {
		return 0, false, ErrBadMode
	}
	mode := PixelMode(flag & 0x7)
	if !mode.valid() {
		return 0, false, ErrBadMode
	}
	return mode, flag&clutFlag != 0, nil
}

// blockHeader is the 12-byte header of a CLUT or pixel block.
type blockHeader struct {
	x, y   int
	width  int
	height int
}

// readBlockHeader reads and validates a block header, returning [ErrCorrupt] when
// its byte count disagrees with its width and height.
func readBlockHeader(r io.Reader) (blockHeader, error) {
	var head [blockHeaderSize]byte
	if _, err := io.ReadFull(r, head[:]); err != nil {
		return blockHeader{}, truncated(err)
	}
	bnum := binary.LittleEndian.Uint32(head[0:4])
	h := blockHeader{
		x:      int(binary.LittleEndian.Uint16(head[4:6])),
		y:      int(binary.LittleEndian.Uint16(head[6:8])),
		width:  int(binary.LittleEndian.Uint16(head[8:10])),
		height: int(binary.LittleEndian.Uint16(head[10:12])),
	}
	if int64(bnum) != int64(blockHeaderSize+h.width*h.height*2) {
		return blockHeader{}, ErrCorrupt
	}
	return h, nil
}

// readBlock reads a block header and its trailing 16-bit words.
func readBlock(r io.Reader) (blockHeader, []byte, error) {
	head, err := readBlockHeader(r)
	if err != nil {
		return blockHeader{}, nil, err
	}
	data := make([]byte, head.width*head.height*2)
	if _, err := io.ReadFull(r, data); err != nil {
		return blockHeader{}, nil, truncated(err)
	}
	return head, data, nil
}

// decodePalettes splits a CLUT block into one [color.Palette] per row.
func decodePalettes(head blockHeader, data []byte) []color.Palette {
	palettes := make([]color.Palette, head.height)
	for i := range palettes {
		palette := make(color.Palette, head.width)
		row := data[i*head.width*2:]
		for j := range palette {
			palette[j] = abgr1555(binary.LittleEndian.Uint16(row[j*2:]))
		}
		palettes[i] = palette
	}
	return palettes
}

// truncated maps the end-of-stream errors from [io.ReadFull] to [ErrTruncated]
// and passes any other error through unchanged.
func truncated(err error) error {
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return ErrTruncated
	}
	return err
}
