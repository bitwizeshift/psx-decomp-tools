package track

import (
	"encoding/binary"
	"fmt"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/edc"
)

// SectorVerifier validates the integrity of a decoded sector. It operates on the
// sector's raw bytes, which include the regions a CD-ROM EDC protects.
type SectorVerifier interface {
	// Verify reports [ErrChecksum] if sector's stored EDC does not match the
	// value computed over its protected region, or [ErrShortSector] if the raw
	// bytes are too short to contain that region.
	Verify(sector Sector) error
}

// Mode1Verifier verifies the EDC of a CD-ROM Mode 1 sector, whose EDC protects
// the sync, header, and user-data regions.
type Mode1Verifier struct{}

// Verify implements [SectorVerifier] for Mode 1 sectors.
func (Mode1Verifier) Verify(sector Sector) error {
	return verifyEDC(sector.Raw, 0, mode1EDCOffset, false)
}

// Mode2Verifier verifies the EDC of a CD-ROM XA Mode 2 sector. Form 1 sectors
// are always protected; Form 2 sectors carry an optional EDC that, when zero, is
// treated as absent and accepted.
type Mode2Verifier struct{}

// Verify implements [SectorVerifier] for Mode 2 sectors, selecting the protected
// region from the sector's [Form].
func (Mode2Verifier) Verify(sector Sector) error {
	if sector.Subheader != nil && sector.Subheader.Form == FormTwo {
		return verifyEDC(sector.Raw, subheaderOffset, form2EDCOffset, true)
	}
	return verifyEDC(sector.Raw, subheaderOffset, form1EDCOffset, false)
}

// NopVerifier performs no verification and accepts every sector.
type NopVerifier struct{}

// Verify implements [SectorVerifier] by reporting success for every sector.
func (NopVerifier) Verify(Sector) error {
	return nil
}

// verifierOrNop returns v, or a [NopVerifier] when v is nil.
func verifierOrNop(v SectorVerifier) SectorVerifier {
	if v == nil {
		return NopVerifier{}
	}
	return v
}

// verifyEDC compares the little-endian EDC stored at edcOffset against the EDC
// computed over raw[start:edcOffset]. When optional is true a stored EDC of zero
// is treated as absent and accepted.
func verifyEDC(raw []byte, start, edcOffset int, optional bool) error {
	if len(raw) < edcOffset+edcSize {
		return fmt.Errorf("%w: need %d bytes for EDC, have %d", ErrShortSector, edcOffset+edcSize, len(raw))
	}
	stored := binary.LittleEndian.Uint32(raw[edcOffset:])
	if optional && stored == 0 {
		return nil
	}
	computed := edc.Compute(raw[start:edcOffset])
	if stored != computed {
		return fmt.Errorf("%w: stored %#08x, computed %#08x", ErrChecksum, stored, computed)
	}
	return nil
}

var (
	_ SectorVerifier = Mode1Verifier{}
	_ SectorVerifier = Mode2Verifier{}
	_ SectorVerifier = NopVerifier{}
)
