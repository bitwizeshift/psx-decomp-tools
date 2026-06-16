package iso

import "time"

// FileFlag is a bit in a directory record's file-flags field.
type FileFlag uint8

// File flags recorded in a directory record.
const (
	// FileFlagHidden marks an entry that should be hidden from the user.
	FileFlagHidden FileFlag = 1 << 0

	// FileFlagDirectory marks an entry whose extent is itself a directory.
	FileFlagDirectory FileFlag = 1 << 1

	// FileFlagAssociated marks an associated file.
	FileFlagAssociated FileFlag = 1 << 2

	// FileFlagRecord marks an entry whose extended attribute record records a
	// record format.
	FileFlagRecord FileFlag = 1 << 3

	// FileFlagProtection marks an entry whose extended attribute record records
	// owner or permission information.
	FileFlagProtection FileFlag = 1 << 4

	// FileFlagMultiExtent marks a record that is not the final extent of a
	// multi-extent file.
	FileFlagMultiExtent FileFlag = 1 << 7
)

// dirRecordFixedLen is the size of the fixed portion of a directory record that
// precedes its variable-length identifier.
const dirRecordFixedLen = 33

// DirectoryRecord is a single entry within a directory extent, describing a file
// or subdirectory.
type DirectoryRecord struct {
	// Extent locates the record's own bytes within the image.
	Extent Extent

	// Identifier is the raw file identifier; the current directory is "\x00" and
	// the parent directory is "\x01".
	Identifier string

	// Name is the human-facing leaf name: "." or ".." for the special entries,
	// otherwise the identifier with its version suffix removed.
	Name string

	// ExtendedAttrLength is the length, in logical blocks, of the entry's extended
	// attribute record.
	ExtendedAttrLength uint8

	// DataBlock is the logical block address at which the entry's data begins.
	DataBlock uint32

	// DataLength is the size, in bytes, of the entry's data.
	DataLength uint32

	// Recorded is the entry's recording timestamp, or the zero [time.Time] when
	// unset.
	Recorded time.Time

	// Flags holds the entry's file flags.
	Flags FileFlag

	// VolumeSequence is the number of the volume on which the entry's data
	// resides.
	VolumeSequence uint16
}

// IsDir reports whether the record describes a subdirectory.
func (r *DirectoryRecord) IsDir() bool {
	return r.Flags&FileFlagDirectory != 0
}

// isSpecial reports whether the record is one of the "." or ".." entries.
func (r *DirectoryRecord) isSpecial() bool {
	return r.Identifier == "\x00" || r.Identifier == "\x01"
}

// parseDirectoryRecord parses a single directory record from the start of b,
// reporting its position with offset. It returns the record and the number of
// bytes it occupied. A leading length byte of zero yields a zero record and a
// zero length, marking the end of records within the current logical block. It
// returns [ErrCorruptImage] when the record's declared length cannot hold its
// identifier or overruns b.
func parseDirectoryRecord(b []byte, offset int64) (DirectoryRecord, int, error) {
	length := int(b[0])
	if length == 0 {
		return DirectoryRecord{}, 0, nil
	}
	if length < dirRecordFixedLen+1 || length > len(b) {
		return DirectoryRecord{}, 0, ErrCorruptImage
	}
	idLen := int(b[32])
	if dirRecordFixedLen+idLen > length {
		return DirectoryRecord{}, 0, ErrCorruptImage
	}
	identifier := string(b[33 : 33+idLen])
	return DirectoryRecord{
		Extent:             Extent{Offset: offset, Length: int64(length), Block: NoBlock},
		Identifier:         identifier,
		Name:               identifierName(identifier),
		ExtendedAttrLength: b[1],
		DataBlock:          bothUint32(b[2:10]),
		DataLength:         bothUint32(b[10:18]),
		Recorded:           decodeRecordingTime(b[18:25]),
		Flags:              FileFlag(b[25]),
		VolumeSequence:     bothUint16(b[28:32]),
	}, length, nil
}

// identifierName returns the human-facing leaf name for a raw file identifier,
// mapping the special "\x00" and "\x01" identifiers to "." and ".." and cleaning
// any other identifier.
func identifierName(identifier string) string {
	switch identifier {
	case "\x00":
		return "."
	case "\x01":
		return ".."
	}
	return cleanName(identifier)
}
