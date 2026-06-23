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

// Unexpected is the trailing bytes of a sector, beyond its ISO logical block,
// that belong to no file stream: the slack of the reserved system area, the tail
// of a Form 2 descriptor or record sector, or the opaque remainder of a bare
// layout. It is reported so that no on-disc byte is dropped.
type Unexpected struct {
	// Block is the logical sector the bytes belong to.
	Block int64

	// Length is the number of bytes.
	Length int64

	// Data streams the bytes.
	Data io.Reader
}

// File is the metadata of a file declared by a directory record. It is plain
// data; its contents are read through the [FileStream] reported alongside it.
type File struct {
	// Path is the file's absolute path from the root directory, using cleaned
	// names and "/" separators, such as "/SYSTEM.CNF".
	Path string

	// Name is the file's leaf name with the version suffix (";N") removed.
	Name string

	// Record is the directory record that declared the file.
	Record *DirectoryRecord

	// Extent locates the file's cooked extent (its 2048-byte logical blocks)
	// within the image.
	Extent Extent
}
