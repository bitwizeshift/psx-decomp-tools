package xatest

// Unit describes one sound unit of an XA sound group: its predictor filter and
// range shift, and the 28 four-bit sample nibbles it carries.
type Unit struct {
	Filter  byte
	Shift   byte
	Nibbles [28]byte
}

// Group assembles a 128-byte XA sound group from its eight sound units, writing
// the per-unit parameter bytes and packing the sample nibbles in the interleaved
// layout the xa package expects.
func Group(units [8]Unit) []byte {
	group := make([]byte, 128)
	for u := range units {
		group[4+u] = units[u].Filter<<4 | units[u].Shift&0x0f
	}
	for s := range 28 {
		for u := range units {
			nibble := units[u].Nibbles[s] & 0x0f
			index := 16 + s*4 + u/2
			if u&1 == 1 {
				group[index] |= nibble << 4
			} else {
				group[index] |= nibble
			}
		}
	}
	return group
}

// Sector returns the 2304 ADPCM bytes of a Form 2 sector whose eighteen sound
// groups are all the given group.
func Sector(group []byte) []byte {
	sector := make([]byte, 0, 18*len(group))
	for range 18 {
		sector = append(sector, group...)
	}
	return sector
}
