package mdec

import "math"

// zigzag maps a natural 8x8 raster index to its position in the zig-zag scan
// order the bitstream uses.
var zigzag = [64]int{
	0, 1, 5, 6, 14, 15, 27, 28,
	2, 4, 7, 13, 16, 26, 29, 42,
	3, 8, 12, 17, 25, 30, 41, 43,
	9, 11, 18, 24, 31, 40, 44, 53,
	10, 19, 23, 32, 39, 45, 52, 54,
	20, 22, 33, 38, 46, 51, 55, 60,
	21, 34, 37, 47, 50, 56, 59, 61,
	35, 36, 48, 49, 57, 58, 62, 63,
}

// zagzig is the inverse of [zigzag]: it maps a scan position to its natural
// raster index, so a coefficient decoded at scan position k is stored at
// zagzig[k].
var zagzig = func() [64]int {
	var inv [64]int
	for natural, scan := range zigzag {
		inv[scan] = natural
	}
	return inv
}()

// quantTable is the standard MDEC quantization table, indexed by scan position.
// STR movies and BS pictures use it for both the luminance and chrominance
// blocks.
var quantTable = [64]int{
	0x02, 0x10, 0x10, 0x13, 0x10, 0x13, 0x16, 0x16,
	0x16, 0x16, 0x16, 0x16, 0x1a, 0x18, 0x1a, 0x1b,
	0x1b, 0x1b, 0x1a, 0x1a, 0x1a, 0x1a, 0x1b, 0x1b,
	0x1b, 0x1d, 0x1d, 0x1d, 0x22, 0x22, 0x22, 0x1d,
	0x1d, 0x1d, 0x1b, 0x1b, 0x1d, 0x1d, 0x20, 0x20,
	0x22, 0x22, 0x25, 0x26, 0x25, 0x23, 0x23, 0x22,
	0x23, 0x26, 0x26, 0x28, 0x28, 0x28, 0x30, 0x30,
	0x2e, 0x2e, 0x38, 0x38, 0x3a, 0x45, 0x45, 0x53,
}

// idctCos holds idctCos[x][u] = C(u)*cos((2x+1)*u*pi/16), with C(0)=1/sqrt(2) and
// C(u)=1 otherwise, the per-axis weights of the inverse DCT.
var idctCos = func() [8][8]float64 {
	var table [8][8]float64
	for x := range 8 {
		for u := range 8 {
			c := 1.0
			if u == 0 {
				c = 1 / math.Sqrt2
			}
			table[x][u] = c * math.Cos(float64((2*x+1)*u)*math.Pi/16)
		}
	}
	return table
}()

// idct converts an 8x8 block of dequantized DCT coefficients in raster order into
// spatial samples, also in raster order.
func idct(coeff *[64]int) [64]int {
	var out [64]int
	for y := range 8 {
		for x := range 8 {
			sum := 0.0
			for v := range 8 {
				for u := range 8 {
					sum += idctCos[x][u] * idctCos[y][v] * float64(coeff[v*8+u])
				}
			}
			out[y*8+x] = int(math.Round(sum / 4))
		}
	}
	return out
}
