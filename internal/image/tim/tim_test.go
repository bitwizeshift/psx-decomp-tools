package tim_test

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/bitwizeshift/psx-decomp-tools/internal/image/tim"
	"github.com/bitwizeshift/psx-decomp-tools/internal/image/tim/timtest"
)

// The colors the ABGR1555 test fixtures encode, expanded to 8-bit channels.
var (
	red   = color.NRGBA{R: 0xff, A: 0xff}
	green = color.NRGBA{G: 0xff, A: 0xff}
	blue  = color.NRGBA{B: 0xff, A: 0xff}
	clear = color.NRGBA{}
)

// errReader fails every read with its error.
type errReader struct {
	err error
}

func (r errReader) Read([]byte) (int, error) {
	return 0, r.err
}

// nrgba builds an NRGBA image of the given size from row-major colors.
func nrgba(width, height int, colors ...color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for i, c := range colors {
		img.SetNRGBA(i%width, i/width, c)
	}
	return img
}

// paletted builds a paletted image of the given size from row-major indices.
func paletted(width, height int, palette color.Palette, indices ...uint8) *image.Paletted {
	img := image.NewPaletted(image.Rect(0, 0, width, height), palette)
	for i, idx := range indices {
		img.SetColorIndex(i%width, i/width, idx)
	}
	return img
}

// mustParse parses data into a [tim.File], failing the test on any error.
func mustParse(t *testing.T, data []byte) *tim.File {
	t.Helper()
	file, err := tim.Parse(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	return file
}

func TestParse(t *testing.T) {
	t.Parallel()

	testErr := errors.New("test error")
	testCases := []struct {
		name    string
		reader  io.Reader
		want    *tim.File
		wantErr error
	}{
		{
			name: "Direct16",
			reader: bytes.NewReader(
				timtest.New(
					timtest.Mode(tim.Mode16bpp),
					timtest.ImageCoords(10, 20),
					timtest.Pixels(2, 1, timtest.U16s(0x7C00, 0x0000)),
				),
			),
			want: &tim.File{
				Mode:   tim.Mode16bpp,
				ImgX:   10,
				ImgY:   20,
				Width:  2,
				Height: 1,
			},
		}, {
			name: "Direct24",
			reader: bytes.NewReader(
				timtest.New(
					timtest.Mode(tim.Mode24bpp),
					timtest.Pixels(3, 1, []byte{1, 2, 3, 4, 5, 6}),
				),
			),
			want: &tim.File{
				Mode:   tim.Mode24bpp,
				Width:  2,
				Height: 1,
			},
		}, {
			name: "Indexed8",
			reader: bytes.NewReader(
				timtest.New(
					timtest.Mode(tim.Mode8bpp),
					timtest.PaletteCoords(5, 6),
					timtest.Palette(2, 1, timtest.U16s(0x001F, 0x7C00)),
					timtest.Pixels(1, 1, []byte{0x00, 0x01}),
				),
			),
			want: &tim.File{
				Mode:     tim.Mode8bpp,
				PalX:     5,
				PalY:     6,
				Width:    2,
				Height:   1,
				Palettes: []color.Palette{{red, blue}},
			},
		}, {
			name: "Indexed4MultiPalette",
			reader: bytes.NewReader(
				timtest.New(
					timtest.Mode(tim.Mode4bpp),
					timtest.Palette(2, 2, timtest.U16s(0x001F, 0x03E0, 0x7C00, 0x0000)),
					timtest.Pixels(1, 1, []byte{0x10, 0x00}),
				),
			),
			want: &tim.File{
				Mode:     tim.Mode4bpp,
				Width:    4,
				Height:   1,
				Palettes: []color.Palette{{red, green}, {blue, clear}},
			},
		}, {
			name: "Mixed",
			reader: bytes.NewReader(
				timtest.New(
					timtest.Mode(tim.ModeMixed),
					timtest.Pixels(1, 1, []byte{0x00, 0x00}),
				),
			),
			want: &tim.File{
				Mode:   tim.ModeMixed,
				Width:  1,
				Height: 1,
			},
		}, {
			name:    "BadMagic",
			reader:  bytes.NewReader(timtest.New(timtest.ID(0x20))),
			wantErr: tim.ErrBadMagic,
		}, {
			name:    "ReservedFlagBits",
			reader:  bytes.NewReader(timtest.New(timtest.Flag(0x10))),
			wantErr: tim.ErrBadMode,
		}, {
			name:    "UnknownMode",
			reader:  bytes.NewReader(timtest.New(timtest.Flag(0x05))),
			wantErr: tim.ErrBadMode,
		}, {
			name:    "EmptyHeader",
			reader:  bytes.NewReader(timtest.New(timtest.Truncate(0))),
			wantErr: tim.ErrTruncated,
		}, {
			name:    "ShortHeader",
			reader:  bytes.NewReader(timtest.New(timtest.Truncate(4))),
			wantErr: tim.ErrTruncated,
		}, {
			name: "TruncatedPixels",
			reader: bytes.NewReader(
				timtest.New(
					timtest.Mode(tim.Mode16bpp),
					timtest.Pixels(2, 1, timtest.U16s(0, 0)),
					timtest.Truncate(20),
				),
			),
			wantErr: tim.ErrTruncated,
		}, {
			name:    "CorruptPixelBlock",
			reader:  bytes.NewReader(timtest.New(timtest.PixelByteCount(99))),
			wantErr: tim.ErrCorrupt,
		}, {
			name: "TruncatedClut",
			reader: bytes.NewReader(
				timtest.New(
					timtest.Mode(tim.Mode8bpp),
					timtest.Palette(2, 1, timtest.U16s(0, 0)),
					timtest.Truncate(14),
				),
			),
			wantErr: tim.ErrTruncated,
		}, {
			name: "CorruptClutBlock",
			reader: bytes.NewReader(
				timtest.New(
					timtest.Mode(tim.Mode8bpp),
					timtest.PaletteByteCount(99),
					timtest.Palette(2, 1, timtest.U16s(0, 0)),
				),
			),
			wantErr: tim.ErrCorrupt,
		}, {
			name:    "ReadError",
			reader:  errReader{err: testErr},
			wantErr: testErr,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			reader := tc.reader

			// Act
			file, err := tim.Parse(reader)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Parse() error = %v, want %v", got, want)
			}
			opts := cmpopts.IgnoreUnexported(tim.File{})
			if got, want := file, tc.want; !cmp.Equal(got, want, opts) {
				t.Errorf("Parse() file mismatch (-want +got):\n%s", cmp.Diff(want, got, opts))
			}
		})
	}
}

