package psexe

import (
	"encoding"
	"errors"
	"fmt"
)

// ErrBadSection indicates a section name that does not match a known
// [SectionKind].
var ErrBadSection = errors.New("psexe: unknown section")

// sectionNames maps each [SectionKind] to its lowercase name.
var sectionNames = map[SectionKind]string{
	Text:  "text",
	Data:  "data",
	BSS:   "bss",
	Stack: "stack",
}

// String returns the lowercase name of the section, or "unknown" when k is not a
// recognized [SectionKind].
func (k SectionKind) String() string {
	if name, ok := sectionNames[k]; ok {
		return name
	}
	return "unknown"
}

var _ fmt.Stringer = (*SectionKind)(nil)

// MarshalText returns the section name. It always succeeds for a recognized
// [SectionKind].
func (k SectionKind) MarshalText() ([]byte, error) {
	if _, ok := sectionNames[k]; !ok {
		return nil, fmt.Errorf("psexe: %d: %w", int(k), ErrBadSection)
	}
	return []byte(k.String()), nil
}

var _ encoding.TextMarshaler = (*SectionKind)(nil)

// UnmarshalText sets k from a section name. It reports [ErrBadSection] for an
// unrecognized name.
func (k *SectionKind) UnmarshalText(text []byte) error {
	for kind, name := range sectionNames {
		if name == string(text) {
			*k = kind
			return nil
		}
	}
	return fmt.Errorf("psexe: %q: %w", text, ErrBadSection)
}

var _ encoding.TextUnmarshaler = (*SectionKind)(nil)
