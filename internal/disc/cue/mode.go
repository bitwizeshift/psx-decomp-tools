package cue

import (
	"encoding"
	"fmt"
)

// Mode identifies the sector layout of a track, taken from the mode token on a
// TRACK line (for example [ModeAudio] or [ModeMode2_2352]).
type Mode int

// The set of TRACK modes defined by the CUE format.
const (
	ModeAudio      Mode = iota // AUDIO: audio data.
	ModeCDG                    // CDG: karaoke CD+G data.
	ModeMode1_2048             // MODE1/2048: CD-ROM Mode 1 user data.
	ModeMode1_2352             // MODE1/2352: CD-ROM Mode 1 raw sectors.
	ModeMode2_2336             // MODE2/2336: CD-ROM XA Mode 2 user data.
	ModeMode2_2352             // MODE2/2352: CD-ROM XA Mode 2 raw sectors.
	ModeCDI_2336               // CDI/2336: CD-i Mode 2 user data.
	ModeCDI_2352               // CDI/2352: CD-i raw sectors.
)

var modeNames = map[Mode]string{
	ModeAudio:      "AUDIO",
	ModeCDG:        "CDG",
	ModeMode1_2048: "MODE1/2048",
	ModeMode1_2352: "MODE1/2352",
	ModeMode2_2336: "MODE2/2336",
	ModeMode2_2352: "MODE2/2352",
	ModeCDI_2336:   "CDI/2336",
	ModeCDI_2352:   "CDI/2352",
}

var modesByName = namesToValues(modeNames)

// String returns the canonical CUE text for m.
func (m Mode) String() string {
	return enumString(modeNames, m, "Mode")
}

// MarshalText returns the canonical CUE text for m.
func (m Mode) MarshalText() ([]byte, error) {
	return []byte(m.String()), nil
}

// UnmarshalText resolves the canonical CUE text in text into m. It returns
// [ErrInvalidMode] if text is not a recognized TRACK mode.
func (m *Mode) UnmarshalText(text []byte) error {
	value, err := enumValue(modesByName, text, ErrInvalidMode)
	if err != nil {
		return err
	}
	*m = value
	return nil
}

var (
	_ fmt.Stringer             = Mode(0)
	_ encoding.TextMarshaler   = Mode(0)
	_ encoding.TextUnmarshaler = (*Mode)(nil)
)
