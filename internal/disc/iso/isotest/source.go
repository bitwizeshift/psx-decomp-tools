package isotest

import (
	"io"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/iso"
)

// Tail is the configured trailing data of a sector in a [SectorView]: the bytes
// past the 2048-byte logical block, whether they are stream data, and an optional
// XA subheader.
type Tail struct {
	Bytes     []byte
	IsStream  bool
	Subheader *iso.Subheader
}

// SectorView serves a 2048-byte-aligned image as decoded [iso.Sector]s for
// iso.FromSectorSource, attaching a configured [Tail] to the sectors named in
// Tails. It lets tests drive raw-file and unexpected-tail handling.
type SectorView struct {
	// Image is the 2048-byte-aligned cooked image served as logical blocks.
	Image []byte

	// Tails maps a sector index to the trailing data attached to it.
	Tails map[int]Tail
}

// ReadSector returns sector n's logical block with any configured tail attached,
// or [io.EOF] once n is past the end of the image.
func (s SectorView) ReadSector(n int) (iso.Sector, error) {
	start := n * BlockSize
	if start >= len(s.Image) {
		return iso.Sector{}, io.EOF
	}
	end := min(start+BlockSize, len(s.Image))
	sector := iso.Sector{Index: int64(n), Block: s.Image[start:end]}
	if tail, ok := s.Tails[n]; ok {
		sector.Tail = tail.Bytes
		sector.TailIsStream = tail.IsStream
		sector.Subheader = tail.Subheader
	}
	return sector, nil
}

var _ iso.SectorSource = (*SectorView)(nil)