func TestFileNumPalettes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		data []byte
		want int
	}{
		{
			name: "NoPalette",
			data: timtest.New(
				timtest.Mode(tim.Mode16bpp),
				timtest.Pixels(1, 1, timtest.U16s(0)),
			),
			want: 0,
		}, {
			name: "TwoPalettes",
			data: timtest.New(
				timtest.Mode(tim.Mode4bpp),
				timtest.Palette(2, 2, timtest.U16s(0, 0, 0, 0)),
				timtest.Pixels(1, 1, []byte{0, 0}),
			),
			want: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			file := mustParse(t, tc.data)

			// Act
			got := file.NumPalettes()

			// Assert
			if want := tc.want; got != want {
				t.Errorf("NumPalettes() = %d, want %d", got, want)
			}
		})
	}
}

func TestFileImage(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		data    []byte
		palette int
		want    image.Image
		wantErr error
	}{
		{
			name:    "Direct16Transparency",
			data:    timtest.New(timtest.Mode(tim.Mode16bpp), timtest.Pixels(2, 1, timtest.U16s(0x7C00, 0x0000))),
			want:    nrgba(2, 1, blue, clear),
			wantErr: nil,
		}, {
			name:    "Direct24",
			data:    timtest.New(timtest.Mode(tim.Mode24bpp), timtest.Pixels(3, 1, []byte{1, 2, 3, 4, 5, 6})),
			want:    nrgba(2, 1, color.NRGBA{R: 1, G: 2, B: 3, A: 0xff}, color.NRGBA{R: 4, G: 5, B: 6, A: 0xff}),
			wantErr: nil,
		}, {
			name:    "Indexed8",
			data:    timtest.New(timtest.Mode(tim.Mode8bpp), timtest.Palette(2, 1, timtest.U16s(0x001F, 0x7C00)), timtest.Pixels(1, 1, []byte{0x00, 0x01})),
			want:    paletted(2, 1, color.Palette{red, blue}, 0, 1),
			wantErr: nil,
		}, {
			name:    "Indexed4Nibbles",
			data:    timtest.New(timtest.Mode(tim.Mode4bpp), timtest.Palette(2, 1, timtest.U16s(0x001F, 0x03E0)), timtest.Pixels(1, 1, []byte{0x10, 0x00})),
			want:    paletted(4, 1, color.Palette{red, green}, 0, 1, 0, 0),
			wantErr: nil,
		}, {
			name:    "Indexed4SecondPalette",
			data:    timtest.New(timtest.Mode(tim.Mode4bpp), timtest.Palette(2, 2, timtest.U16s(0x001F, 0x03E0, 0x7C00, 0x0000)), timtest.Pixels(1, 1, []byte{0x10, 0x00})),
			palette: 1,
			want:    paletted(4, 1, color.Palette{blue, clear}, 0, 1, 0, 0),
			wantErr: nil,
		}, {
			name:    "PaletteOutOfRange",
			data:    timtest.New(timtest.Mode(tim.Mode8bpp), timtest.Palette(2, 1, timtest.U16s(0, 0)), timtest.Pixels(1, 1, []byte{0, 0})),
			palette: 2,
			want:    nil,
			wantErr: tim.ErrBadPalette,
		}, {
			name:    "PaletteNegative",
			data:    timtest.New(timtest.Mode(tim.Mode8bpp), timtest.Palette(2, 1, timtest.U16s(0, 0)), timtest.Pixels(1, 1, []byte{0, 0})),
			palette: -1,
			want:    nil,
			wantErr: tim.ErrBadPalette,
		}, {
			name:    "UnsupportedMode",
			data:    timtest.New(timtest.Mode(tim.ModeMixed), timtest.Pixels(1, 1, []byte{0, 0})),
			want:    nil,
			wantErr: tim.ErrUnsupportedMode,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			file := mustParse(t, tc.data)

			// Act
			img, err := file.Image(tc.palette)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Image() error = %v, want %v", got, want)
			}
			if got, want := img, tc.want; !cmp.Equal(got, want) {
				t.Errorf("Image() mismatch (-want +got):\n%s", cmp.Diff(want, got))
			}
		})
	}
}

