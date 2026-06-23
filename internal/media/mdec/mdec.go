package mdec

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
)

// HeaderSize is the length of the BS header that precedes a frame's bitstream.
const HeaderSize = 8

// FileID is the identifier word every BS frame header carries.
const FileID = 0x3800

// endOfFrame is the DC value that ends a frame's bitstream, sent in place of the
// next block's DC coefficient.
const endOfFrame = 0x1FF

// Sentinel errors reported while decoding.
var (
	// ErrInvalidHeader indicates a bitstream that does not begin with a valid BS
	// header.
	ErrInvalidHeader = errors.New("mdec: invalid header")

	// ErrUnsupportedVersion indicates a BS version this package does not decode.
	ErrUnsupportedVersion = errors.New("mdec: unsupported version")

	// ErrTruncated indicates a bitstream that ended before the frame was fully
	// decoded.
	ErrTruncated = errors.New("mdec: truncated bitstream")
)

// Header describes a BS frame: the version of its bitstream and the
// quantization scale applied to every block.
type Header struct {
	// Version is the BS bitstream version; this package decodes 1 and 2.
	Version int

	// Quant is the frame's quantization scale.
	Quant int

	// Size is the declared decompressed size in 4-byte units.
	Size int
}

// DecodeHeader reads the 8-byte BS header at the start of data. It reports
// [ErrInvalidHeader] when data is short or carries the wrong file ID.
func DecodeHeader(data []byte) (Header, error) {
	if len(data) < HeaderSize {
		return Header{}, fmt.Errorf("mdec: %d bytes: %w", len(data), ErrInvalidHeader)
	}
	if id := binary.LittleEndian.Uint16(data[2:4]); id != FileID {
		return Header{}, fmt.Errorf("mdec: id %#04x: %w", id, ErrInvalidHeader)
	}
	return Header{
		Size:    int(binary.LittleEndian.Uint16(data[0:2])),
		Quant:   int(binary.LittleEndian.Uint16(data[4:6])),
		Version: int(binary.LittleEndian.Uint16(data[6:8])),
	}, nil
}

// Decode decodes a BS frame of the given pixel dimensions into a 4:2:0
// [image.YCbCr]. The dimensions come from the STR sector header, as the
// bitstream itself records none. It reports [ErrInvalidHeader] for a malformed
// header, [ErrUnsupportedVersion] for a version other than 1 or 2, or
// [ErrTruncated] when the bitstream is exhausted early.
func Decode(data []byte, width, height int) (image.Image, error) {
	header, err := DecodeHeader(data)
	if err != nil {
		return nil, err
	}
	if header.Version != 1 && header.Version != 2 {
		return nil, fmt.Errorf("mdec: version %d: %w", header.Version, ErrUnsupportedVersion)
	}

	img := image.NewYCbCr(image.Rect(0, 0, width, height), image.YCbCrSubsampleRatio420)
	d := &decoder{reader: newBitReader(data[HeaderSize:]), quant: header.Quant, img: img}
	for x := 0; x < width; x += 16 {
		for y := 0; y < height; y += 16 {
			done, err := d.macroblock(x, y)
			if err != nil {
				return nil, err
			}
			if done {
				return img, nil
			}
		}
	}
	return img, nil
}

// decoder decodes the macroblocks of one frame into an image, carrying the bit
// reader and the frame's quantization scale.
type decoder struct {
	reader *bitReader
	quant  int
	img    *image.YCbCr
}

// macroblock decodes the six blocks of the macroblock whose top-left luma pixel
// is (px, py) and writes them into the image. It returns true once the
// end-of-frame code is reached.
func (d *decoder) macroblock(px, py int) (bool, error) {
	cr, done, err := d.block()
	if err != nil || done {
		return done, err
	}
	cb, done, err := d.block()
	if err != nil || done {
		return done, err
	}
	d.writeChroma(d.img.Cr, px, py, cr)
	d.writeChroma(d.img.Cb, px, py, cb)

	for i := range 4 {
		luma, done, err := d.block()
		if err != nil || done {
			return done, err
		}
		d.writeLuma(px+8*(i%2), py+8*(i/2), luma)
	}
	return false, nil
}

// block decodes one 8x8 block: its DC coefficient and run-length AC
// coefficients, dequantized and inverse-transformed into spatial samples. It
// returns true when the block's DC is the end-of-frame code.
func (d *decoder) block() ([64]int, bool, error) {
	dc := d.reader.read(10)
	if d.reader.err {
		return [64]int{}, false, ErrTruncated
	}
	if dc == endOfFrame {
		return [64]int{}, true, nil
	}

	var coeff [64]int
	scale := d.quant
	word := scale<<10 | dc&0x3FF
	value := signed10(word&0x3FF) * quantTable[0]
	scan := 0
	for {
		if scale == 0 {
			value = signed10(word&0x3FF) * 2
		}
		value = clamp(value, -0x400, 0x3FF)
		if scale > 0 {
			coeff[zagzig[scan]] = value
		} else {
			coeff[scan] = value
		}

		next := decodeAC(d.reader)
		if d.reader.err {
			return [64]int{}, false, ErrTruncated
		}
		if next == eobCode {
			break
		}
		word = int(next)
		scan += word>>10&0x3F + 1
		if scan > 63 {
			break
		}
		value = (signed10(word&0x3FF)*quantTable[scan]*scale + 4) / 8
	}
	return idct(&coeff), false, nil
}

// writeLuma writes an 8x8 luma block at top-left pixel (px, py), clipping to the
// image bounds so the padding rows of a non-16-aligned frame are dropped.
func (d *decoder) writeLuma(px, py int, block [64]int) {
	for y := range 8 {
		for x := range 8 {
			if px+x >= d.img.Rect.Max.X || py+y >= d.img.Rect.Max.Y {
				continue
			}
			d.img.Y[(py+y)*d.img.YStride+(px+x)] = clampByte(block[y*8+x] + 128)
		}
	}
}

// writeChroma writes an 8x8 chroma block covering the macroblock at luma pixel
// (px, py) into plane, clipping to the subsampled chroma bounds.
func (d *decoder) writeChroma(plane []byte, px, py int, block [64]int) {
	cx, cy := px/2, py/2
	cw, ch := (d.img.Rect.Max.X+1)/2, (d.img.Rect.Max.Y+1)/2
	for y := range 8 {
		for x := range 8 {
			if cx+x >= cw || cy+y >= ch {
				continue
			}
			plane[(cy+y)*d.img.CStride+(cx+x)] = clampByte(block[y*8+x] + 128)
		}
	}
}

// signed10 interprets the low 10 bits of v as a signed value.
func signed10(v int) int {
	if v&0x200 != 0 {
		return v - 0x400
	}
	return v
}

// clamp limits v to the inclusive range [lo, hi].
func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// clampByte limits v to the range of a byte.
func clampByte(v int) byte {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return byte(v)
}
