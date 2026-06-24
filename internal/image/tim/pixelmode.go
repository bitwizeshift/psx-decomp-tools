package tim

import "fmt"

// PixelMode is how a TIM stores its pixels, held in the low three bits of the
// file's flag word.
type PixelMode uint8

// The pixel modes a TIM flag word may select. Modes [Mode4bpp] and [Mode8bpp]
// store CLUT indices; [Mode16bpp] and [Mode24bpp] store direct colors;
// [ModeMixed] is recognized but not decodable.
const (
	Mode4bpp  PixelMode = 0 // 4-bit CLUT indices, four pixels per 16-bit word.
	Mode8bpp  PixelMode = 1 // 8-bit CLUT indices, two pixels per 16-bit word.
	Mode16bpp PixelMode = 2 // 16-bit direct ABGR1555 color, one pixel per word.
	Mode24bpp PixelMode = 3 // 24-bit direct RGB color, three bytes per pixel.
	ModeMixed PixelMode = 4 // Mixed-mode image; recognized but not decoded.
)

// String returns the conventional name of the mode, such as "4bpp".
func (m PixelMode) String() string {
	switch m {
	case Mode4bpp:
		return "4bpp"
	case Mode8bpp:
		return "8bpp"
	case Mode16bpp:
		return "16bpp"
	case Mode24bpp:
		return "24bpp"
	case ModeMixed:
		return "mixed"
	default:
		return fmt.Sprintf("PixelMode(%d)", uint8(m))
	}
}

// valid reports whether the mode is one a TIM flag word may legally carry.
func (m PixelMode) valid() bool {
	return m <= ModeMixed
}

// usesCLUT reports whether the mode stores indices into a CLUT rather than
// direct colors.
func (m PixelMode) usesCLUT() bool {
	return m == Mode4bpp || m == Mode8bpp
}

// pixelWidth returns the image width in pixels for a pixel block whose declared
// width is words 16-bit words.
func (m PixelMode) pixelWidth(words int) int {
	switch m {
	case Mode4bpp:
		return words * 4
	case Mode8bpp:
		return words * 2
	case Mode24bpp:
		return words * 2 / 3
	default:
		return words
	}
}
