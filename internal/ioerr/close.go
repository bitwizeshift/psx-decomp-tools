package ioerr

import "io"

// CloseAndReport attempts to close the provided [io.Closer], and will zero the
// output value and set the output error if closing fails.
//
// This is intended to be used in a defer statement to ensure that any error from
// closing is properly reported, and that the output value is not used if closing
// fails. For example:
//
//	defer ioerr.CloseAndReport(f, &value, &err)
func CloseAndReport[T any](closer io.Closer, outVal *T, outErr *error) {
	if err := closer.Close(); err != nil {
		*outErr = err
		*outVal = *new(T)
	}
}
