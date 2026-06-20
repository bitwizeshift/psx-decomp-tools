package subheader

import "io"

// Size is the number of bytes one subheader occupies in a sidecar.
const Size = 4

// User-data sizes of the two Mode 2 sector forms.
const (
	// Form1Size is the user-data length of a Form 1 sector, such as MDEC video.
	Form1Size = 2048

	// Form2Size is the user-data length of a Form 2 sector, such as XA audio.
	Form2Size = 2324
)

// Submode flag bits used to classify a sector.
const (
	// audioBit marks a sector that carries XA ADPCM audio.
	audioBit = 0x04

	// formBit marks a Form 2 sector; it is clear for Form 1.
	formBit = 0x20

	// realTimeBit marks a real-time sector, as used by interleaved streams.
	realTimeBit = 0x40
)

// Subheader is the 4-byte CD-XA subheader of a single sector.
type Subheader struct {
	// File is the interleaved file number.
	File byte

	// Channel is the interleaved channel number, distinguishing audio streams.
	Channel byte

	// SubMode is the submode flag bits.
	SubMode byte

	// Coding is the coding byte, describing audio parameters for an audio sector.
	Coding byte
}

// Form2 reports whether the sector is a Form 2 sector, whose user data is larger.
func (s Subheader) Form2() bool {
	return s.SubMode&formBit != 0
}

// Audio reports whether the sector carries XA ADPCM audio.
func (s Subheader) Audio() bool {
	return s.SubMode&audioBit != 0
}

// RealTime reports whether the sector is flagged real-time, as the interleaved
// video and audio sectors of a stream are.
func (s Subheader) RealTime() bool {
	return s.SubMode&realTimeBit != 0
}

// UserDataSize returns the number of user-data bytes the sector occupies in an
// extracted file.
func (s Subheader) UserDataSize() int {
	if s.Form2() {
		return Form2Size
	}
	return Form1Size
}

// Parse decodes the subheaders packed in a sidecar. Any trailing bytes shorter
// than one subheader are ignored.
func Parse(data []byte) []Subheader {
	subheaders := make([]Subheader, len(data)/Size)
	for i := range subheaders {
		b := data[i*Size:]
		subheaders[i] = Subheader{File: b[0], Channel: b[1], SubMode: b[2], Coding: b[3]}
	}
	return subheaders
}

// Read reads a whole sidecar from r and decodes its subheaders.
func Read(r io.Reader) ([]Subheader, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return Parse(data), nil
}
