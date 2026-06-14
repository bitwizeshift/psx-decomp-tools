package track

import "fmt"

// Mode1Decoder decodes CD-ROM Mode 1 sectors. A SectorSize of [rawSectorSize]
// selects the full layout with sync, header, and error-detection regions; a
// SectorSize of [bareMode1Size] selects a bare layout that is entirely user
// data.
type Mode1Decoder struct {
	// SectorSize is the on-disc size of the sectors this decoder reads.
	SectorSize int

	// Verifier validates the decoded sector. A nil Verifier disables
	// verification.
	Verifier SectorVerifier
}

// DecodeSector decodes a Mode 1 sector. For the full layout it validates the
// header and verifies the sector, returning a best-effort [Sector] with
// [ErrShortSector], [ErrBadSync], [ErrInvalidBCD], [ErrInvalidSectorMode], or
// [ErrChecksum] on failure.
func (d Mode1Decoder) DecodeSector(sector RawSector) (Sector, error) {
	if d.SectorSize == bareMode1Size {
		return bareSector(sector), nil
	}
	decoded := Sector{Number: sector.Number, Raw: sector.Data}
	if len(sector.Data) < rawSectorSize {
		return decoded, fmt.Errorf("%w: sector %d has %d of %d bytes", ErrShortSector, sector.Number, len(sector.Data), rawSectorSize)
	}
	decoded.UserData = sector.Data[mode1DataOffset:mode1DataEnd]
	header, err := decodeHeader(sector.Data)
	if err != nil {
		return decoded, err
	}
	decoded.Header = header
	if err := verifierOrNop(d.Verifier).Verify(decoded); err != nil {
		return decoded, err
	}
	return decoded, nil
}

var _ SectorDecoder = Mode1Decoder{}
