package isotest_test

import (
	"bytes"
	"errors"
	"strconv"
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/iso"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/iso/isotest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// collector is an [iso.Visitor] that records the path and contents of every file.
type collector struct {
	iso.BaseVisitor
	files map[string]string
}

func (c *collector) VisitFile(f *iso.File, s *iso.FileStream) error {
	var data []byte
	listener := iso.ListenerFunc(func(chunk []byte, _ *iso.Subheader) error {
		data = append(data, chunk...)
		return nil
	})
	if err := s.Stream(listener); err != nil {
		return err
	}
	c.files[f.Path] = string(data)
	return nil
}

// extract visits image and returns its files keyed by absolute path, failing the
// test on any visit error.
func extract(t *testing.T, image []byte) map[string]string {
	t.Helper()
	c := &collector{files: map[string]string{}}
	if err := iso.FromReaderAt(bytes.NewReader(image)).Visit(c); err != nil {
		t.Fatalf("Visit(...) = unexpected error %v", err)
	}
	return c.files
}

func TestBuilderRoundTrip(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		image []byte
		want  map[string]string
	}{
		{
			name:  "RootFile",
			image: isotest.New().AddFile("/SYSTEM.CNF", []byte("boot")).Build(),
			want:  map[string]string{"/SYSTEM.CNF": "boot"},
		},
		{
			name:  "NestedDirectories",
			image: isotest.New().AddFile("/A/B/DEEP.TXT", []byte("deep")).Build(),
			want:  map[string]string{"/A/B/DEEP.TXT": "deep"},
		},
		{
			name:  "SharedDirectory",
			image: isotest.New().AddFile("/DIR/A.TXT", []byte("a")).AddFile("/DIR/B.TXT", []byte("b")).Build(),
			want:  map[string]string{"/DIR/A.TXT": "a", "/DIR/B.TXT": "b"},
		},
		{
			name:  "ExplicitEmptyDirectory",
			image: isotest.New().AddDir("/EMPTY").AddFile("/A.TXT", []byte("a")).Build(),
			want:  map[string]string{"/A.TXT": "a"},
		},
		{
			name:  "EmptyFileOmitted",
			image: isotest.New().AddFile("/ZERO.DAT", nil).AddFile("/A.TXT", []byte("a")).Build(),
			want:  map[string]string{"/A.TXT": "a"},
		},
		{
			name:  "WithGap",
			image: isotest.New().AddFile("/A.TXT", []byte("a")).Gap(2).Build(),
			want:  map[string]string{"/A.TXT": "a"},
		},
		{
			name:  "WithTrailing",
			image: isotest.New().AddFile("/A.TXT", []byte("a")).Trailing([]byte("LEFTOVER")).Build(),
			want:  map[string]string{"/A.TXT": "a"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange / Act
			files := extract(t, tc.image)

			// Assert
			if got, want := files, tc.want; !cmp.Equal(got, want) {
				t.Errorf("Build(...) files = mismatch (-want +got):\n%s", cmp.Diff(want, got))
			}
		})
	}
}

func TestBuilderManyEntries(t *testing.T) {
	t.Parallel()

	// Arrange
	const count = 60
	builder := isotest.New()
	want := map[string]string{}
	for i := range count {
		name := "/FILE" + strconv.Itoa(i) + ".TXT"
		contents := strconv.Itoa(i)
		builder.AddFile(name, []byte(contents))
		want[name] = contents
	}

	// Act
	files := extract(t, builder.Build())

	// Assert
	if got := files; !cmp.Equal(got, want) {
		t.Errorf("Build(...) files = mismatch (-want +got):\n%s", cmp.Diff(want, got))
	}
}

func TestBuilderTrailingBytes(t *testing.T) {
	t.Parallel()

	// Arrange
	plain := isotest.New(
		isotest.File("/A.TXT", []byte("a")),
	).Build()

	// Act
	withTrailing := isotest.New(
		isotest.File("/A.TXT", []byte("a")),
		isotest.Trailing([]byte("LEFTOVER")),
	).Build()

	// Assert
	if got, want := len(withTrailing), len(plain)+len("LEFTOVER"); !cmp.Equal(got, want) {
		t.Errorf("Build(...) trailing size = %d, want %d", got, want)
	}
	if got, want := withTrailing[len(plain):], []byte("LEFTOVER"); !cmp.Equal(got, want) {
		t.Errorf("Build(...) trailing bytes = %q, want %q", got, want)
	}
}

