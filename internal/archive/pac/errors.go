package pac

import "errors"

// Sentinel errors reported while parsing a PAC archive.
var (
	// ErrBadMagic indicates a node whose 16-byte header does not begin with the
	// required "PAC\x00" tag.
	ErrBadMagic = errors.New("pac: bad magic")

	// ErrTruncated indicates the archive ended before a node's header or payload
	// could be read in full.
	ErrTruncated = errors.New("pac: truncated archive")
)
