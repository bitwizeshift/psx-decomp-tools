package iso

import "io"

// NoBlock is the [Extent.Block] value for a range that is not block-addressed,
// such as an unreferenced gap that does not begin on a logical block boundary.
const NoBlock = int64(-1)

// ToEOF is the [Extent.Length] value for a range whose length is not known in
// advance because it runs to the end of the image.
const ToEOF = int64(-1)

// Extent locates a contiguous byte range within an image.
type Extent struct {
	// Offset is the absolute byte offset of the range from the start of the image.
	Offset int64

	// Length is the size of the range in bytes, or [ToEOF] when the range runs to
	// the end of the image.
	Length int64

	// Block is the logical block address at which the range begins, or [NoBlock]
	// when the range is not aligned to a logical block.
	Block int64
}

// Region is a raw byte range together with a reader over its bytes.
type Region struct {
	Extent

	// Data streams the region's bytes in order. It is read incrementally and is
	// never backed by a fully buffered copy of the region.
	Data io.Reader
}

// File is a file declared by a directory record, together with a reader over its
// contents.
type File struct {
	// Path is the file's absolute path from the root directory, using cleaned
	// names and "/" separators, such as "/SYSTEM.CNF".
	Path string

	// Name is the file's leaf name with the version suffix (";N") removed.
	Name string

	// Record is the directory record that declared the file.
	Record *DirectoryRecord

	// Extent locates the file's data within the image.
	Extent Extent

	// Data streams the file's contents in order, read incrementally.
	Data io.Reader
}
