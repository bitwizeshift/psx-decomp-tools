package cue

import (
	"encoding"
	"fmt"
)

// Type identifies the binary layout of a FILE referenced by a CUE sheet, taken
// from the type token on a FILE line (for example [TypeBinary]).
type Type int

// The set of FILE types defined by the CUE format.
const (
	TypeBinary   Type = iota // BINARY: little-endian binary data.
	TypeMotorola             // MOTOROLA: big-endian binary data.
	TypeAIFF                 // AIFF: audio in AIFF format.
	TypeWave                 // WAVE: audio in WAVE format.
	TypeMP3                  // MP3: audio in MP3 format.
)

var typeNames = map[Type]string{
	TypeBinary:   "BINARY",
	TypeMotorola: "MOTOROLA",
	TypeAIFF:     "AIFF",
	TypeWave:     "WAVE",
	TypeMP3:      "MP3",
}

var typesByName = namesToValues(typeNames)

// String returns the canonical CUE text for t.
func (t Type) String() string {
	return enumString(typeNames, t, "Type")
}

// MarshalText returns the canonical CUE text for t.
func (t Type) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText resolves the canonical CUE text in text into t. It returns
// [ErrInvalidType] if text is not a recognized FILE type.
func (t *Type) UnmarshalText(text []byte) error {
	value, err := enumValue(typesByName, text, ErrInvalidType)
	if err != nil {
		return err
	}
	*t = value
	return nil
}

var (
	_ fmt.Stringer             = Type(0)
	_ encoding.TextMarshaler   = Type(0)
	_ encoding.TextUnmarshaler = (*Type)(nil)
)
