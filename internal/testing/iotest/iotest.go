package iotest

import (
	"io"
	"testing"
)

type errWriter struct {
	err error
}

func ErrWriter(err error) io.Writer {
	return &errWriter{err: err}
}

func (w *errWriter) Write(p []byte) (int, error) {
	return 0, w.err
}

type truncateErrWriter struct {
	limit   int
	written int
	err     error
}

func (w *truncateErrWriter) Write(p []byte) (int, error) {
	if w.written >= w.limit {
		return 0, w.err
	}
	n := min(len(p), w.limit-w.written)
	w.written += n
	if n < len(p) {
		return n, w.err
	}
	return n, nil
}

// TruncateErrWriter returns an io.Writer that writes at most limit bytes, then
// returns err for all subsequent writes.
func TruncateErrWriter(limit int, err error) io.Writer {
	return &truncateErrWriter{limit: limit, err: err}
}

// ReadAllFromWriter reads all bytes from w if it implements [io.Reader]. If it
// doesn't, it returns nil. The test will fail with a fatal error if reading
// fails.
//
// This utility exists to make it easier to read from [strings.Builder] and
// [bytes.Buffer] in table-tests that might also be using [ErrWriter]-like
// writers.
func ReadAllFromWriter(t testing.TB, w io.Writer) []byte {
	t.Helper()
	r, ok := w.(io.Reader)
	if !ok {
		return nil
	}
	bytes, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAllFromWriter: %v", err)
		return nil
	}
	return bytes
}
