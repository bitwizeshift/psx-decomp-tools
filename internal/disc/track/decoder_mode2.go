package track

import "fmt"

// Mode2Decoder decodes CD-ROM XA Mode 2 sectors. A SectorSize of [rawSectorSize]
// selects the full layout with sync, header, subheader, and error-detection
// regions; a SectorSize of [bareMode2Size] selects a bare layout that is
// entirely user data.
type Mode2Decoder struct {
	// SectorSize is the on-disc size of the sectors this decoder reads.
	SectorSize int

	// Verifier validates the decoded sector. A nil Verifier disables
	// verification.
	Verifier SectorVerifier
}

// DecodeSector decodes a Mode 2 sector. For the full layout it decodes the
// subheader, slices the user data for the detected [Form], validates the header,
// and verifies the sector, returning a best-effort [Sector] with
// [ErrShortSector], [ErrBadSync], [ErrInvalidBCD], [ErrInvalidSectorMode], or
// [ErrChecksum] on failure.
func (d Mode2Decoder) DecodeSector(sector RawSector) (Sector, error) {
	if d.SectorSize == bareMode2Size {
		return bareSector(sector), nil
	}
	decoded := Sector{Number: sector.Number, Raw: sector.Data}
	if len(sector.Data) < rawSectorSize {
		return decoded, fmt.Errorf("%w: sector %d has %d of %d bytes", ErrShortSector, sector.Number, len(sector.Data), rawSectorSize)
	}
	subheader := decodeSubheader(sector.Data)
	decoded.Subheader = subheader
	decoded.UserData = mode2UserData(sector.Data, subheader.Form)
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

// mode2UserData returns the user-data region of a full Mode 2 sector for the
// given form.
func mode2UserData(raw []byte, form Form) []byte {
	if form == FormTwo {
		return raw[mode2DataOffset:form2DataEnd]
	}
	return raw[mode2DataOffset:form1DataEnd]
}

var _ SectorDecoder = Mode2Decoder{}
