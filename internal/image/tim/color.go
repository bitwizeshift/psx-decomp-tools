package tim

import "image/color"

// abgr1555 converts a 16-bit TIM color into an 8-bit-per-channel color. Red is in
// bits 0-4, green in 5-9, and blue in 10-14. The all-zero value is fully
// transparent; every other value is opaque, the semi-transparency bit aside.
func abgr1555(u uint16) color.NRGBA {
	if u == 0 {
		return color.NRGBA{}
	}
	return color.NRGBA{
		R: expand5(u & 0x1f),
		G: expand5((u >> 5) & 0x1f),
		B: expand5((u >> 10) & 0x1f),
		A: 0xff,
	}
}

// expand5 scales a 5-bit channel value to the full 8-bit range, replicating the
// high bits into the low ones so 0x1f maps to 0xff.
func expand5(c uint16) uint8 {
	return uint8(c<<3 | c>>2)
}
