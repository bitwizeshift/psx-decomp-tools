package edc_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/edc"
)

// sequential returns n bytes counting up modulo 256.
func sequential(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i)
	}
	return b
}

func TestCompute(t *testing.T) {
	t.Parallel()

	// The single-byte 0x01 answer is derived directly from the ECMA-130 EDC
	// polynomial 0xD8018001: it is table[1] after the eight reflected reductions,
	// and anchors the algorithm independently of this implementation. The
	// all-zero answers follow from a zero seed over a zero polynomial remainder.
	testCases := []struct {
		name string
		data []byte
		want uint32
	}{
		{
			name: "Empty",
			data: []byte{},
			want: 0x00000000,
		},
		{
			name: "SingleByte",
			data: []byte{0x01},
			want: 0x90910101,
		},
		{
			name: "Zeros",
			data: make([]byte, 2064),
			want: 0x00000000,
		},
		{
			name: "Fox",
			data: []byte("the quick brown fox"),
			want: 0xea889f83,
		},
		{
			name: "Sequential",
			data: sequential(2064),
			want: 0x9bd19ac4,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange

			// Act
			sum := edc.Compute(tc.data)

			// Assert
			if got, want := sum, tc.want; !cmp.Equal(got, want) {
				t.Errorf("Compute(...) = %#08x, want %#08x", got, want)
			}
		})
	}
}
