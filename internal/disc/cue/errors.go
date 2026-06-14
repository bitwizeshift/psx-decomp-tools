package cue

import "errors"

// Sentinel errors reported while parsing a CUE sheet.
var (
	// ErrSyntax indicates a structurally malformed or misplaced line, such as a
	// command with the wrong number of arguments or one that appears before the
	// FILE or TRACK it depends on.
	ErrSyntax = errors.New("cue: syntax error")

	// ErrInvalidType indicates an unrecognized FILE type.
	ErrInvalidType = errors.New("cue: invalid file type")

	// ErrInvalidMode indicates an unrecognized TRACK mode.
	ErrInvalidMode = errors.New("cue: invalid track mode")

	// ErrInvalidFlag indicates an unrecognized FLAGS value.
	ErrInvalidFlag = errors.New("cue: invalid flag")

	// ErrInvalidMSF indicates a malformed MM:SS:FF timecode.
	ErrInvalidMSF = errors.New("cue: invalid MSF timecode")
)
