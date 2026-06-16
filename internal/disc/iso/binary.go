package iso

import (
	"encoding/binary"
	"strings"
)

// bothUint32 decodes an ISO 9660 both-byte-order 32-bit integer from b, which
// stores the value little-endian in b[0:4] then big-endian in b[4:8]. The
// little-endian encoding is trusted, so a disc whose redundant big-endian copy
// disagrees is still read.
func bothUint32(b []byte) uint32 {
	return binary.LittleEndian.Uint32(b[0:4])
}

// bothUint16 decodes an ISO 9660 both-byte-order 16-bit integer from b, which
// stores the value little-endian in b[0:2] then big-endian in b[2:4]. The
// little-endian encoding is trusted, so a disc whose redundant big-endian copy
// disagrees is still read.
func bothUint16(b []byte) uint16 {
	return binary.LittleEndian.Uint16(b[0:2])
}

// trimField returns b as a string with the trailing spaces ISO 9660 uses to pad
// fixed-width fields removed.
func trimField(b []byte) string {
	return strings.TrimRight(string(b), " ")
}

// cleanName returns the human-facing leaf name of a raw file identifier, with the
// ISO 9660 version suffix (";N") and any trailing "." removed.
func cleanName(id string) string {
	if i := strings.IndexByte(id, ';'); i >= 0 {
		id = id[:i]
	}
	return strings.TrimSuffix(id, ".")
}
