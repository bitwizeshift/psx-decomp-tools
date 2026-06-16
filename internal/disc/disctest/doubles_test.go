package disctest_test

import (
	"io"
	"io/fs"
	"testing"
	"time"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/disctest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func payload() []byte {
	return []byte("in-memory sector data")
}

func TestOpenFuncMissingName(t *testing.T) {
	t.Parallel()

	files := map[string][]byte{"present.bin": payload()}

	testCases := []struct {
		name string
		open disc.OpenFunc
	}{
		{
			name: "RandomAccess",
			open: disctest.OpenFunc(files),
		},
		{
			name: "Sequential",
			open: disctest.SequentialOpenFunc(files),
		},
		{
			name: "CloseError",
			open: disctest.CloseErrOpenFunc(files, fs.ErrClosed),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Act
			file, err := tc.open("absent.bin")

			// Assert
			if got, want := err, fs.ErrNotExist; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("open(absent.bin) = error %v, want %v", got, want)
			}
			if got, want := file, fs.File(nil); !cmp.Equal(got, want) {
				t.Errorf("open(absent.bin) = %v, want %v", got, want)
			}
		})
	}
}

func TestOpenFuncServesData(t *testing.T) {
	t.Parallel()

	// Arrange
	want := payload()
	sut := disctest.OpenFunc(map[string][]byte{"present.bin": want})

	// Act
	file, openErr := sut("present.bin")
	data, readErr := io.ReadAll(file)
	closeErr := file.Close()

	// Assert
	if got, want := openErr, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("open(present.bin) = error %v, want %v", got, want)
	}
	if got, want := readErr, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("io.ReadAll(file) = error %v, want %v", got, want)
	}
	if got, want := closeErr, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("file.Close() = error %v, want %v", got, want)
	}
	if got, want := data, want; !cmp.Equal(got, want) {
		t.Errorf("io.ReadAll(file) = mismatch (-want +got):\n%s", cmp.Diff(want, got))
	}
}

func TestOpenFuncFileInfo(t *testing.T) {
	t.Parallel()

	// Arrange
	data := payload()
	sut := disctest.OpenFunc(map[string][]byte{"present.bin": data})

	// Act
	file, openErr := sut("present.bin")
	info, statErr := file.Stat()

	// Assert
	if got, want := openErr, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("open(present.bin) = error %v, want %v", got, want)
	}
	if got, want := statErr, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("file.Stat() = error %v, want %v", got, want)
	}
	if got, want := info.Name(), "present.bin"; !cmp.Equal(got, want) {
		t.Errorf("FileInfo.Name() = %q, want %q", got, want)
	}
	if got, want := info.Size(), int64(len(data)); !cmp.Equal(got, want) {
		t.Errorf("FileInfo.Size() = %d, want %d", got, want)
	}
	if got, want := info.Mode(), fs.FileMode(0); !cmp.Equal(got, want) {
		t.Errorf("FileInfo.Mode() = %v, want %v", got, want)
	}
	if got, want := info.ModTime(), (time.Time{}); !cmp.Equal(got, want) {
		t.Errorf("FileInfo.ModTime() = %v, want %v", got, want)
	}
	if got, want := info.IsDir(), false; !cmp.Equal(got, want) {
		t.Errorf("FileInfo.IsDir() = %v, want %v", got, want)
	}
	if got, want := info.Sys(), any(nil); !cmp.Equal(got, want) {
		t.Errorf("FileInfo.Sys() = %v, want %v", got, want)
	}
}

func TestStatFunc_KnownFile_ReportsSize(t *testing.T) {
	t.Parallel()

	// Arrange
	data := payload()
	sut := disctest.StatFunc(map[string][]byte{"present.bin": data})

	// Act
	info, err := sut("present.bin")

	// Assert
	if got, want := err, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("stat(present.bin) = error %v, want %v", got, want)
	}
	if got, want := info.Size(), int64(len(data)); !cmp.Equal(got, want) {
		t.Errorf("stat(present.bin).Size() = %d, want %d", got, want)
	}
}

func TestStatFunc_MissingFile_ReturnsNotExist(t *testing.T) {
	t.Parallel()

	// Arrange
	sut := disctest.StatFunc(map[string][]byte{"present.bin": payload()})

	// Act
	info, err := sut("absent.bin")

	// Assert
	if got, want := err, fs.ErrNotExist; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("stat(absent.bin) = error %v, want %v", got, want)
	}
	if got, want := info, fs.FileInfo(nil); !cmp.Equal(got, want) {
		t.Errorf("stat(absent.bin) = %v, want %v", got, want)
	}
}

func TestErrOpenFunc(t *testing.T) {
	t.Parallel()

	// Arrange
	want := fs.ErrPermission
	sut := disctest.ErrOpenFunc(want)

	// Act
	file, err := sut("any.bin")

	// Assert
	if got, want := err, want; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("open(any.bin) = error %v, want %v", got, want)
	}
	if got, want := file, fs.File(nil); !cmp.Equal(got, want) {
		t.Errorf("open(any.bin) = %v, want %v", got, want)
	}
}

func TestErrStatFunc(t *testing.T) {
	t.Parallel()

	// Arrange
	want := fs.ErrPermission
	sut := disctest.ErrStatFunc(want)

	// Act
	info, err := sut("any.bin")

	// Assert
	if got, want := err, want; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("stat(any.bin) = error %v, want %v", got, want)
	}
	if got, want := info, fs.FileInfo(nil); !cmp.Equal(got, want) {
		t.Errorf("stat(any.bin) = %v, want %v", got, want)
	}
}

func TestSequentialOpenFuncServesSequentialFile(t *testing.T) {
	t.Parallel()

	// Arrange
	want := payload()
	sut := disctest.SequentialOpenFunc(map[string][]byte{"present.bin": want})

	// Act
	file, openErr := sut("present.bin")
	_, isReaderAt := file.(io.ReaderAt)
	info, statErr := file.Stat()
	data, readErr := io.ReadAll(file)
	closeErr := file.Close()

	// Assert
	if got, want := openErr, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("open(present.bin) = error %v, want %v", got, want)
	}
	if got, want := statErr, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("file.Stat() = error %v, want %v", got, want)
	}
	if got, want := readErr, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("io.ReadAll(file) = error %v, want %v", got, want)
	}
	if got, want := closeErr, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("file.Close() = error %v, want %v", got, want)
	}
	if got, want := isReaderAt, false; !cmp.Equal(got, want) {
		t.Errorf("file.(io.ReaderAt) ok = %v, want %v", got, want)
	}
	if got, want := info.Name(), "present.bin"; !cmp.Equal(got, want) {
		t.Errorf("file.Stat().Name() = %q, want %q", got, want)
	}
	if got, want := data, want; !cmp.Equal(got, want) {
		t.Errorf("io.ReadAll(file) = mismatch (-want +got):\n%s", cmp.Diff(want, got))
	}
}

func TestCloseErrOpenFuncReportsCloseError(t *testing.T) {
	t.Parallel()

	// Arrange
	want := fs.ErrClosed
	sut := disctest.CloseErrOpenFunc(map[string][]byte{"present.bin": payload()}, want)

	// Act
	file, openErr := sut("present.bin")
	closeErr := file.Close()

	// Assert
	if got, want := openErr, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("open(present.bin) = error %v, want %v", got, want)
	}
	if got, want := closeErr, want; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("file.Close() = error %v, want %v", got, want)
	}
}
