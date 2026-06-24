package tim

import "errors"

// Sentinel errors reported while decoding a TIM image.
var (
	// ErrBadMagic indicates a file whose leading id word is not the required
	// 0x10.
	ErrBadMagic = errors.New("tim: bad id word")

	// ErrBadMode indicates a flag word with reserved bits set or a pixel mode
	// outside the recognized range.
	ErrBadMode = errors.New("tim: invalid pixel mode")

	// ErrUnsupportedMode indicates a structurally valid image whose pixel mode
	// this package cannot turn into pixels, such as the mixed mode.
	ErrUnsupportedMode = errors.New("tim: unsupported pixel mode")

	// ErrTruncated indicates a read reached the end of the stream before a
	// complete header or block could be read.
	ErrTruncated = errors.New("tim: truncated image")

	// ErrCorrupt indicates a block whose declared byte count disagrees with its
	// width and height.
	ErrCorrupt = errors.New("tim: corrupt block")

	// ErrBadPalette indicates a palette index outside the range of the image's
	// CLUT.
	ErrBadPalette = errors.New("tim: palette index out of range")
)
