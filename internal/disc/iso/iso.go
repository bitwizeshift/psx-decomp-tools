package iso

import (
	"bytes"
	"errors"
	"io"
	"sort"
)

// ISO parses an ISO 9660 image and walks every byte of it past a [Visitor].
// Construct it from a flat reader with [FromReaderAt] or from a decoded sector
// source with [FromSectorSource]; the latter reassembles files from their
// raw, form-aware user data and reports the trailing bytes of sectors that
// belong to no file as [Unexpected].
type ISO struct {
	sectors SectorSource
	cooked  io.ReaderAt
	tailed  bool
}

// FromReaderAt returns an [ISO] that parses the cooked ISO image read through r,
// addressed as fixed 2048-byte logical sectors. The image's size is discovered
// while walking; r need only report [io.EOF] from [io.ReaderAt.ReadAt] once its
// end is reached. Its sectors carry no tails, so it reports no [Unexpected] bytes
// and its files are the plain 2048-byte-per-sector data.
func FromReaderAt(r io.ReaderAt) *ISO {
	source := flatSource{r: r}
	return &ISO{sectors: source, cooked: SectorReader{Source: source}}
}

// FromSectorSource returns an [ISO] that parses the image addressed over s,
// reassembles each file from the raw user data of its sectors, and reports
// the trailing bytes of sectors outside any file as [Unexpected].
func FromSectorSource(s SectorSource) *ISO {
	return &ISO{sectors: s, cooked: SectorReader{Source: s}, tailed: true}
}

// Visit walks the whole image in ascending byte order, calling v for the reserved
// system area, each volume descriptor, each path table and directory record, each
// file, and every unreferenced range between and after them. For an [ISO] built
// with [FromSectorSource] it then reports the unexpected tail of every sector that
// belongs to no file. It stops at and returns the first error from v or from
// parsing, or one of [ErrBadMagic], [ErrNoPrimaryDescriptor], [ErrCorruptImage],
// or [ErrTruncated] when the image is malformed.
func (i *ISO) Visit(v Visitor) error {
	segments, files, err := i.layout()
	if err != nil {
		return err
	}
	cursor := int64(0)
	for s := range segments {
		extent := segments[s].extent
		if extent.Offset > cursor {
			if err := i.visitGap(v, cursor, extent.Offset); err != nil {
				return err
			}
		}
		if err := segments[s].visit(i, v); err != nil {
			return err
		}
		cursor = extent.Offset + extent.Length
	}
	if err := i.visitTrailing(v, cursor); err != nil {
		return err
	}
	if i.tailed {
		return i.visitUnexpected(v, files)
	}
	return nil
}

// visitUnexpected reports the tail of every sector that belongs to no file, in
// sector order, stopping at the source's end. A file sector's stream tail belongs
// to the file and is not reported here.
func (i *ISO) visitUnexpected(v Visitor, files []fileRange) error {
	for n := 0; ; n++ {
		sector, err := i.sectors.ReadSector(n)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if len(sector.Tail) == 0 {
			continue
		}
		if sector.TailIsStream && inFile(n, files) {
			continue
		}
		unexpected := &Unexpected{
			Block:  int64(n),
			Length: int64(len(sector.Tail)),
			Data:   bytes.NewReader(sector.Tail),
		}
		if err := v.VisitUnexpected(unexpected); err != nil {
			return err
		}
	}
}

// visitGap reports the bytes in [from, to) as an unreferenced region.
func (i *ISO) visitGap(v Visitor, from, to int64) error {
	return v.VisitUnreferenced(i.region(Extent{Offset: from, Length: to - from, Block: NoBlock}))
}

// visitTrailing reports any bytes from cursor to the end of the cooked image as a
// final unreferenced region, doing nothing when cursor is already at the end.
func (i *ISO) visitTrailing(v Visitor, cursor int64) error {
	var probe [1]byte
	n, err := i.cooked.ReadAt(probe[:], cursor)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if n == 0 {
		return nil
	}
	region := &Region{
		Extent: Extent{Offset: cursor, Length: ToEOF, Block: NoBlock},
		Data:   &offsetReader{r: i.cooked, off: cursor},
	}
	return v.VisitUnreferenced(region)
}

// region returns a [Region] over extent backed by an incremental reader over the
// cooked logical blocks.
func (i *ISO) region(extent Extent) *Region {
	return &Region{
		Extent: extent,
		Data:   io.NewSectionReader(i.cooked, extent.Offset, extent.Length),
	}
}

// fileRange is the half-open sector range [start, end) a file occupies.
type fileRange struct {
	start int
	end   int
}

// inFile reports whether sector n falls within one of the sorted, non-overlapping
// file ranges.
func inFile(n int, files []fileRange) bool {
	idx := sort.Search(len(files), func(i int) bool {
		return files[i].end > n
	})
	return idx < len(files) && n >= files[idx].start
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
