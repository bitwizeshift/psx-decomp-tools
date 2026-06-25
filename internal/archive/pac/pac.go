package pac

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strconv"
)

// SectorSize is the 2048-byte boundary that every node is padded and aligned to.
const SectorSize = 2048

// headerSize is the fixed size of a node header: tag, kind, header-size, and
// total-size words.
const headerSize = 16

// Magic is the tag that begins every PAC node header.
const Magic = "PAC\x00"

// magic is the tag bytes that begin every PAC node header.
var magic = []byte(Magic)

// File is a single leaf payload discovered within a PAC archive, or the raw
// remainder of a directory whose contents could not be parsed. Its bytes are read
// through [File.Open]; the struct itself is plain metadata.
type File struct {
	// Name is the slash-separated index path of the payload within the archive
	// tree, such as "0" or "2/1/0".
	Name string

	// Kind is the payload type word from the owning leaf's header. It is not
	// meaningful when [File.Raw] is set.
	Kind uint32

	// Raw reports that the payload is the unparsed remainder of a directory, rather
	// than a PAC leaf.
	Raw bool

	// Size is the payload's length in bytes, excluding the node header and any
	// padding.
	Size uint32

	offset int64
	r      io.ReaderAt
}

// Offset returns the payload's absolute byte offset within the archive.
func (f *File) Offset() int64 {
	return f.offset
}

// Open returns a reader over the payload's [File.Size] bytes. The returned reader
// may be read independently of any other payload's reader.
func (f *File) Open() (io.ReadCloser, error) {
	return io.NopCloser(io.NewSectionReader(f.r, f.offset, int64(f.Size))), nil
}

// Reader provides access to the leaf payloads of a PAC archive in tree order.
type Reader struct {
	// File holds every leaf payload, and every raw directory remainder, in the
	// order they appear in the archive tree.
	File []*File

	r io.ReaderAt
}

// NewReader walks the size-byte PAC archive addressed through r and collects its
// leaf payloads. It returns [ErrBadMagic] when the root node is not a PAC node and
// [ErrTruncated] when a node header cannot be read.
func NewReader(r io.ReaderAt, size int64) (*Reader, error) {
	reader := &Reader{r: r}
	if err := reader.walk(0, size, ""); err != nil {
		return nil, err
	}
	return reader, nil
}

// header is the decoded 16-byte prologue of a PAC node.
type header struct {
	kind  uint32
	hdr   uint32
	total uint32
}

// leaf reports whether the node is a leaf, which carries a payload, rather than a
// directory, which carries children.
func (h header) leaf() bool {
	return h.hdr == 0
}

// dataStart returns the absolute offset of a directory's first child relative to
// the directory node at base.
func (h header) dataStart(base int64) int64 {
	return base + int64(h.hdr+1)*SectorSize
}

// span returns the sector-aligned number of bytes a leaf node occupies, including
// its header.
func (h header) span() int64 {
	return align(int64(h.total))
}

// walk parses the node in [base, end), appending the leaf payloads it contains to
// the reader under the given name prefix. A directory's nested directory child is
// treated as the final child and given the remainder of the node.
func (r *Reader) walk(base, end int64, name string) error {
	h, err := r.readHeader(base)
	if err != nil {
		return err
	}
	if h.leaf() {
		r.appendLeaf(h, base, end, name)
		return nil
	}
	dataStart := min(h.dataStart(base), end)
	r.appendData(h, base+headerSize, dataStart, name)
	return r.walkChildren(dataStart, end, name)
}

// walkChildren parses the children that tile [cursor, end) under the given name
// prefix. It stops at the first child that is not a PAC node, reporting the
// remaining bytes as a single raw [File].
func (r *Reader) walkChildren(cursor, end int64, name string) error {
	for index := 0; cursor < end; index++ {
		childName := join(name, strconv.Itoa(index))
		child, err := r.readHeader(cursor)
		if errors.Is(err, ErrBadMagic) {
			r.appendRaw(cursor, end, childName)
			return nil
		}
		if err != nil {
			return err
		}
		childEnd := end
		if child.leaf() {
			childEnd = min(cursor+child.span(), end)
		}
		if childEnd <= cursor {
			r.appendRaw(cursor, end, childName)
			return nil
		}
		if err := r.walk(cursor, childEnd, childName); err != nil {
			return err
		}
		cursor = childEnd
	}
	return nil
}

// appendLeaf records the payload of the leaf node in [base, end) under the given
// name. The payload is the whole region the node was allotted, after its header,
// rather than the node's total-size word: that word is reliable only for some
// kinds, whereas the allotted region is always correct and never drops bytes.
func (r *Reader) appendLeaf(h header, base, end int64, name string) {
	size := int64(0)
	if end > base+headerSize {
		size = end - (base + headerSize)
	}
	r.File = append(r.File, &File{
		Name:   orRoot(name),
		Kind:   h.kind,
		Size:   uint32(size),
		offset: base + headerSize,
		r:      r.r,
	})
}

// appendData records a directory's own data region, the bytes between its header
// and its first child, as a payload under the name "<name>/data". It records
// nothing when the region is empty or holds only the zero padding of a directory
// that carries no inline data.
func (r *Reader) appendData(h header, start, end int64, name string) {
	if end <= start || r.zero(start, end) {
		return
	}
	r.File = append(r.File, &File{
		Name:   join(name, "data"),
		Kind:   h.kind,
		Size:   uint32(end - start),
		offset: start,
		r:      r.r,
	})
}

// zero reports whether every byte in [start, end) is zero, treating an unreadable
// region as zero so it is skipped rather than reported.
func (r *Reader) zero(start, end int64) bool {
	section := io.NewSectionReader(r.r, start, end-start)
	buf := make([]byte, 4096)
	for {
		n, err := section.Read(buf)
		for _, b := range buf[:n] {
			if b != 0 {
				return false
			}
		}
		if err != nil {
			return true
		}
	}
}

// appendRaw records the bytes in [start, end) as a single unparsed remainder under
// the given name.
func (r *Reader) appendRaw(start, end int64, name string) {
	r.File = append(r.File, &File{
		Name:   orRoot(name),
		Raw:    true,
		Size:   uint32(end - start),
		offset: start,
		r:      r.r,
	})
}

// readHeader decodes the node header at off. It reports [ErrTruncated] when the
// header cannot be read and [ErrBadMagic] when its tag is wrong.
func (r *Reader) readHeader(off int64) (header, error) {
	var b [headerSize]byte
	if _, err := r.r.ReadAt(b[:], off); err != nil {
		return header{}, fmt.Errorf("pac: read node at %d: %w", off, ErrTruncated)
	}
	if !bytes.Equal(b[0:4], magic[:]) {
		return header{}, ErrBadMagic
	}
	return header{
		kind:  binary.LittleEndian.Uint32(b[4:8]),
		hdr:   binary.LittleEndian.Uint32(b[8:12]),
		total: binary.LittleEndian.Uint32(b[12:16]),
	}, nil
}

// align rounds n up to the next [SectorSize] boundary.
func align(n int64) int64 {
	return (n + SectorSize - 1) &^ (SectorSize - 1)
}

// join concatenates two non-empty name parts with a slash, returning the other
// when one is empty.
func join(prefix, part string) string {
	if prefix == "" {
		return part
	}
	return prefix + "/" + part
}

// orRoot returns name, or the root name "0" when name is empty.
func orRoot(name string) string {
	if name == "" {
		return "0"
	}
	return name
}
