package sqvtest

import "encoding/binary"

// headerSize is the length of the SQV container header.
const headerSize = 16

// File assembles an SQV container: the ".sqv" header, the sequences in order,
// and the bank. When more than one sequence is present the header's next-
// sequence word is set to the second sequence's offset less four.
func File(sequences [][]byte, bank []byte) []byte {
	out := make([]byte, headerSize)
	copy(out[0:4], ".sqv")
	if len(sequences) > 1 {
		binary.LittleEndian.PutUint32(out[4:8], uint32(headerSize+len(sequences[0])-4))
	}
	for _, sequence := range sequences {
		out = append(out, sequence...)
	}
	return append(out, bank...)
}
