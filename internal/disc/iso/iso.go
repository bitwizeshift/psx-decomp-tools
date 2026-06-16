package iso

import (
	"errors"
	"io"
)

// Reader parses an ISO 9660 image read through an [io.ReaderAt] and walks every
// byte of it past a [Visitor].
type Reader struct {
	r io.ReaderAt
}

// New returns a [Reader] that parses the ISO 9660 image read through r. The
// image's size is discovered while walking; r need only report [io.EOF] from
// [io.ReaderAt.ReadAt] once its end is reached.
func New(r io.ReaderAt) *Reader {
	return &Reader{r: r}
}

// Visit walks the whole image in ascending byte order, calling v for the reserved
// system area, each volume descriptor, each path table and directory record, each
// file, and every unreferenced range between and after them. It stops at and
// returns the first error from v or from parsing, or one of [ErrBadMagic],
// [ErrNoPrimaryDescriptor], [ErrCorruptImage], or [ErrTruncated] when the image
// is malformed.
func (rd *Reader) Visit(v Visitor) error {
	segments, err := rd.layout()
	if err != nil {
		return err
	}
	cursor := int64(0)
	for i := range segments {
		extent := segments[i].extent
		if extent.Offset > cursor {
			if err := rd.visitGap(v, cursor, extent.Offset); err != nil {
				return err
			}
		}
		if err := segments[i].visit(rd, v); err != nil {
			return err
		}
		cursor = extent.Offset + extent.Length
	}
	return rd.visitTrailing(v, cursor)
}

// visitGap reports the bytes in [from, to) as an unreferenced region.
func (rd *Reader) visitGap(v Visitor, from, to int64) error {
	return v.VisitUnreferenced(rd.region(Extent{Offset: from, Length: to - from, Block: NoBlock}))
}

// visitTrailing reports any bytes from cursor to the end of the image as a final
// unreferenced region, doing nothing when cursor is already at the end.
func (rd *Reader) visitTrailing(v Visitor, cursor int64) error {
	var probe [1]byte
	n, err := rd.r.ReadAt(probe[:], cursor)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if n == 0 {
		return nil
	}
	region := &Region{
		Extent: Extent{Offset: cursor, Length: ToEOF, Block: NoBlock},
		Data:   &offsetReader{r: rd.r, off: cursor},
	}
	return v.VisitUnreferenced(region)
}

// region returns a [Region] over extent backed by an incremental reader.
func (rd *Reader) region(extent Extent) *Region {
	return &Region{
		Extent: extent,
		Data:   io.NewSectionReader(rd.r, extent.Offset, extent.Length),
	}
}

// offsetReader adapts an [io.ReaderAt] into a sequential [io.Reader] that streams
// from a starting offset to the end of the source.
type offsetReader struct {
	r   io.ReaderAt
	off int64
}

// Read streams the next bytes of the source into p, advancing the offset. It
// returns [io.EOF] once the end of the source is reached.
func (or *offsetReader) Read(p []byte) (int, error) {
	n, err := or.r.ReadAt(p, or.off)
	or.off += int64(n)
	return n, err
}

var _ io.Reader = (*offsetReader)(nil)
