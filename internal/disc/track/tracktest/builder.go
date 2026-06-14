package tracktest

import (
	"encoding/binary"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/cue"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/edc"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track"
)

const rawSectorSize = 2352

// syncPattern is the 12-byte sync field at the start of every full sector.
var syncPattern = []byte{
	0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x00,
}

// toBCD encodes v as a two-digit binary-coded decimal byte.
func toBCD(v int) byte {
	return byte((v/10)<<4 | v%10)
}

// writeHeader writes the sync pattern and the BCD MM:SS:FF header with mode as
// the sector's mode byte.
func writeHeader(sector []byte, msf cue.MSF, mode byte) {
	copy(sector[0:12], syncPattern)
	sector[12] = toBCD(msf.Minute)
	sector[13] = toBCD(msf.Second)
	sector[14] = toBCD(msf.Frame)
	sector[15] = mode
}

// writeSubheader writes the two redundant copies of the XA subheader, using
// submode for the submode byte.
func writeSubheader(sector []byte, sub track.Subheader, submode byte) {
	fields := []byte{sub.File, sub.Channel, submode, sub.Coding}
	copy(sector[16:20], fields)
	copy(sector[20:24], fields)
}

// BuildMode1Sector assembles a valid raw 2352-byte CD-ROM Mode 1 sector at the
// given address carrying data as its 2048-byte user-data region, with a correct
// EDC.
func BuildMode1Sector(msf cue.MSF, data []byte) []byte {
	sector := make([]byte, rawSectorSize)
	writeHeader(sector, msf, 0x01)
	copy(sector[16:2064], data)
	binary.LittleEndian.PutUint32(sector[2064:], edc.Compute(sector[0:2064]))
	return sector
}

// BuildMode2Form1Sector assembles a valid raw 2352-byte CD-ROM XA Mode 2 Form 1
// sector at the given address with sub as its subheader and data as its
// 2048-byte user-data region, with a correct EDC. The subheader's Form 2 bit is
// cleared to mark the sector as Form 1.
func BuildMode2Form1Sector(sub track.Subheader, msf cue.MSF, data []byte) []byte {
	sector := make([]byte, rawSectorSize)
	writeHeader(sector, msf, 0x02)
	writeSubheader(sector, sub, sub.SubMode&^track.SubModeForm2)
	copy(sector[24:2072], data)
	binary.LittleEndian.PutUint32(sector[2072:], edc.Compute(sector[16:2072]))
	return sector
}

// BuildMode2Form2Sector assembles a valid raw 2352-byte CD-ROM XA Mode 2 Form 2
// sector at the given address with sub as its subheader and data as its
// 2324-byte user-data region, with a correct EDC. The subheader's Form 2 bit is
// set to mark the sector as Form 2.
func BuildMode2Form2Sector(sub track.Subheader, msf cue.MSF, data []byte) []byte {
	sector := make([]byte, rawSectorSize)
	writeHeader(sector, msf, 0x02)
	writeSubheader(sector, sub, sub.SubMode|track.SubModeForm2)
	copy(sector[24:2348], data)
	binary.LittleEndian.PutUint32(sector[2348:], edc.Compute(sector[16:2348]))
	return sector
}
