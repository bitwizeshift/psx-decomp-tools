package iso

// Visitor receives a callback for every region of an ISO 9660 image as it is
// walked in ascending byte order. Every byte of the image is delivered to
// exactly one callback, so a visitor that records the reported extents observes
// the whole image with no gaps or overlaps. Any callback may return a non-nil
// error to stop the walk, which [ISO.Visit] then returns.
type Visitor interface {
	// VisitSystemArea reports the reserved logical sectors that precede the volume
	// descriptor set.
	VisitSystemArea(r *Region) error

	// VisitVolumeDescriptor reports one descriptor of the volume descriptor set.
	VisitVolumeDescriptor(d *VolumeDescriptor) error

	// VisitPathTableRecord reports one record of a path table.
	VisitPathTableRecord(rec *PathTableRecord) error

	// VisitDirectoryRecord reports one directory record, in directory-tree order.
	VisitDirectoryRecord(rec *DirectoryRecord) error

	// VisitFile reports one file's metadata together with a [FileStream] over its
	// raw, form-aware contents.
	VisitFile(file *File, stream *FileStream) error

	// VisitUnreferenced reports a byte range claimed by no known structure, such
	// as an inter-extent gap, the slack after a file within its final block, or
	// data trailing the volume. Its length is [ToEOF] when the range runs to the
	// end of the image.
	VisitUnreferenced(r *Region) error

	// VisitUnexpected reports the trailing bytes of a sector, beyond its ISO 9660
	// logical block, that belong to no file stream. It is called only for an [ISO]
	// built with [FromSectorSource], once per such sector, after the byte-ordered
	// walk completes.
	VisitUnexpected(u *Unexpected) error
}

// BaseVisitor is a no-op implementation of [Visitor]. Embed it in a visitor to
// inherit no-op callbacks and override only those of interest.
type BaseVisitor struct{}

// VisitSystemArea implements [Visitor] by doing nothing.
func (BaseVisitor) VisitSystemArea(*Region) error { return nil }

// VisitVolumeDescriptor implements [Visitor] by doing nothing.
func (BaseVisitor) VisitVolumeDescriptor(*VolumeDescriptor) error { return nil }

// VisitPathTableRecord implements [Visitor] by doing nothing.
func (BaseVisitor) VisitPathTableRecord(*PathTableRecord) error { return nil }

// VisitDirectoryRecord implements [Visitor] by doing nothing.
func (BaseVisitor) VisitDirectoryRecord(*DirectoryRecord) error { return nil }

// VisitFile implements [Visitor] by doing nothing.
func (BaseVisitor) VisitFile(*File, *FileStream) error { return nil }

// VisitUnreferenced implements [Visitor] by doing nothing.
func (BaseVisitor) VisitUnreferenced(*Region) error { return nil }

// VisitUnexpected implements [Visitor] by doing nothing.
func (BaseVisitor) VisitUnexpected(*Unexpected) error { return nil }

var _ Visitor = BaseVisitor{}