func TestDecode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		reader  io.Reader
		want    image.Image
		wantErr error
	}{
		{
			name:    "Direct16",
			reader:  bytes.NewReader(timtest.New(timtest.Mode(tim.Mode16bpp), timtest.Pixels(1, 1, timtest.U16s(0x001F)))),
			want:    nrgba(1, 1, red),
			wantErr: nil,
		}, {
			name:    "Indexed8Palette0",
			reader:  bytes.NewReader(timtest.New(timtest.Mode(tim.Mode8bpp), timtest.Palette(2, 1, timtest.U16s(0x001F, 0x7C00)), timtest.Pixels(1, 1, []byte{0x01, 0x00}))),
			want:    paletted(2, 1, color.Palette{red, blue}, 1, 0),
			wantErr: nil,
		}, {
			name:    "BadMagic",
			reader:  bytes.NewReader(timtest.New(timtest.ID(0x20))),
			want:    nil,
			wantErr: tim.ErrBadMagic,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			reader := tc.reader

			// Act
			img, err := tim.Decode(reader)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Decode() error = %v, want %v", got, want)
			}
			if got, want := img, tc.want; !cmp.Equal(got, want) {
				t.Errorf("Decode() mismatch (-want +got):\n%s", cmp.Diff(want, got))
			}
		})
	}
}

// equateColorModel compares CLUT color models by their palette and the standard
// models by identity, which a plain cmp cannot do for the function-backed models.
func equateColorModel() cmp.Option {
	return cmp.Comparer(func(lhs, rhs color.Model) bool {
		lp, lok := lhs.(color.Palette)
		rp, rok := rhs.(color.Palette)
		if lok != rok {
			return false
		}
		if lok {
			return cmp.Equal(lp, rp)
		}
		return lhs == rhs
	})
}

