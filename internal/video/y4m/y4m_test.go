package y4m_test

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"io"
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/testing/iotest"
	"github.com/bitwizeshift/psx-decomp-tools/internal/video/y4m"
	"github.com/bitwizeshift/psx-decomp-tools/internal/video/y4m/y4mtest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestNewEncoderInvalidFormat(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		format y4m.Format
	}{
		{
			name:   "OddWidth",
			format: y4m.Format{Width: 3, Height: 2, FrameRateNum: 15, FrameRateDen: 1},
		}, {
			name:   "OddHeight",
			format: y4m.Format{Width: 4, Height: 3, FrameRateNum: 15, FrameRateDen: 1},
		}, {
			name:   "ZeroHeight",
			format: y4m.Format{Width: 4, Height: 0, FrameRateNum: 15, FrameRateDen: 1},
		}, {
			name:   "ZeroRate",
			format: y4m.Format{Width: 4, Height: 2, FrameRateNum: 0, FrameRateDen: 1},
		}, {
			name:   "ZeroDenominator",
			format: y4m.Format{Width: 4, Height: 2, FrameRateNum: 15, FrameRateDen: 0},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			buf := &bytes.Buffer{}

			// Act
			got, err := y4m.NewEncoder(buf, tc.format)

			// Assert
			if got, want := err, y4m.ErrInvalidFormat; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("NewEncoder() error = %v, want %v", got, want)
			}
			if got, want := got, (*y4m.Encoder)(nil); got != want {
				t.Errorf("NewEncoder() = %v, want %v", got, want)
			}
		})
	}
}

func TestEncoderWriteFrame(t *testing.T) {
	t.Parallel()

	writeErr := errors.New("write failed")
	format := y4m.Format{Width: 4, Height: 2, FrameRateNum: 15, FrameRateDen: 1}
	header := y4mtest.Header(4, 2, 15, 1)
	rgbaY, rgbaCb, rgbaCr := color.RGBToYCbCr(200, 100, 50)

	testCases := []struct {
		name    string
		writer  io.Writer
		frames  []image.Image
		want    []byte
		wantErr error
	}{
		{
			name:   "SingleYCbCrFrame",
			writer: &bytes.Buffer{},
			frames: []image.Image{y4mtest.SolidYCbCr(4, 2, 16, 128, 200)},
			want:   concat(header, y4mtest.Frame(y4mtest.SolidPlanes(4, 2, 16, 128, 200))),
		}, {
			name:   "TwoFramesShareHeader",
			writer: &bytes.Buffer{},
			frames: []image.Image{
				y4mtest.SolidYCbCr(4, 2, 16, 128, 200),
				y4mtest.SolidYCbCr(4, 2, 32, 64, 96),
			},
			want: concat(
				header,
				y4mtest.Frame(y4mtest.SolidPlanes(4, 2, 16, 128, 200)),
				y4mtest.Frame(y4mtest.SolidPlanes(4, 2, 32, 64, 96)),
			),
		}, {
			name:   "ConvertsRGBA",
			writer: &bytes.Buffer{},
			frames: []image.Image{solidRGBA(4, 2, color.RGBA{R: 200, G: 100, B: 50, A: 255})},
			want:   concat(header, y4mtest.Frame(y4mtest.SolidPlanes(4, 2, rgbaY, rgbaCb, rgbaCr))),
		}, {
			name:    "FrameSizeMismatch",
			writer:  &bytes.Buffer{},
			frames:  []image.Image{y4mtest.SolidYCbCr(2, 2, 16, 128, 200)},
			want:    nil,
			wantErr: y4m.ErrFrameSize,
		}, {
			name:    "HeaderWriteFails",
			writer:  iotest.TruncateErrWriter(0, writeErr),
			frames:  []image.Image{y4mtest.SolidYCbCr(4, 2, 16, 128, 200)},
			want:    nil,
			wantErr: writeErr,
		}, {
			name:    "PlaneWriteFails",
			writer:  iotest.TruncateErrWriter(len(header)+len("FRAME\n"), writeErr),
			frames:  []image.Image{y4mtest.SolidYCbCr(4, 2, 16, 128, 200)},
			want:    nil,
			wantErr: writeErr,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			encoder := newEncoder(t, tc.writer, format)

			// Act
			err := writeFrames(encoder, tc.frames)
			got := iotest.ReadAllFromWriter(t, tc.writer)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("WriteFrame() error = %v, want %v", got, want)
			}
			if got, want := got, tc.want; !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
				t.Errorf("WriteFrame() diff (-got +want):\n%s", cmp.Diff(got, want, cmpopts.EquateEmpty()))
			}
		})
	}
}

// newEncoder builds an [y4m.Encoder], failing the test if the format is rejected.
func newEncoder(t *testing.T, w io.Writer, format y4m.Format) *y4m.Encoder {
	t.Helper()
	encoder, err := y4m.NewEncoder(w, format)
	if err != nil {
		t.Fatalf("NewEncoder() error = %v", err)
	}
	return encoder
}

// writeFrames writes each frame in order, returning the first error.
func writeFrames(e *y4m.Encoder, frames []image.Image) error {
	for _, frame := range frames {
		if err := e.WriteFrame(frame); err != nil {
			return err
		}
	}
	return nil
}

// solidRGBA returns a w by h RGBA image filled with c.
func solidRGBA(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

// concat joins byte slices into one.
func concat(parts ...[]byte) []byte {
	return bytes.Join(parts, nil)
}
