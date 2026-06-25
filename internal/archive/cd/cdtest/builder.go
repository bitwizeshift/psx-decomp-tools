package cdtest

import (
	"encoding/binary"

	"github.com/bitwizeshift/psx-decomp-tools/internal/archive/cd"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// Build assembles a well-formed ".CD" archive in memory containing the given
// members. The table of contents occupies the leading sectors and each member is
// laid out, in order, on the sector boundary where the previous member's sectors
// end.
func Build(members ...[]byte) []byte {
	tableBytes := 8 + len(members)*8
	sector := sectorsFor(tableBytes)

	type placed struct {
		start uint32
		size  uint32
		data  []byte
	}
	entries := make([]placed, len(members))
	for i, member := range members {
		entries[i] = placed{start: sector, size: uint32(len(member)), data: member}
		sector += sectorsFor(len(member))
	}

	image := make([]byte, int(sector)*cd.SectorSize)
	binary.LittleEndian.PutUint32(image[0:4], uint32(len(members)))
	for i, entry := range entries {
		off := 8 + i*8
		binary.LittleEndian.PutUint32(image[off:off+4], entry.start)
		binary.LittleEndian.PutUint32(image[off+4:off+8], entry.size)
		copy(image[int(entry.start)*cd.SectorSize:], entry.data)
	}
	return image
}

// CompareFiles returns a [cmp.Option] that compares [cd.File] values by the
// fields that identify a member, ignoring the unexported reader they carry.
func CompareFiles() cmp.Option {
	return cmpopts.IgnoreUnexported(cd.File{})
}

// sectorsFor returns the number of whole sectors needed to hold n bytes.
func sectorsFor(n int) uint32 {
	return uint32((n + cd.SectorSize - 1) / cd.SectorSize)
}