func TestDecodeConfig(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		reader  io.Reader
		want    image.Config
		wantErr error
	}{
		{
			name:    "Direct16",
			reader:  bytes.NewReader(timtest.New(timtest.Mode(tim.Mode16bpp), timtest.Pixels(3, 2, make([]byte, 12)))),
			want:    image.Config{ColorModel: color.NRGBAModel, Width: 3, Height: 2},
			wantErr: nil,
		}, {
			name:    "Mixed",
			reader:  bytes.NewReader(timtest.New(timtest.Mode(tim.ModeMixed), timtest.Pixels(1, 1, []byte{0, 0}))),
			want:    image.Config{ColorModel: color.NRGBAModel, Width: 1, Height: 1},
			wantErr: nil,
		}, {
			name:    "Indexed8",
			reader:  bytes.NewReader(timtest.New(timtest.Mode(tim.Mode8bpp), timtest.Palette(2, 1, timtest.U16s(0x001F, 0x7C00)), timtest.Pixels(1, 1, []byte{0, 0}))),
			want:    image.Config{ColorModel: color.Palette{red, blue}, Width: 2, Height: 1},
			wantErr: nil,
		}, {
			name:    "EmptyPalette",
			reader:  bytes.NewReader(timtest.New(timtest.Mode(tim.Mode8bpp), timtest.Palette(0, 0, nil), timtest.Pixels(1, 1, []byte{0, 0}))),
			want:    image.Config{ColorModel: color.Palette(nil), Width: 2, Height: 1},
			wantErr: nil,
		}, {
			name:    "BadMagic",
			reader:  bytes.NewReader(timtest.New(timtest.ID(0x20))),
			want:    image.Config{},
			wantErr: tim.ErrBadMagic,
		}, {
			name:    "TruncatedImageHeader",
			reader:  bytes.NewReader(timtest.New(timtest.Truncate(10))),
			want:    image.Config{},
			wantErr: tim.ErrTruncated,
		}, {
			name:    "CorruptImageHeader",
			reader:  bytes.NewReader(timtest.New(timtest.PixelByteCount(99))),
			want:    image.Config{},
			wantErr: tim.ErrCorrupt,
		}, {
			name:    "TruncatedClut",
			reader:  bytes.NewReader(timtest.New(timtest.Mode(tim.Mode8bpp), timtest.Palette(2, 1, timtest.U16s(0, 0)), timtest.Truncate(14))),
			want:    image.Config{},
			wantErr: tim.ErrTruncated,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			reader := tc.reader

			// Act
			config, err := tim.DecodeConfig(reader)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("DecodeConfig() error = %v, want %v", got, want)
			}
			if got, want := config, tc.want; !cmp.Equal(got, want, equateColorModel()) {
				t.Errorf("DecodeConfig() mismatch (-want +got):\n%s", cmp.Diff(want, got, equateColorModel()))
			}
		})
	}
}

func TestImageDecodeRegistered(t *testing.T) {
	t.Parallel()

	// Arrange
	reader := bytes.NewReader(timtest.New(timtest.Mode(tim.Mode16bpp), timtest.Pixels(1, 1, timtest.U16s(0x001F))))

	// Act
	img, format, err := image.Decode(reader)

	// Assert
	if err != nil {
		t.Fatalf("image.Decode() error = %v, want nil", err)
	}
	if got, want := format, "tim"; got != want {
		t.Errorf("image.Decode() format = %q, want %q", got, want)
	}
	if got, want := img, image.Image(nrgba(1, 1, red)); !cmp.Equal(got, want) {
		t.Errorf("image.Decode() mismatch (-want +got):\n%s", cmp.Diff(want, got))
	}
}

