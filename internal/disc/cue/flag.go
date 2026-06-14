package cue

import (
	"encoding"
	"fmt"
)

// Flag is a subcode flag declared on a FLAGS line for a track (for example
// [FlagDCP]).
type Flag int

// The set of FLAGS values defined by the CUE format.
const (
	FlagDCP  Flag = iota // DCP: digital copy permitted.
	Flag4CH              // 4CH: four-channel audio.
	FlagPRE              // PRE: pre-emphasis enabled.
	FlagSCMS             // SCMS: serial copy management system.
)

var flagNames = map[Flag]string{
	FlagDCP:  "DCP",
	Flag4CH:  "4CH",
	FlagPRE:  "PRE",
	FlagSCMS: "SCMS",
}

var flagsByName = namesToValues(flagNames)

// String returns the canonical CUE text for f.
func (f Flag) String() string {
	return enumString(flagNames, f, "Flag")
}

// MarshalText returns the canonical CUE text for f.
func (f Flag) MarshalText() ([]byte, error) {
	return []byte(f.String()), nil
}

// UnmarshalText resolves the canonical CUE text in text into f. It returns
// [ErrInvalidFlag] if text is not a recognized FLAGS value.
func (f *Flag) UnmarshalText(text []byte) error {
	value, err := enumValue(flagsByName, text, ErrInvalidFlag)
	if err != nil {
		return err
	}
	*f = value
	return nil
}

var (
	_ fmt.Stringer             = Flag(0)
	_ encoding.TextMarshaler   = Flag(0)
	_ encoding.TextUnmarshaler = (*Flag)(nil)
)
