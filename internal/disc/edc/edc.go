package edc

// poly is the reflected generator polynomial of the CD-ROM EDC, the product of
// (x^16 + x^15 + x^2 + 1) and (x^16 + x^2 + x + 1).
const poly = 0xD8018001

var table = makeTable()

// makeTable builds the byte-wise lookup table for the reflected CD-ROM EDC.
func makeTable() [256]uint32 {
	var t [256]uint32
	for i := range t {
		crc := uint32(i)
		for range 8 {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ poly
			} else {
				crc >>= 1
			}
		}
		t[i] = crc
	}
	return t
}

// Compute returns the CD-ROM error-detection code of data: a reflected 32-bit
// CRC over the CD-ROM EDC polynomial with a zero seed and no final inversion.
func Compute(data []byte) uint32 {
	var crc uint32
	for _, b := range data {
		crc = (crc >> 8) ^ table[byte(crc)^b]
	}
	return crc
}
