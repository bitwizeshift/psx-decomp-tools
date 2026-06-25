package y4m

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
)

// Sentinel errors reported while encoding.
var (
	// ErrInvalidFormat indicates a [Format] whose dimensions are not positive and
	// even or whose frame rate is not positive.
	ErrInvalidFormat = errors.New("y4m: invalid format")

	// ErrFrameSize indicates a frame whose pixel dimensions differ from the stream
	// [Format].
	ErrFrameSize = errors.New("y4m: frame size mismatch")
)

// Format describes the frames of a YUV4MPEG2 stream.
type Format struct {
	// Width is the frame width in pixels; it must be even for 4:2:0 chroma.
	Width int

	// Height is the frame height in pixels; it must be even for 4:2:0 chroma.
	Height int

	// FrameRateNum and FrameRateDen are the numerator and denominator of the frame
	// rate in frames per second, such as 15 and 1.
	FrameRateNum int
	FrameRateDen int
}

// valid reports whether the format has even, positive dimensions and a positive
// frame rate.
func (f Format) valid() bool {
	return f.Width > 0 && f.Height > 0 && f.Width%2 == 0 && f.Height%2 == 0 &&
		f.FrameRateNum > 0 && f.FrameRateDen > 0
}

// Encoder writes a YUV4MPEG2 stream to an underlying writer. It emits the stream
// header before the first frame.
type Encoder struct {
	w       io.Writer
	format  Format
	started bool
}

// NewEncoder returns an [Encoder] that writes frames described by format to w. It
// reports [ErrInvalidFormat] for a format with non-positive or odd dimensions or
// a non-positive frame rate.
func NewEncoder(w io.Writer, format Format) (*Encoder, error) {
	if !format.valid() {
		return nil, ErrInvalidFormat
	}
	return &Encoder{w: w, format: format}, nil
}

// WriteFrame writes img as the next frame, emitting the stream header first when
// img is the first frame. It reports [ErrFrameSize] when img does not match the
// stream dimensions, or any error from writing to the underlying writer. An image
// that is not already 4:2:0 [image.YCbCr] is converted before writing.
func (e *Encoder) WriteFrame(img image.Image) error {
	bounds := img.Bounds()
	if bounds.Dx() != e.format.Width || bounds.Dy() != e.format.Height {
		return fmt.Errorf("y4m: %dx%d frame in %dx%d stream: %w",
			bounds.Dx(), bounds.Dy(), e.format.Width, e.format.Height, ErrFrameSize)
	}
	if !e.started {
		if err := e.writeStreamHeader(); err != nil {
			return err
		}
		e.started = true
	}
	if _, err := io.WriteString(e.w, "FRAME\n"); err != nil {
		return err
	}
	return e.writePlanes(e.toYCbCr(img))
}

// writeStreamHeader writes the YUV4MPEG2 header line that opens the stream.
func (e *Encoder) writeStreamHeader() error {
	header := fmt.Sprintf("YUV4MPEG2 W%d H%d F%d:%d Ip A1:1 C420jpeg\n",
		e.format.Width, e.format.Height, e.format.FrameRateNum, e.format.FrameRateDen)
	_, err := io.WriteString(e.w, header)
	return err
}

// writePlanes writes the Y, Cb, and Cr planes of yc in row order, dropping any
// stride padding.
func (e *Encoder) writePlanes(yc *image.YCbCr) error {
	b := yc.Bounds()
	w, h := b.Dx(), b.Dy()
	for y := range h {
		off := yc.YOffset(b.Min.X, b.Min.Y+y)
		if _, err := e.w.Write(yc.Y[off : off+w]); err != nil {
			return err
		}
	}
	cw, ch := (w+1)/2, (h+1)/2
	for _, plane := range [2][]byte{yc.Cb, yc.Cr} {
		for y := range ch {
			off := yc.COffset(b.Min.X, b.Min.Y+y*2)
			if _, err := e.w.Write(plane[off : off+cw]); err != nil {
				return err
			}
		}
	}
	return nil
}

// toYCbCr returns img as a 4:2:0 [image.YCbCr], reusing it when it is already in
// that form and converting it pixel by pixel otherwise.
func (e *Encoder) toYCbCr(img image.Image) *image.YCbCr {
	if yc, ok := img.(*image.YCbCr); ok && yc.SubsampleRatio == image.YCbCrSubsampleRatio420 {
		return yc
	}
	b := img.Bounds()
	yc := image.NewYCbCr(b, image.YCbCrSubsampleRatio420)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, blue, _ := img.At(x, y).RGBA()
			cy, cb, cr := color.RGBToYCbCr(uint8(r>>8), uint8(g>>8), uint8(blue>>8))
			yc.Y[yc.YOffset(x, y)] = cy
			yc.Cb[yc.COffset(x, y)] = cb
			yc.Cr[yc.COffset(x, y)] = cr
		}
	}
	return yc
}
