package track

import "errors"

// Sentinel errors reported while reading and decoding track sectors.
var (
	// ErrShortSector indicates a sector could not be read in full because the
	// underlying data ended partway through it.
	ErrShortSector = errors.New("track: short sector")

	// ErrBadSync indicates a sector's 12-byte sync pattern does not match the
	// expected CD-ROM value.
	ErrBadSync = errors.New("track: invalid sync pattern")

	// ErrInvalidSectorMode indicates a sector header's mode byte is not a
	// recognized CD-ROM sector mode.
	ErrInvalidSectorMode = errors.New("track: invalid sector mode")

	// ErrInvalidBCD indicates a sector header field is not valid binary-coded
	// decimal.
	ErrInvalidBCD = errors.New("track: invalid BCD value")

	// ErrChecksum indicates a sector's stored EDC does not match its computed
	// value.
	ErrChecksum = errors.New("track: checksum mismatch")

	// ErrUnsupportedMode indicates a [cue.Mode] has no known sector layout.
	ErrUnsupportedMode = errors.New("track: unsupported track mode")
)
