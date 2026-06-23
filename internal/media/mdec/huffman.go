package mdec

// eobCode is the MDEC value that ends a block; its run field of 63 advances the
// scan position past the last coefficient.
const eobCode = 0xFE00

// acKind distinguishes the three outcomes of decoding an AC Huffman code.
type acKind int

const (
	acValue  acKind = iota // a run/level MDEC word
	acEOB                  // end of block
	acEscape               // a 16-bit raw MDEC word follows in the stream
)

// acEntry is a decoded AC Huffman code.
type acEntry struct {
	kind acKind
	word uint16
}

// acRow is one line of the BS v1/v2/v3 AC Huffman table: a fixed bit prefix, a
// number of index bits selecting one of values, and a trailing sign bit that
// negates the level.
type acRow struct {
	prefix string
	index  int
	values []uint16
}

// acRows is the BS v1/v2/v3 AC value table. Each MDEC word packs a zero run in
// bits 15-10 and a signed level in bits 9-0.
var acRows = []acRow{
	{prefix: "11", index: 0, values: []uint16{0x0001}},
	{prefix: "011", index: 0, values: []uint16{0x0401}},
	{prefix: "010", index: 1, values: []uint16{0x0002, 0x0801}},
	{prefix: "0011", index: 1, values: []uint16{0x1001, 0x0C01}},
	{prefix: "00101", index: 0, values: []uint16{0x0003}},
	{prefix: "00100", index: 3, values: []uint16{0x3401, 0x0006, 0x3001, 0x2C01, 0x0C02, 0x0403, 0x0005, 0x2801}},
	{prefix: "0001", index: 2, values: []uint16{0x1C01, 0x1801, 0x0402, 0x1401}},
	{prefix: "00001", index: 2, values: []uint16{0x0802, 0x2401, 0x0004, 0x2001}},
	{prefix: "0000001", index: 3, values: []uint16{0x4001, 0x1402, 0x0007, 0x0803, 0x0404, 0x3C01, 0x3801, 0x1002}},
	{prefix: "00000001", index: 4, values: []uint16{
		0x000B, 0x2002, 0x1003, 0x000A, 0x0804, 0x1C02, 0x5401, 0x5001,
		0x0009, 0x4C01, 0x4801, 0x0405, 0x0C03, 0x0008, 0x1802, 0x4401,
	}},
	{prefix: "000000001", index: 4, values: []uint16{
		0x2802, 0x2402, 0x1403, 0x0C04, 0x0805, 0x0407, 0x0406, 0x000F,
		0x000E, 0x000D, 0x000C, 0x6801, 0x6401, 0x6001, 0x5C01, 0x5801,
	}},
	{prefix: "0000000001", index: 4, values: []uint16{
		0x001F, 0x001E, 0x001D, 0x001C, 0x001B, 0x001A, 0x0019, 0x0018,
		0x0017, 0x0016, 0x0015, 0x0014, 0x0013, 0x0012, 0x0011, 0x0010,
	}},
	{prefix: "00000000001", index: 4, values: []uint16{
		0x0028, 0x0027, 0x0026, 0x0025, 0x0024, 0x0023, 0x0022, 0x0021,
		0x0020, 0x040E, 0x040D, 0x040C, 0x040B, 0x040A, 0x0409, 0x0408,
	}},
	{prefix: "000000000001", index: 4, values: []uint16{
		0x0412, 0x0411, 0x0410, 0x040F, 0x1803, 0x4002, 0x3C02, 0x3802,
		0x3402, 0x3002, 0x2C02, 0x7C01, 0x7801, 0x7401, 0x7001, 0x6C01,
	}},
}

// acTable maps a canonical code key to its decoded AC entry. The key is the code
// bits prefixed with a leading one, so codes of different lengths never collide.
var acTable = buildACTable()

// buildACTable expands the AC rows, the EOB code, and the escape code into the
// canonical lookup used by [decodeAC].
func buildACTable() map[int]acEntry {
	table := map[int]acEntry{}
	put := func(code string, entry acEntry) {
		key := codeKey(code)
		if _, ok := table[key]; ok {
			panic("mdec: duplicate AC Huffman code " + code)
		}
		table[key] = entry
	}
	put("10", acEntry{kind: acEOB})
	put("000001", acEntry{kind: acEscape})
	for _, row := range acRows {
		for i, value := range row.values {
			bits := row.prefix + indexBits(i, row.index)
			put(bits+"0", acEntry{kind: acValue, word: value})
			put(bits+"1", acEntry{kind: acValue, word: negateLevel(value)})
		}
	}
	return table
}

// decodeAC reads one AC Huffman code and returns its MDEC word: [eobCode] for end
// of block, the 16-bit escape word, or a run/level value. It sets the reader's
// error on a truncated or unrecognized code.
func decodeAC(r *bitReader) uint16 {
	key := 1
	for range maxACBits {
		key = key<<1 | r.bit()
		if r.err {
			return eobCode
		}
		entry, ok := acTable[key]
		if !ok {
			continue
		}
		switch entry.kind {
		case acEOB:
			return eobCode
		case acEscape:
			return uint16(r.read(16))
		default:
			return entry.word
		}
	}
	r.err = true
	return eobCode
}

// maxACBits is the longest AC code length, the bound that stops decoding on
// invalid input.
const maxACBits = 17

// codeKey turns a string of '0'/'1' bits into its canonical key: the bits with a
// leading one prepended.
func codeKey(code string) int {
	key := 1
	for _, c := range code {
		key = key<<1 | int(c-'0')
	}
	return key
}

// indexBits formats v as an n-bit string, most-significant bit first.
func indexBits(v, n int) string {
	bits := make([]byte, n)
	for i := range n {
		bits[n-1-i] = byte('0' + (v>>i)&1)
	}
	return string(bits)
}

// negateLevel returns word with its signed 10-bit level negated, keeping its run
// field.
func negateLevel(word uint16) uint16 {
	level := -int(word&0x3FF) & 0x3FF
	return word&0xFC00 | uint16(level)
}
