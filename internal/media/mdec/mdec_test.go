package mdec_test

import (
	"image"
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/media/mdec"
	"github.com/bitwizeshift/psx-decomp-tools/internal/media/mdec/mdectest"
	"github.com/bitwizeshift/psx-decomp-tools/internal/video/y4m/y4mtest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestDecodeHeader(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		data    []byte
		want    mdec.Header
		wantErr error
	}{
		{
			name:    "Valid",
			data:    mdectest.New(7).Build(),
			want:    mdec.Header{Version: 2, Quant: 7, Size: 0},
			wantErr: nil,
		}, {
			name:    "WrongID",
			data:    make([]byte, 8),
			want:    mdec.Header{},
			wantErr: mdec.ErrInvalidHeader,
		}, {
			name:    "Short",
			data:    make([]byte, 4),
			want:    mdec.Header{},
			wantErr: mdec.ErrInvalidHeader,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange / Act
			got, err := mdec.DecodeHeader(tc.data)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("DecodeHeader() error = %v, want %v", got, want)
			}
			if got, want := got, tc.want; !cmp.Equal(got, want) {
				t.Errorf("DecodeHeader() diff (-got +want):\n%s", cmp.Diff(got, want))
			}
		})
	}
}

func TestDecode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		data    []byte
		width   int
		height  int
		want    image.Image
		wantErr error
	}{
		{
			name: "SolidMacroblock",
			data: mdectest.New(1).
				Block(100).Block(40).
				Block(200).Block(200).Block(200).Block(200).
				EndOfFrame().Build(),
			width:   16,
			height:  16,
			want:    y4mtest.SolidYCbCr(16, 16, 178, 138, 153),
			wantErr: nil,
		}, {
			name: "ClipsToBounds",
			data: mdectest.New(1).
				Block(100).Block(40).
				Block(200).Block(200).Block(200).Block(200).
				EndOfFrame().Build(),
			width:   16,
			height:  8,
			want:    y4mtest.SolidYCbCr(16, 8, 178, 138, 153),
			wantErr: nil,
		}, {
			name: "SaturatesAndSigns",
			data: mdectest.New(1).
				Block(0x3FF).Block(0x380).
				Block(0x1FE).Block(0x1FE).Block(0x1FE).Block(0x1FE).
				EndOfFrame().Build(),
			width:   16,
			height:  16,
			want:    y4mtest.SolidYCbCr(16, 16, 255, 96, 128),
			wantErr: nil,
		}, {
			name:    "ACValueCodeRoundsAway",
			data:    acFrame("110"),
			width:   16,
			height:  16,
			want:    y4mtest.SolidYCbCr(16, 16, 128, 128, 128),
			wantErr: nil,
		}, {
			name:    "ACEscapeGradient",
			data:    acFrame("000001" + "0000000000000100"),
			width:   16,
			height:  16,
			want:    tiledLuma([8]byte{129, 129, 129, 128, 128, 127, 127, 127}),
			wantErr: nil,
		}, {
			name:    "EndOfFrameLeavesBlankImage",
			data:    mdectest.New(1).EndOfFrame().Build(),
			width:   16,
			height:  16,
			want:    y4mtest.SolidYCbCr(16, 16, 0, 0, 0),
			wantErr: nil,
		}, {
			name:    "Truncated",
			data:    mdectest.New(1).Build(),
			width:   16,
			height:  16,
			want:    nil,
			wantErr: mdec.ErrTruncated,
		}, {
			name:    "TruncatedMidBlock",
			data:    mdectest.New(1).Block(0).Block(0).DC(0).Build(),
			width:   16,
			height:  16,
			want:    nil,
			wantErr: mdec.ErrTruncated,
		}, {
			name:    "InvalidACCode",
			data:    mdectest.New(1).Block(0).Block(0).DC(0).Bits("000000000000000000000000").Build(),
			width:   16,
			height:  16,
			want:    nil,
			wantErr: mdec.ErrTruncated,
		}, {
			name:    "UnsupportedVersion",
			data:    mdectest.New(1).Version(3).Build(),
			width:   16,
			height:  16,
			want:    nil,
			wantErr: mdec.ErrUnsupportedVersion,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange / Act
			got, err := mdec.Decode(tc.data, tc.width, tc.height)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Decode() error = %v, want %v", got, want)
			}
			if got, want := got, tc.want; !cmp.Equal(got, want) {
				t.Errorf("Decode() diff (-got +want):\n%s", cmp.Diff(got, want))
			}
		})
	}
}

// acFrame builds a single 16x16 macroblock whose four luma blocks each carry one
// AC coefficient encoded by ac, with zero DC and zero chroma.
func acFrame(ac string) []byte {
	b := mdectest.New(1).Block(0).Block(0)
	for range 4 {
		b.DC(0).Bits(ac).Bits("10")
	}
	return b.EndOfFrame().Build()
}

// tiledLuma returns a 16x16 image whose luma rows are row repeated across the
// width and whose chroma is neutral, the shape a single horizontal-frequency AC
// coefficient produces in every block.
func tiledLuma(row [8]byte) *image.YCbCr {
	img := image.NewYCbCr(image.Rect(0, 0, 16, 16), image.YCbCrSubsampleRatio420)
	for i := range img.Cb {
		img.Cb[i] = 128
		img.Cr[i] = 128
	}
	for y := range 16 {
		for x := range 16 {
			img.Y[y*img.YStride+x] = row[x%8]
		}
	}
	return img
}
