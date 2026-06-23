package psexe

import "errors"

var (
	// ErrInvalidHeader indicates content that does not begin with a complete
	// PS-X EXE header or that carries the wrong magic tag.
	ErrInvalidHeader = errors.New("psexe: invalid header")

	// ErrTruncated indicates a valid header whose declared text payload runs past
	// the end of the stream.
	ErrTruncated = errors.New("psexe: truncated executable")
)
