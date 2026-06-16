package disc

import "errors"

// Sentinel errors reported while opening tracks and their backing files.
var (
	// ErrTrackNotFound indicates the disc has no track with the requested
	// number.
	ErrTrackNotFound = errors.New("disc: track not found")

	// ErrRandomAccess indicates a backing file opened by an [OpenFunc] does not
	// support random access, meaning it does not implement [io.ReaderAt].
	ErrRandomAccess = errors.New("disc: backing file does not support random access")

	// ErrNegativeOffset indicates a read or seek was asked for a negative
	// payload offset.
	ErrNegativeOffset = errors.New("disc: negative offset")

	// ErrWhence indicates a seek was asked with an unrecognized whence value.
	ErrWhence = errors.New("disc: invalid whence")
)
