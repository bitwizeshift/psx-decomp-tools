package cd

import "errors"

// Sentinel errors reported while parsing a ".CD" archive.
var (
	// ErrTruncated indicates the archive ended before its table of contents could
	// be read in full.
	ErrTruncated = errors.New("cd: truncated archive")

	// ErrCorrupt indicates a table of contents that is internally inconsistent,
	// such as a member whose extent runs past the end of the archive or one that
	// does not begin where the previous member's sectors end.
	ErrCorrupt = errors.New("cd: corrupt archive")
)
