package cd

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

// SectorSize is the 2048-byte sector that every member's starting position is
// expressed in and that every member is aligned and padded to.
const SectorSize = 2048

// headerSize is the fixed prologue of the table of contents: a member count and a
// reserved word. entrySize is the size of each table of contents entry.
const (
	headerSize = 8
	entrySize  = 8
)

// File is a single member of a ".CD" archive. Its bytes are read through
// [File.Open]; the struct itself is plain metadata.
type File struct {
	// StartSector is the member's starting [SectorSize]-byte sector within the
	// archive.
	StartSector uint32

	// Size is the member's exact length in bytes, excluding the padding that aligns
	// the following member to a sector boundary.
	Size uint32

	r io.ReaderAt
}

// Offset returns the member's absolute byte offset within the archive.
func (f *File) Offset() int64 {
	return int64(f.StartSector) * SectorSize
}

// Open returns a reader over the member's [File.Size] bytes. The returned reader
// may be read independently of any other member's reader.
func (f *File) Open() (io.ReadCloser, error) {
	section := io.NewSectionReader(f.r, f.Offset(), int64(f.Size))
	return io.NopCloser(section), nil
}

// Reader provides random access to the members of a ".CD" archive.
type Reader struct {
	// File holds every member of the archive in table of contents order.
	File []*File
}

// ReadCloser is a [Reader] backed by an open file, closed with [ReadCloser.Close].
type ReadCloser struct {
	Reader
	f *os.File
}

// Close closes the file underlying the archive.
func (rc *ReadCloser) Close() error {
	return rc.f.Close()
}

// OpenReader opens the named file as a ".CD" archive. The returned [ReadCloser]
// must be closed when it is no longer needed. It reports the same errors as
// [NewReader] in addition to any error from opening the file.
func OpenReader(name string) (*ReadCloser, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		cerr := f.Close()
		return nil, errors.Join(err, cerr)
	}
	reader, err := NewReader(f, info.Size())
	if err != nil {
		cerr := f.Close()
		return nil, errors.Join(err, cerr)
	}
	return &ReadCloser{Reader: *reader, f: f}, nil
}

// NewReader reads the table of contents of the size-byte ".CD" archive addressed
// through r. It returns [ErrTruncated] when the archive is too small to hold its
// table of contents and [ErrCorrupt] when a member's extent runs past the end of
// the archive or breaks the contiguous sector chaining of the layout.
func NewReader(r io.ReaderAt, size int64) (*Reader, error) {
	header := make([]byte, headerSize)
	if _, err := r.ReadAt(header, 0); err != nil {
		return nil, fmt.Errorf("cd: read header: %w", ErrTruncated)
	}
	count := binary.LittleEndian.Uint32(header[0:4])

	tableEnd := int64(headerSize) + int64(count)*entrySize
	if tableEnd > size {
		return nil, fmt.Errorf("cd: table of contents of %d members does not fit in %d bytes: %w", count, size, ErrTruncated)
	}

	table := make([]byte, int64(count)*entrySize)
	if _, err := r.ReadAt(table, headerSize); err != nil {
		return nil, fmt.Errorf("cd: read table of contents: %w", ErrTruncated)
	}

	files := make([]*File, count)
	nextSector := uint32(0)
	for i := range files {
		entry := table[i*entrySize:]
		file := &File{
			StartSector: binary.LittleEndian.Uint32(entry[0:4]),
			Size:        binary.LittleEndian.Uint32(entry[4:8]),
			r:           r,
		}
		if err := validate(file, i, nextSector, size); err != nil {
			return nil, err
		}
		nextSector = file.StartSector + sectorsFor(file.Size)
		files[i] = file
	}
	return &Reader{File: files}, nil
}

// validate checks that file's extent lies within an archive of the given byte
// size and, for every member after the first, begins exactly where the previous
// member's sectors ended. The first member may begin at any sector, since the
// table of contents itself occupies the sectors before it.
func validate(file *File, index int, wantSector uint32, size int64) error {
	if end := file.Offset() + int64(file.Size); end > size {
		return fmt.Errorf("cd: member %d ends at %d, past archive end %d: %w", index, end, size, ErrCorrupt)
	}
	if index > 0 && file.StartSector != wantSector {
		return fmt.Errorf("cd: member %d starts at sector %d, expected %d: %w", index, file.StartSector, wantSector, ErrCorrupt)
	}
	return nil
}

// sectorsFor returns the number of whole [SectorSize] sectors needed to hold n
// bytes.
func sectorsFor(n uint32) uint32 {
	return (n + SectorSize - 1) / SectorSize
}
