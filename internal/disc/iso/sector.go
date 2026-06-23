package iso

import (
	"errors"
	"io"
)

// Subheader is the decoded eight-byte XA subheader carried by a Mode 2 sector,
// reported so that an XA stream can be demultiplexed later.
type Subheader struct {
	File    byte
	Channel byte
	SubMode byte
	Coding  byte
}

// Sector is one decoded CD-ROM sector presented to the walk. Its layout is
// resolved by the caller from the track mode and per-sector form, so the ISO
// reader can stay mode-agnostic.
type Sector struct {
	// Index is the sector's absolute logical block address.
	Index int64

	// Block is the sector's 2048-byte ISO 9660 logical block. It may be shorter
	// than 2048 only for a partial final sector.
	Block []byte

	// Tail is the user data beyond the logical block (276 bytes on an XA Form 2
	// sector, 304 on an audio sector, and so on), or nil when the sector has none.
	Tail []byte

	// TailIsStream reports whether Tail is the continuation of a real stream (XA
	// Form 2 or audio) rather than opaque trailing bytes. Stream tails of a file's
	// sectors belong to the file; everything else is unexpected.
	TailIsStream bool

	// Subheader is the sector's XA subheader, or nil when the layout has none.
	Subheader *Subheader
}

// SectorSource reads decoded sectors by index. ReadSector returns sector n, or
// [io.EOF] once n is past the final sector.
type SectorSource interface {
	ReadSector(n int) (Sector, error)
}

// SectorReader adapts a [SectorSource] to an [io.ReaderAt] over the cooked, fixed
// 2048-byte logical blocks the ISO 9660 structure is addressed in. It is the view
// the parser reads through; file data and sector tails are read from the
// [SectorSource] directly.
type SectorReader struct {
	// Source supplies the sectors addressed by the reader.
	Source SectorSource
}

// ReadAt reads len(p) bytes starting at off, drawing each 2048-byte block from
// the corresponding sector's logical block. It reports [io.EOF] once a sector
// past the end of the source is reached, or a block runs out before p is filled.
func (r SectorReader) ReadAt(p []byte, off int64) (int, error) {
	read := 0
	for read < len(p) {
		abs := off + int64(read)
		sector, err := r.Source.ReadSector(int(abs / logicalSectorSize))
		if err != nil {
			return read, err
		}
		within := int(abs % logicalSectorSize)
		if within >= len(sector.Block) {
			return read, io.EOF
		}
		read += copy(p[read:], sector.Block[within:])
	}
	return read, nil
}

var _ io.ReaderAt = SectorReader{}

// flatSource adapts an [io.ReaderAt] to a [SectorSource] of uniform 2048-byte
// logical blocks with no tails, used by [FromReaderAt] for flat ISO images.
type flatSource struct {
	r io.ReaderAt
}

// ReadSector returns the 2048-byte block at index n, or [io.EOF] once n is past
// the end of the reader. The final block may be short.
func (f flatSource) ReadSector(n int) (Sector, error) {
	buf := make([]byte, logicalSectorSize)
	read, err := f.r.ReadAt(buf, int64(n)*logicalSectorSize)
	if err != nil && !errors.Is(err, io.EOF) {
		return Sector{}, err
	}
	if read == 0 {
		return Sector{}, io.EOF
	}
	return Sector{Index: int64(n), Block: buf[:read]}, nil
}

var _ SectorSource = flatSource{}
