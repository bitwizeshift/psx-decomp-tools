package timtest_test

import (
	"bytes"
	"image/color"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/bitwizeshift/psx-decomp-tools/internal/image/tim"
	"github.com/bitwizeshift/psx-decomp-tools/internal/image/tim/timtest"
)

// The colors used by the fixtures, expanded from ABGR1555 to 8-bit channels.
var (
	red  = color.NRGBA{R: 0xff, A: 0xff}
	blue = color.NRGBA{B: 0xff, A: 0xff}
)

func TestNew(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		opts    []timtest.Option
		want    *tim.File
		wantErr error
	}{
		{
			name: "DefaultIsValid",
			opts: nil,
			want: &tim.File{Mode: tim.Mode16bpp, Width: 1, Height: 1},
		}, {
			name: "FullyConfigured",
			opts: []timtest.Option{
				timtest.Mode(tim.Mode8bpp),
				timtest.PaletteCoords(3, 4),
				timtest.Palette(2, 1, timtest.U16s(0x001F, 0x7C00)),
				timtest.ImageCoords(5, 6),
				timtest.Pixels(1, 1, []byte{0x00, 0x01}),
			},
			want: &tim.File{Mode: tim.Mode8bpp, PalX: 3, PalY: 4, ImgX: 5, ImgY: 6, Width: 2, Height: 1, Palettes: []color.Palette{{red, blue}}},
		}, {
			name:    "BadID",
			opts:    []timtest.Option{timtest.ID(0x20)},
			wantErr: tim.ErrBadMagic,
		}, {
			name:    "RawFlag",
			opts:    []timtest.Option{timtest.Flag(0x10)},
			wantErr: tim.ErrBadMode,
		}, {
			name:    "Truncated",
			opts:    []timtest.Option{timtest.Truncate(4)},
			wantErr: tim.ErrTruncated,
		}, {
			name:    "CorruptPixelBlock",
			opts:    []timtest.Option{timtest.PixelByteCount(99)},
			wantErr: tim.ErrCorrupt,
		}, {
			name:    "CorruptClutBlock",
			opts:    []timtest.Option{timtest.Mode(tim.Mode8bpp), timtest.PaletteByteCount(99), timtest.Palette(2, 1, timtest.U16s(0, 0))},
			wantErr: tim.ErrCorrupt,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			data := timtest.New(tc.opts...)

			// Act
			file, err := tim.Parse(bytes.NewReader(data))

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Parse() error = %v, want %v", got, want)
			}
			if got, want := file, tc.want; !cmp.Equal(got, want, cmpopts.IgnoreUnexported(tim.File{})) {
				t.Errorf("Parse() file mismatch (-want +got):\n%s", cmp.Diff(want, got, cmpopts.IgnoreUnexported(tim.File{})))
			}
		})
	}
}

func TestU16s(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		values []uint16
		want   []byte
	}{
		{
			name:   "Empty",
			values: nil,
			want:   []byte{},
		}, {
			name:   "LittleEndian",
			values: []uint16{0x0102, 0xABCD},
			want:   []byte{0x02, 0x01, 0xCD, 0xAB},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			values := tc.values

			// Act
			got := timtest.U16s(values...)

			// Assert
			if want := tc.want; !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
				t.Errorf("U16s() mismatch (-want +got):\n%s", cmp.Diff(want, got, cmpopts.EquateEmpty()))
			}
		})
	}
}
