package xa_test

import (
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/media/xa"
	"github.com/bitwizeshift/psx-decomp-tools/internal/media/xa/xatest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestNewDecoder(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		coding  xa.Coding
		wantErr error
	}{
		{
			name:    "FourBitStereo",
			coding:  0x01,
			wantErr: nil,
		}, {
			name:    "FourBitMono",
			coding:  0x00,
			wantErr: nil,
		}, {
			name:    "EightBit",
			coding:  0x10,
			wantErr: xa.ErrUnsupportedCoding,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange / Act
			_, err := xa.NewDecoder(tc.coding)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("NewDecoder() error = %v, want %v", got, want)
			}
		})
	}
}

func TestDecodeSectorStereo(t *testing.T) {
	t.Parallel()

	// Each unit u carries the constant nibble u, decoded at shift 12 with no
	// predictor (filter 0), so every sample of unit u is simply u.
	var units [8]xatest.Unit
	for u := range units {
		units[u] = xatest.Unit{Filter: 0, Shift: 12, Nibbles: fill(byte(u))}
	}
	sector := xatest.Sector(xatest.Group(units))

	// Even units are the left channel, odd units the right, interleaved per pair.
	var want []int16
	for range 18 {
		for pair := range 4 {
			for range 28 {
				want = append(want, int16(2*pair), int16(2*pair+1))
			}
		}
	}

	// Arrange
	decoder, err := xa.NewDecoder(0x01)
	if err != nil {
		t.Fatalf("NewDecoder() error = %v", err)
	}

	// Act
	got, err := decoder.DecodeSector(sector)

	// Assert
	if err != nil {
		t.Fatalf("DecodeSector() error = %v", err)
	}
	if !cmp.Equal(got, want) {
		t.Errorf("DecodeSector() mismatch: got %d samples, want %d", len(got), len(want))
	}
}

func TestDecodeSectorMono(t *testing.T) {
	t.Parallel()

	var units [8]xatest.Unit
	for u := range units {
		units[u] = xatest.Unit{Filter: 0, Shift: 12, Nibbles: fill(byte(u))}
	}
	sector := xatest.Sector(xatest.Group(units))

	var want []int16
	for range 18 {
		for u := range 8 {
			for range 28 {
				want = append(want, int16(u))
			}
		}
	}

	// Arrange
	decoder, err := xa.NewDecoder(0x00)
	if err != nil {
		t.Fatalf("NewDecoder() error = %v", err)
	}

	// Act
	got, err := decoder.DecodeSector(sector)

	// Assert
	if err != nil {
		t.Fatalf("DecodeSector() error = %v", err)
	}
	if !cmp.Equal(got, want) {
		t.Errorf("DecodeSector() mismatch: got %d samples, want %d", len(got), len(want))
	}
}

func TestDecodeSectorPredictor(t *testing.T) {
	t.Parallel()

	// Unit 0 uses filter 1 (K0 = 60/64) at shift 0 with a single impulse, so the
	// output decays: 4096, then each next sample is round(prev * 60 / 64).
	var units [8]xatest.Unit
	units[0] = xatest.Unit{Filter: 1, Shift: 0, Nibbles: impulse()}
	sector := xatest.Sector(xatest.Group(units))
	want := []int16{4096, 3840, 3600}

	// Arrange
	decoder, err := xa.NewDecoder(0x00)
	if err != nil {
		t.Fatalf("NewDecoder() error = %v", err)
	}

	// Act
	got, err := decoder.DecodeSector(sector)

	// Assert
	if err != nil {
		t.Fatalf("DecodeSector() error = %v", err)
	}
	if got, want := got[:3], want; !cmp.Equal(got, want) {
		t.Errorf("DecodeSector() head = %v, want %v", got, want)
	}
}

func TestDecodeSectorClamps(t *testing.T) {
	t.Parallel()

	// Unit 0 uses filter 1 at shift 0 with a constant nibble, so the predictor
	// accumulates past the 16-bit range within two samples and saturates.
	testCases := []struct {
		name   string
		nibble byte
		want   int16
	}{
		{
			name:   "PositiveSaturates",
			nibble: 0x7,
			want:   32767,
		}, {
			name:   "NegativeSaturates",
			nibble: 0x8,
			want:   -32768,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			var units [8]xatest.Unit
			units[0] = xatest.Unit{Filter: 1, Shift: 0, Nibbles: fill(tc.nibble)}
			sector := xatest.Sector(xatest.Group(units))
			decoder, err := xa.NewDecoder(0x00)
			if err != nil {
				t.Fatalf("NewDecoder() error = %v", err)
			}

			// Act
			got, err := decoder.DecodeSector(sector)

			// Assert
			if err != nil {
				t.Fatalf("DecodeSector() error = %v", err)
			}
			if got, want := got[1], tc.want; got != want {
				t.Errorf("DecodeSector() sample[1] = %d, want %d", got, want)
			}
		})
	}
}

func TestDecodeSectorShort(t *testing.T) {
	t.Parallel()

	// Arrange
	decoder, err := xa.NewDecoder(0x01)
	if err != nil {
		t.Fatalf("NewDecoder() error = %v", err)
	}
	short := make([]byte, 100)

	// Act
	_, err = decoder.DecodeSector(short)

	// Assert
	if got, want := err, xa.ErrShortSector; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Errorf("DecodeSector() error = %v, want %v", got, want)
	}
}

// fill returns a unit's nibbles all set to the given value.
func fill(nibble byte) [28]byte {
	var nibbles [28]byte
	for i := range nibbles {
		nibbles[i] = nibble
	}
	return nibbles
}

// impulse returns a unit's nibbles with a single leading one.
func impulse() [28]byte {
	var nibbles [28]byte
	nibbles[0] = 1
	return nibbles
}
