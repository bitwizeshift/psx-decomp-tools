package disctest

import (
	"bytes"
	"io"
	"io/fs"
	"time"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc"
)

// OpenFunc returns a [disc.OpenFunc] that serves each named buffer in files as
// a randomly-accessible in-memory file, reporting [fs.ErrNotExist] for any
// other name.
func OpenFunc(files map[string][]byte) disc.OpenFunc {
	return func(name string) (fs.File, error) {
		data, ok := files[name]
		if !ok {
			return nil, fs.ErrNotExist
		}
		return &memFile{Reader: bytes.NewReader(data), info: infoFor(name, data)}, nil
	}
}

// StatFunc returns a [disc.StatFunc] that reports the size of each named buffer
// in files, reporting [fs.ErrNotExist] for any other name.
func StatFunc(files map[string][]byte) disc.StatFunc {
	return func(name string) (fs.FileInfo, error) {
		data, ok := files[name]
		if !ok {
			return nil, fs.ErrNotExist
		}
		return infoFor(name, data), nil
	}
}

// ErrOpenFunc returns a [disc.OpenFunc] that always fails with err.
func ErrOpenFunc(err error) disc.OpenFunc {
	return func(string) (fs.File, error) {
		return nil, err
	}
}

// ErrStatFunc returns a [disc.StatFunc] that always fails with err.
func ErrStatFunc(err error) disc.StatFunc {
	return func(string) (fs.FileInfo, error) {
		return nil, err
	}
}

// SequentialOpenFunc returns a [disc.OpenFunc] whose files support only
// sequential reads, not random access, driving disc's [disc.ErrRandomAccess]
// path. It reports [fs.ErrNotExist] for names absent from files.
func SequentialOpenFunc(files map[string][]byte) disc.OpenFunc {
	return func(name string) (fs.File, error) {
		data, ok := files[name]
		if !ok {
			return nil, fs.ErrNotExist
		}
		return &seqFile{reader: bytes.NewReader(data), info: infoFor(name, data)}, nil
	}
}

// CloseErrOpenFunc returns a [disc.OpenFunc] that serves each named buffer in
// files but whose files fail to close with err, driving disc's close-error
// handling. It reports [fs.ErrNotExist] for names absent from files.
func CloseErrOpenFunc(files map[string][]byte, err error) disc.OpenFunc {
	return func(name string) (fs.File, error) {
		data, ok := files[name]
		if !ok {
			return nil, fs.ErrNotExist
		}
		return &memFile{Reader: bytes.NewReader(data), info: infoFor(name, data), closeErr: err}, nil
	}
}

// memFile is a randomly-accessible in-memory [fs.File].
type memFile struct {
	*bytes.Reader
	info     fileInfo
	closeErr error
}

func (f *memFile) Stat() (fs.FileInfo, error) { return f.info, nil }

func (f *memFile) Close() error { return f.closeErr }

var _ fs.File = (*memFile)(nil)
var _ io.ReaderAt = (*memFile)(nil)

// seqFile is an in-memory [fs.File] that supports only sequential reads.
type seqFile struct {
	reader io.Reader
	info   fileInfo
}

func (f *seqFile) Read(p []byte) (int, error) { return f.reader.Read(p) }

func (f *seqFile) Stat() (fs.FileInfo, error) { return f.info, nil }

func (f *seqFile) Close() error { return nil }

var _ fs.File = (*seqFile)(nil)

// infoFor returns the [fs.FileInfo] describing a buffer of named in-memory data.
func infoFor(name string, data []byte) fileInfo {
	return fileInfo{name: name, size: int64(len(data))}
}

// fileInfo is the [fs.FileInfo] of an in-memory file.
type fileInfo struct {
	name string
	size int64
}

func (i fileInfo) Name() string       { return i.name }
func (i fileInfo) Size() int64        { return i.size }
func (i fileInfo) Mode() fs.FileMode  { return 0 }
func (i fileInfo) ModTime() time.Time { return time.Time{} }
func (i fileInfo) IsDir() bool        { return false }
func (i fileInfo) Sys() any           { return nil }

var _ fs.FileInfo = fileInfo{}
