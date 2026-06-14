package track

import (
	"fmt"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/cue"
)

// sectorSizes maps each supported [cue.Mode] to its on-disc sector size in
// bytes.
var sectorSizes = map[cue.Mode]int{
	cue.ModeAudio:      2352,
	cue.ModeCDG:        2352,
	cue.ModeMode1_2048: 2048,
	cue.ModeMode1_2352: 2352,
	cue.ModeMode2_2336: 2336,
	cue.ModeMode2_2352: 2352,
	cue.ModeCDI_2336:   2336,
	cue.ModeCDI_2352:   2352,
}

// SectorSize returns the on-disc size, in bytes, of a single sector for a track
// of the given mode. It returns [ErrUnsupportedMode] if mode has no known sector
// layout.
func SectorSize(mode cue.Mode) (int, error) {
	size, ok := sectorSizes[mode]
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrUnsupportedMode, mode)
	}
	return size, nil
}
