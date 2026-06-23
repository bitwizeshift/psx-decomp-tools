package iso

import "io"

// reservedSectors is the number of system-area logical sectors before the volume
// descriptor set, which begins at logical sector 16.
const reservedSectors = 16

// LooksLikeImage reports whether r begins with an ISO 9660 volume descriptor set,
// detected by the "CD001" standard identifier at the start of the first descriptor
// sector. It reads a single field and does not validate the rest of the image, so
// it is suitable for cheap content sniffing rather than full parsing.
func LooksLikeImage(r io.ReaderAt) bool {
	var b [6]byte
	if _, err := r.ReadAt(b[:], reservedSectors*logicalSectorSize); err != nil {
		return false
	}
	return string(b[1:6]) == standardIdentifier
}
