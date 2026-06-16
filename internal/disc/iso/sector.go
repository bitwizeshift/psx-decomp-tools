package iso

import "io"

// SectorSource reads the logical user data of CD-ROM sectors by index.
// ReadSector returns sector n's user data, or [io.EOF] once n is past the final
// sector.
type SectorSource interface {
	ReadSector(n int) ([]byte, error)
}

// SectorReader adapts a [SectorSource] to an [io.ReaderAt] that addresses it as
// fixed 2048-byte logical sectors, the unit ISO 9660 is defined over.
//
// It exists because a source's per-sector user data is not a uniform size: an XA
// Mode 2 track mixes 2048-byte Form 1 sectors with 2324-byte Form 2 sectors, so
// concatenating that data yields a stream whose byte offsets no longer line up
// with sector boundaries. SectorReader restores the alignment by serving the
// first 2048 bytes of each sector at a fixed 2048-byte stride, so that byte
// offset n*2048 always addresses sector n. A [Reader] reads an ISO 9660 image
// through one of these.
type SectorReader struct {
	// Source supplies the sectors addressed by the reader.
	Source SectorSource
}

// ReadAt reads len(p) bytes starting at off, drawing each 2048-byte block from
// the corresponding sector of the source. It reports [io.EOF] once a sector past
// the end of the source is reached, or a shorter-than-requested read runs out of
// sector data.
func (r SectorReader) ReadAt(p []byte, off int64) (int, error) {
	read := 0
	for read < len(p) {
		abs := off + int64(read)
		data, err := r.Source.ReadSector(int(abs / logicalSectorSize))
		if err != nil {
			return read, err
		}
		if len(data) > logicalSectorSize {
			data = data[:logicalSectorSize]
		}
		within := int(abs % logicalSectorSize)
		if within >= len(data) {
			return read, io.EOF
		}
		read += copy(p[read:], data[within:])
	}
	return read, nil
}

var _ io.ReaderAt = SectorReader{}