func TestBuilderCorruptions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		image   []byte
		wantErr error
	}{
		{
			name: "ZeroBlockSize",
			image: isotest.New(
				isotest.File("/A.TXT", []byte("a")),
			).BuildZeroBlockSize(),
			wantErr: iso.ErrCorruptImage,
		},
		{
			name: "ShortDirectoryRecord",
			image: isotest.New(
				isotest.File("/A.TXT", []byte("a")),
			).BuildShortDirectoryRecord(),
			wantErr: iso.ErrCorruptImage,
		},
		{
			name: "OverlongIdentifier",
			image: isotest.New(
				isotest.File("/A.TXT", []byte("a")),
			).BuildOverlongIdentifier(),
			wantErr: iso.ErrCorruptImage,
		},
		{
			name: "TruncatedPathTableRecord",
			image: isotest.New(
				isotest.File("/A.TXT", []byte("a")),
			).BuildTruncatedPathTableRecord(),
			wantErr: iso.ErrTruncated,
		},
		{
			name: "ZeroPathTableRecord",
			image: isotest.New(
				isotest.File("/A.TXT", []byte("a")),
			).BuildZeroPathTableRecord(),
			wantErr: iso.ErrCorruptImage,
		},
		{
			name: "BadRootRecord",
			image: isotest.New(
				isotest.File("/A.TXT", []byte("a")),
			).BuildBadRootRecord(),
			wantErr: iso.ErrCorruptImage,
		},
		{
			name: "PathTableBeyondEnd",
			image: isotest.New(
				isotest.File("/A.TXT", []byte("a")),
			).BuildPathTableBeyondEnd(),
			wantErr: iso.ErrTruncated,
		},
		{
			name:    "SubdirectoryBeyondEnd",
			image:   isotest.New(isotest.Dir("/DIR")).BuildSubdirectoryBeyondEnd(),
			wantErr: iso.ErrTruncated,
		},
		{
			name:    "NoPrimary",
			image:   isotest.NoPrimaryImage(),
			wantErr: iso.ErrNoPrimaryDescriptor,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := iso.FromReaderAt(bytes.NewReader(tc.image))

			// Act
			err := sut.Visit(&collector{files: map[string]string{}})

			// Assert
			opts := cmpopts.EquateErrors()
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, opts) {
				t.Errorf("Visit(...) = error %v, want %v", got, want)
			}
		})
	}
}

func TestBuilderDuplicateExtents(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		image []byte
		want  map[string]string
	}{
		{
			name: "DuplicateFileExtent",
			image: isotest.New(
				isotest.File("/A.TXT", []byte("aa")),
				isotest.File("/B.TXT", []byte("bb")),
			).BuildDuplicateFileExtent(),
			want: map[string]string{"/A.TXT": "bb"},
		},
		{
			name: "DuplicateDirectoryExtent",
			image: isotest.New(
				isotest.Dir("/ONE"),
				isotest.Dir("/TWO"),
			).BuildDuplicateDirectoryExtent(),
			want: map[string]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange / Act
			files := extract(t, tc.image)

			// Assert
			if got, want := files, tc.want; !cmp.Equal(got, want) {
				t.Errorf("Build(...) files = mismatch (-want +got):\n%s", cmp.Diff(want, got))
			}
		})
	}
}

func TestErrorReaderAt(t *testing.T) {
	t.Parallel()

	// Arrange
	sentinel := errors.New("read failure")
	sut := isotest.ErrorReaderAt{Err: sentinel}
	buf := make([]byte, 8)

	// Act
	n, err := sut.ReadAt(buf, 0)

	// Assert
	if got, want := err, sentinel; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("ReadAt(...) = error %v, want %v", got, want)
	}
	if got, want := n, 0; !cmp.Equal(got, want) {
		t.Errorf("ReadAt(...) = %d bytes, want %d", got, want)
	}
}

func TestTailErrorReaderAt(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("tail failure")

	testCases := []struct {
		name     string
		offset   int64
		size     int
		wantData string
		wantErr  error
	}{
		{
			name:     "WithinData",
			offset:   0,
			size:     5,
			wantData: "hello",
			wantErr:  nil,
		},
		{
			name:     "SpanningEnd",
			offset:   8,
			size:     5,
			wantData: "rld",
			wantErr:  sentinel,
		},
		{
			name:     "PastEnd",
			offset:   11,
			size:     4,
			wantData: "",
			wantErr:  sentinel,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := isotest.TailErrorReaderAt{Data: []byte("hello world"), Err: sentinel}
			buf := make([]byte, tc.size)

			// Act
			n, err := sut.ReadAt(buf, tc.offset)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("ReadAt(...) = error %v, want %v", got, want)
			}
			if got, want := string(buf[:n]), tc.wantData; !cmp.Equal(got, want) {
				t.Errorf("ReadAt(...) = data %q, want %q", got, want)
			}
		})
	}
}
