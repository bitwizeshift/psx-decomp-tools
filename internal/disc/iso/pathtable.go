package iso

import "encoding/binary"

// pathRecordFixedLen is the size of the fixed portion of a path table record that
// precedes its variable-length identifier.
const pathRecordFixedLen = 8

// PathTableRecord is a single entry within a path table, locating one directory
// and naming its parent.
type PathTableRecord struct {
	// Extent locates the record's own bytes within the image.
	Extent Extent

	// Identifier is the raw directory identifier; the root directory is "\x00".
	Identifier string

	// Name is the human-facing directory name: "." for the root, otherwise the
	// identifier.
	Name string

	// ExtendedAttrLength is the length, in logical blocks, of the directory's
	// extended attribute record.
	ExtendedAttrLength uint8

	// Block is the logical block address of the directory's extent.
	Block uint32

	// Parent is the one-based path table index of the directory's parent.
	Parent uint16
}

// parsePathTableRecord parses a single path table record from the start of b,
// reporting its position with offset and reading its multi-byte fields with
// order. It returns the record and the number of bytes it occupied. A leading
// identifier length of zero yields a zero record and a zero length, which a path
// table never contains. It returns [ErrTruncated] when b is too short to hold the
// declared record.
func parsePathTableRecord(b []byte, offset int64, order binary.ByteOrder) (PathTableRecord, int, error) {
	idLen := int(b[0])
	if idLen == 0 {
		return PathTableRecord{}, 0, nil
	}
	length := pathRecordFixedLen + idLen + (idLen & 1)
	if length > len(b) {
		return PathTableRecord{}, 0, ErrTruncated
	}
	identifier := string(b[8 : 8+idLen])
	return PathTableRecord{
		Extent:             Extent{Offset: offset, Length: int64(length), Block: NoBlock},
		Identifier:         identifier,
		Name:               pathName(identifier),
		ExtendedAttrLength: b[1],
		Block:              order.Uint32(b[2:6]),
		Parent:             order.Uint16(b[6:8]),
	}, length, nil
}

// pathName returns the human-facing directory name for a raw path table
// identifier, mapping the root's "\x00" identifier to ".".
func pathName(identifier string) string {
	if identifier == "\x00" {
		return "."
	}
	return identifier
}
