package mdec

// bitReader reads bits from a BS bitstream. The stream is a sequence of
// little-endian 16-bit halfwords whose bits are consumed most-significant first,
// the order the PSX hardware expects.
type bitReader struct {
	data []byte
	pos  int
	cur  uint16
	bits int
	err  bool
}

// newBitReader returns a [bitReader] over data.
func newBitReader(data []byte) *bitReader {
	return &bitReader{data: data}
}

// bit returns the next bit, or sets err and returns zero once the data is
// exhausted.
func (r *bitReader) bit() int {
	if r.bits == 0 {
		if r.pos+1 >= len(r.data) {
			r.err = true
			return 0
		}
		r.cur = uint16(r.data[r.pos]) | uint16(r.data[r.pos+1])<<8
		r.pos += 2
		r.bits = 16
	}
	r.bits--
	return int((r.cur >> uint(r.bits)) & 1)
}

// read returns the next n bits as an unsigned integer, most-significant bit
// first.
func (r *bitReader) read(n int) int {
	v := 0
	for range n {
		v = v<<1 | r.bit()
	}
	return v
}
