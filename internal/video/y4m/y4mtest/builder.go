package y4mtest

import (
	"bytes"
	"fmt"
	"image"
)

// SolidYCbCr returns a w by h 4:2:0 [image.YCbCr] with every luma and chroma
// sample set to the given values.
func SolidYCbCr(w, h int, y, cb, cr byte) *image.YCbCr {
	img := image.NewYCbCr(image.Rect(0, 0, w, h), image.YCbCrSubsampleRatio420)
	fill(img.Y, y)
	fill(img.Cb, cb)
	fill(img.Cr, cr)
	return img
}

// SolidPlanes returns the Y, Cb, and Cr plane bytes a w by h 4:2:0 frame of the
// given solid values encodes to, with no stride padding.
func SolidPlanes(w, h int, y, cb, cr byte) []byte {
	cw, ch := (w+1)/2, (h+1)/2
	out := bytes.Repeat([]byte{y}, w*h)
	out = append(out, bytes.Repeat([]byte{cb}, cw*ch)...)
	out = append(out, bytes.Repeat([]byte{cr}, cw*ch)...)
	return out
}

// Header returns the YUV4MPEG2 stream header line for the given geometry and
// frame rate.
func Header(w, h, num, den int) []byte {
	return fmt.Appendf(nil, "YUV4MPEG2 W%d H%d F%d:%d Ip A1:1 C420jpeg\n", w, h, num, den)
}

// Frame returns one FRAME record carrying the given plane bytes.
func Frame(planes []byte) []byte {
	return append([]byte("FRAME\n"), planes...)
}

// fill sets every byte of plane to value.
func fill(plane []byte, value byte) {
	for i := range plane {
		plane[i] = value
	}
}
