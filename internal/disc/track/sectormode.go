package track

import "fmt"

// SectorMode identifies the data mode of a CD-ROM sector, taken from the mode
// byte at offset 15 of a raw sector header. Its values match the on-disc mode
// byte.
type SectorMode int

const (
	// SectorModeBlank is a blank or reserved sector (mode byte 0x00).
	SectorModeBlank SectorMode = iota
	// SectorModeMode1 is a CD-ROM Mode 1 sector (mode byte 0x01).
	SectorModeMode1
	// SectorModeMode2 is a CD-ROM XA Mode 2 sector (mode byte 0x02).
	SectorModeMode2
)

var sectorModeNames = map[SectorMode]string{
	SectorModeBlank: "Blank",
	SectorModeMode1: "Mode1",
	SectorModeMode2: "Mode2",
}

// String returns a human-readable name for m, falling back to a "SectorMode(n)"
// form for values outside the known set.
func (m SectorMode) String() string {
	if name, ok := sectorModeNames[m]; ok {
		return name
	}
	return fmt.Sprintf("SectorMode(%d)", int(m))
}

// parseSectorMode converts a raw header mode byte into a [SectorMode]. It
// returns [ErrInvalidSectorMode] if b is not a recognized mode.
func parseSectorMode(b byte) (SectorMode, error) {
	mode := SectorMode(b)
	if _, ok := sectorModeNames[mode]; !ok {
		return SectorModeBlank, fmt.Errorf("%w: %#02x", ErrInvalidSectorMode, b)
	}
	return mode, nil
}

var _ fmt.Stringer = SectorMode(0)