func TestPixelModeString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		mode tim.PixelMode
		want string
	}{
		{name: "4bpp", mode: tim.Mode4bpp, want: "4bpp"},
		{name: "8bpp", mode: tim.Mode8bpp, want: "8bpp"},
		{name: "16bpp", mode: tim.Mode16bpp, want: "16bpp"},
		{name: "24bpp", mode: tim.Mode24bpp, want: "24bpp"},
		{name: "Mixed", mode: tim.ModeMixed, want: "mixed"},
		{name: "Unknown", mode: tim.PixelMode(7), want: "PixelMode(7)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			mode := tc.mode

			// Act
			got := mode.String()

			// Assert
			if want := tc.want; got != want {
				t.Errorf("String() = %q, want %q", got, want)
			}
		})
	}
}

func TestFileEncodedSize(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		data []byte
		want int
	}{
		{
			name: "Direct16",
			data: timtest.New(timtest.Mode(tim.Mode16bpp), timtest.Pixels(2, 1, timtest.U16s(0, 0))),
			want: 8 + 12 + 4,
		}, {
			name: "Indexed8WithClut",
			data: timtest.New(timtest.Mode(tim.Mode8bpp), timtest.Palette(2, 1, timtest.U16s(0, 0)), timtest.Pixels(1, 1, []byte{0, 0})),
			want: 8 + (12 + 4) + (12 + 2),
		}, {
			name: "Indexed4MultiPalette",
			data: timtest.New(timtest.Mode(tim.Mode4bpp), timtest.Palette(2, 2, timtest.U16s(0, 0, 0, 0)), timtest.Pixels(1, 1, []byte{0, 0})),
			want: 8 + (12 + 8) + (12 + 2),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			file := mustParse(t, tc.data)

			// Act
			got := file.EncodedSize()

			// Assert
			if want := tc.want; got != want {
				t.Errorf("EncodedSize() = %d, want %d", got, want)
			}
		})
	}
}

// tim16 returns a minimal valid 16-bit, single-pixel TIM; it occupies 22 bytes.
func tim16() []byte {
	return timtest.New(timtest.Mode(tim.Mode16bpp), timtest.Pixels(1, 1, timtest.U16s(0x001F)))
}

func TestDecodeAll(t *testing.T) {
	t.Parallel()

	const size = 8 + 12 + 2 // EncodedSize of tim16

	one := &tim.File{Mode: tim.Mode16bpp, Width: 1, Height: 1}

	testCases := []struct {
		name    string
		data    []byte
		want    []tim.Located
		wantErr error
	}{
		{
			name:    "SingleAtZero",
			data:    tim16(),
			want:    []tim.Located{{Offset: 0, File: one}},
			wantErr: nil,
		}, {
			name:    "LeadingZeroPadding",
			data:    append(make([]byte, 5), tim16()...),
			want:    []tim.Located{{Offset: 5, File: one}},
			wantErr: nil,
		}, {
			name:    "TwoContiguous",
			data:    append(tim16(), tim16()...),
			want:    []tim.Located{{Offset: 0, File: one}, {Offset: size, File: one}},
			wantErr: nil,
		}, {
			name:    "PaddingTwoAndTrailing",
			data:    bytes.Join([][]byte{make([]byte, 4), tim16(), tim16(), make([]byte, 3)}, nil),
			want:    []tim.Located{{Offset: 4, File: one}, {Offset: 4 + size, File: one}},
			wantErr: nil,
		}, {
			name:    "MalformedAfterValid",
			data:    append(tim16(), timtest.New(timtest.PixelByteCount(0xffffffff))...),
			want:    []tim.Located{{Offset: 0, File: one}},
			wantErr: tim.ErrCorrupt,
		}, {
			name:    "NonTIMBytes",
			data:    []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			want:    nil,
			wantErr: nil,
		}, {
			name:    "AllZero",
			data:    make([]byte, 16),
			want:    nil,
			wantErr: nil,
		}, {
			name:    "Empty",
			data:    nil,
			want:    nil,
			wantErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			data := tc.data

			// Act
			located, err := tim.DecodeAll(data)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("DecodeAll() error = %v, want %v", got, want)
			}
			if got, want := located, tc.want; !cmp.Equal(got, want, cmpopts.IgnoreUnexported(tim.File{})) {
				t.Errorf("DecodeAll() mismatch (-want +got):\n%s", cmp.Diff(want, got, cmpopts.IgnoreUnexported(tim.File{})))
			}
		})
	}
}
