package track_test

import "github.com/bitwizeshift/psx-decomp-tools/internal/disc/cue"

// testMSF is a representative sector address used across decoder and verifier
// tests.
var testMSF = cue.MSF{Minute: 1, Second: 2, Frame: 3}

// payload returns n deterministic, mostly non-zero bytes for use as sector data.
func payload(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i + 1)
	}
	return b
}

// corrupt returns a copy of data with the byte at index inverted.
func corrupt(data []byte, index int) []byte {
	out := append([]byte(nil), data...)
	out[index] ^= 0xFF
	return out
}
