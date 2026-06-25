package avi

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
)

// FrameEncoder compresses one decoded frame into the bytes of an AVI video chunk
// and names the codec they are stored under.
type FrameEncoder interface {
	// FourCC is the codec tag written into the stream header, such as "MPNG".
	FourCC() string

	// Encode returns img compressed as one frame.
	Encode(img image.Image) ([]byte, error)
}

// PNG returns a lossless [FrameEncoder] backed by the standard image/png encoder.
func PNG() FrameEncoder {
	return pngEncoder{}
}

// pngEncoder stores frames as PNG, the lossless default.
type pngEncoder struct{}

// FourCC returns the AVI codec tag for PNG frames.
func (pngEncoder) FourCC() string {
	return "MPNG"
}

// Encode compresses img as a PNG.
func (pngEncoder) Encode(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var _ FrameEncoder = pngEncoder{}

// MJPEG returns a lossy [FrameEncoder] backed by the standard image/jpeg encoder
// at the given quality (1..100).
func MJPEG(quality int) FrameEncoder {
	return mjpegEncoder{quality: quality}
}

// mjpegEncoder stores frames as JPEG, smaller but lossy.
type mjpegEncoder struct {
	quality int
}

// FourCC returns the AVI codec tag for Motion JPEG frames.
func (mjpegEncoder) FourCC() string {
	return "MJPG"
}

// Encode compresses img as a JPEG at the encoder's quality.
func (e mjpegEncoder) Encode(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: e.quality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var _ FrameEncoder = mjpegEncoder{}
