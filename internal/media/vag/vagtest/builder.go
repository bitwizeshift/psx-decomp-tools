package vagtest

import "encoding/binary"

// Frame describes one 16-byte SPU ADPCM frame: its predictor filter and range
// shift, the loop-flag byte, and the 28 four-bit sample nibbles it carries.
type Frame struct {
	Filter  byte
	Shift   byte
	Flags   byte
	Nibbles [28]byte
}

// Bytes encodes f as its 16 raw bytes, packing two nibbles per sample byte with
// the even sample in the low nibble.
func (f Frame) Bytes() []byte {
	frame := make([]byte, 16)
	frame[0] = f.Filter<<4 | f.Shift&0x0f
	frame[1] = f.Flags
	for i := range 28 {
		nibble := f.Nibbles[i] & 0x0f
		if i&1 == 1 {
			frame[2+i/2] |= nibble << 4
		} else {
			frame[2+i/2] |= nibble
		}
	}
	return frame
}

// Body concatenates frames into a headerless SPU ADPCM stream.
func Body(frames ...Frame) []byte {
	body := make([]byte, 0, len(frames)*16)
	for _, frame := range frames {
		body = append(body, frame.Bytes()...)
	}
	return body
}

// File wraps body in a standalone "VAGp" header recording rate and name.
func File(rate int, name string, body []byte) []byte {
	out := make([]byte, 48)
	copy(out[0:4], "VAGp")
	binary.BigEndian.PutUint32(out[4:8], 0x20)
	binary.BigEndian.PutUint32(out[12:16], uint32(len(body)))
	binary.BigEndian.PutUint32(out[16:20], uint32(rate))
	copy(out[32:48], name)
	return append(out, body...)
}
