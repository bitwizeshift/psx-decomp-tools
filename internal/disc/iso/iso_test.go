package iso_test

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/iso"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/iso/isotest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// visitEvent is a single recorded [iso.Visitor] callback.
type visitEvent struct {
	Kind   string
	Name   string
	Path   string
	Offset int64
	Length int64
	Block  int64
	Data   string
}

// kindName is the kind and name of a recorded callback, without its location.
type kindName struct {
	Kind string
	Name string
}

// recorder is an [iso.Visitor] that records every callback and reads the bytes of
// every region and file it is given.
type recorder struct {
	iso.BaseVisitor
	events []visitEvent
}

func (r *recorder) VisitSystemArea(region *iso.Region) error {
	io.ReadAll(region.Data)
	r.events = append(r.events, visitEvent{Kind: "system", Offset: region.Offset, Length: region.Length, Block: region.Block})
	return nil
}

func (r *recorder) VisitVolumeDescriptor(d *iso.VolumeDescriptor) error {
	r.events = append(r.events, visitEvent{Kind: "vd", Name: strconv.Itoa(int(d.Type)), Offset: d.Extent.Offset, Length: d.Extent.Length, Block: d.Extent.Block})
	return nil
}

func (r *recorder) VisitPathTableRecord(rec *iso.PathTableRecord) error {
	r.events = append(r.events, visitEvent{Kind: "path", Name: rec.Name, Offset: rec.Extent.Offset, Length: rec.Extent.Length, Block: rec.Extent.Block})
	return nil
}

func (r *recorder) VisitDirectoryRecord(rec *iso.DirectoryRecord) error {
	r.events = append(r.events, visitEvent{Kind: "dir", Name: rec.Name, Offset: rec.Extent.Offset, Length: rec.Extent.Length, Block: rec.Extent.Block})
	return nil
}

func (r *recorder) VisitFile(f *iso.File) error {
	data, _ := io.ReadAll(f.Data)
	r.events = append(r.events, visitEvent{Kind: "file", Name: f.Name, Path: f.Path, Offset: f.Extent.Offset, Length: f.Extent.Length, Block: f.Extent.Block, Data: string(data)})
	return nil
}

func (r *recorder) VisitUnreferenced(region *iso.Region) error {
	io.ReadAll(region.Data)
	r.events = append(r.events, visitEvent{Kind: "gap", Offset: region.Offset, Length: region.Length, Block: region.Block})
	return nil
}

// recordVisit visits image and returns the recorded events, failing the test on
// any visit error.
func recordVisit(t *testing.T, image []byte) []visitEvent {
	t.Helper()
	r := &recorder{}
	if err := iso.New(bytes.NewReader(image)).Visit(r); err != nil {
		t.Fatalf("Visit(...) = unexpected error %v", err)
	}
	return r.events
}

// kindNames projects events to their kinds and names.
func kindNames(events []visitEvent) []kindName {
	out := make([]kindName, len(events))
	for i, e := range events {
		out[i] = kindName{Kind: e.Kind, Name: e.Name}
	}
	return out
}

// assertTiles asserts that events cover [0, size) contiguously, in order, with no
// gap or overlap.
func assertTiles(t *testing.T, events []visitEvent, size int64) {
	t.Helper()
	cursor := int64(0)
	for i, e := range events {
		if e.Offset != cursor {
			t.Errorf("Visit(...) event %d (%s %q) = offset %d, want %d", i, e.Kind, e.Name, e.Offset, cursor)
			return
		}
		if e.Length == iso.ToEOF {
			cursor = size
			continue
		}
		cursor += e.Length
	}
	if got, want := cursor, size; !cmp.Equal(got, want) {
		t.Errorf("Visit(...) covered = %d bytes, want %d", got, want)
	}
}

// fileData collects the path, name, and contents of every file event.
type fileData struct {
	Path string
	Name string
	Data string
}

func TestReaderVisitSequence(t *testing.T) {
	t.Parallel()

	// Arrange
	image := isotest.New().AddFile("/A.TXT", []byte("hi")).Build()

	// Act
	events := recordVisit(t, image)

	// Assert
	want := []kindName{
		{Kind: "system", Name: ""},
		{Kind: "vd", Name: "1"},
		{Kind: "vd", Name: "255"},
		{Kind: "path", Name: "."},
		{Kind: "gap", Name: ""},
		{Kind: "path", Name: "."},
		{Kind: "gap", Name: ""},
		{Kind: "dir", Name: "."},
		{Kind: "dir", Name: ".."},
		{Kind: "dir", Name: "A.TXT"},
		{Kind: "gap", Name: ""},
		{Kind: "file", Name: "A.TXT"},
		{Kind: "gap", Name: ""},
	}
	if got := kindNames(events); !cmp.Equal(got, want) {
		t.Errorf("Visit(...) = mismatch (-want +got):\n%s", cmp.Diff(want, got))
	}
}

func TestReaderVisitTiling(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		image []byte
	}{
		{
			name:  "EmptyRoot",
			image: isotest.New().Build(),
		},
		{
			name:  "SingleFile",
			image: isotest.New().AddFile("/A.TXT", []byte("hello")).Build(),
		},
		{
			name:  "EmptyFile",
			image: isotest.New().AddFile("/EMPTY.DAT", nil).Build(),
		},
		{
			name:  "NestedTree",
			image: isotest.New().AddFile("/SYSTEM.CNF", []byte("boot")).AddFile("/DATA/HELLO.TXT", []byte("hi")).Build(),
		},
		{
			name:  "ExactBlockFile",
			image: isotest.New().AddFile("/BLOCK.BIN", bytes.Repeat([]byte{0x5A}, isotest.BlockSize)).Build(),
		},
		{
			name:  "WithGap",
			image: isotest.New().AddFile("/A.TXT", []byte("x")).Gap(3).Build(),
		},
		{
			name:  "WithTrailing",
			image: isotest.New().AddFile("/A.TXT", []byte("x")).Trailing([]byte("LEFTOVER")).Build(),
		},
		{
			name:  "ManyEntries",
			image: manyEntryImage(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			size := int64(len(tc.image))

			// Act
			events := recordVisit(t, tc.image)

			// Assert
			assertTiles(t, events, size)
		})
	}
}

// manyEntryImage builds an image whose root directory holds enough files that its
// records span more than one logical block.
func manyEntryImage() []byte {
	b := isotest.New()
	for i := range 80 {
		b.AddFile("/FILE"+strconv.Itoa(i)+".TXT", []byte{byte(i)})
	}
	return b.Build()
}

func TestReaderVisitFiles(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		image []byte
		want  []fileData
	}{
		{
			name:  "RootFile",
			image: isotest.New().AddFile("/SYSTEM.CNF", []byte("BOOT")).Build(),
			want: []fileData{
				{Path: "/SYSTEM.CNF", Name: "SYSTEM.CNF", Data: "BOOT"},
			},
		},
		{
			name:  "NestedFiles",
			image: isotest.New().AddFile("/SYSTEM.CNF", []byte("boot")).AddFile("/DATA/HELLO.TXT", []byte("hello")).Build(),
			want: []fileData{
				{Path: "/SYSTEM.CNF", Name: "SYSTEM.CNF", Data: "boot"},
				{Path: "/DATA/HELLO.TXT", Name: "HELLO.TXT", Data: "hello"},
			},
		},
		{
			name:  "EmptyFileOmitted",
			image: isotest.New().AddFile("/EMPTY.DAT", nil).AddFile("/A.TXT", []byte("a")).Build(),
			want: []fileData{
				{Path: "/A.TXT", Name: "A.TXT", Data: "a"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			events := recordVisit(t, tc.image)

			// Act
			var files []fileData
			for _, e := range events {
				if e.Kind == "file" {
					files = append(files, fileData{Path: e.Path, Name: e.Name, Data: e.Data})
				}
			}

			// Assert
			if got, want := files, tc.want; !cmp.Equal(got, want) {
				t.Errorf("Visit(...) files = mismatch (-want +got):\n%s", cmp.Diff(want, got))
			}
		})
	}
}

func TestReaderVisitErrors(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("read failure")

	testCases := []struct {
		name    string
		reader  io.ReaderAt
		wantErr error
	}{
		{
			name:    "BadMagic",
			reader:  bytes.NewReader(make([]byte, 17*isotest.BlockSize)),
			wantErr: iso.ErrBadMagic,
		},
		{
			name:    "Truncated",
			reader:  bytes.NewReader(make([]byte, 100)),
			wantErr: iso.ErrTruncated,
		},
		{
			name:    "NoPrimary",
			reader:  bytes.NewReader(isotest.NoPrimaryImage()),
			wantErr: iso.ErrNoPrimaryDescriptor,
		},
		{
			name:    "ZeroBlockSize",
			reader:  bytes.NewReader(isotest.New().AddFile("/A.TXT", []byte("x")).BuildZeroBlockSize()),
			wantErr: iso.ErrCorruptImage,
		},
		{
			name:    "ShortDirectoryRecord",
			reader:  bytes.NewReader(isotest.New().AddFile("/A.TXT", []byte("x")).BuildShortDirectoryRecord()),
			wantErr: iso.ErrCorruptImage,
		},
		{
			name:    "OverlongIdentifier",
			reader:  bytes.NewReader(isotest.New().AddFile("/A.TXT", []byte("x")).BuildOverlongIdentifier()),
			wantErr: iso.ErrCorruptImage,
		},
		{
			name:    "TruncatedPathTableRecord",
			reader:  bytes.NewReader(isotest.New().AddFile("/A.TXT", []byte("x")).BuildTruncatedPathTableRecord()),
			wantErr: iso.ErrTruncated,
		},
		{
			name:    "ZeroPathTableRecord",
			reader:  bytes.NewReader(isotest.New().AddFile("/A.TXT", []byte("x")).BuildZeroPathTableRecord()),
			wantErr: iso.ErrCorruptImage,
		},
		{
			name:    "BadRootRecord",
			reader:  bytes.NewReader(isotest.New().AddFile("/A.TXT", []byte("x")).BuildBadRootRecord()),
			wantErr: iso.ErrCorruptImage,
		},
		{
			name:    "PathTableBeyondEnd",
			reader:  bytes.NewReader(isotest.New().AddFile("/A.TXT", []byte("x")).BuildPathTableBeyondEnd()),
			wantErr: iso.ErrTruncated,
		},
		{
			name:    "SubdirectoryBeyondEnd",
			reader:  bytes.NewReader(isotest.New().AddDir("/DIR").BuildSubdirectoryBeyondEnd()),
			wantErr: iso.ErrTruncated,
		},
		{
			name:    "StructuralReadError",
			reader:  isotest.ErrorReaderAt{Err: sentinel},
			wantErr: sentinel,
		},
		{
			name:    "TrailingReadError",
			reader:  isotest.TailErrorReaderAt{Data: isotest.New().AddFile("/BLOCK.BIN", bytes.Repeat([]byte{1}, isotest.BlockSize)).Build(), Err: sentinel},
			wantErr: sentinel,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := iso.New(tc.reader)

			// Act
			err := sut.Visit(&recorder{})

			// Assert
			opts := cmpopts.EquateErrors()
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, opts) {
				t.Errorf("Visit(...) = error %v, want %v", got, want)
			}
		})
	}
}

func TestReaderVisitDedupe(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		image     []byte
		wantFiles int
		wantDirs  int
	}{
		{
			name:      "DuplicateFileExtent",
			image:     isotest.New().AddFile("/A.TXT", []byte("aa")).AddFile("/B.TXT", []byte("bb")).BuildDuplicateFileExtent(),
			wantFiles: 1,
		},
		{
			name:      "DuplicateDirectoryExtent",
			image:     isotest.New().AddDir("/ONE").AddDir("/TWO").BuildDuplicateDirectoryExtent(),
			wantFiles: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			events := recordVisit(t, tc.image)

			// Act
			files := 0
			for _, e := range events {
				if e.Kind == "file" {
					files++
				}
			}

			// Assert
			if got, want := files, tc.wantFiles; !cmp.Equal(got, want) {
				t.Errorf("Visit(...) file count = %d, want %d", got, want)
			}
			assertTiles(t, events, int64(len(tc.image)))
		})
	}
}

func TestReaderVisitAbort(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("abort")

	testCases := []struct {
		name    string
		visitor iso.Visitor
	}{
		{
			name:    "OnSystemArea",
			visitor: &abortVisitor{on: "system", err: sentinel},
		},
		{
			name:    "OnVolumeDescriptor",
			visitor: &abortVisitor{on: "vd", err: sentinel},
		},
		{
			name:    "OnPathTableRecord",
			visitor: &abortVisitor{on: "path", err: sentinel},
		},
		{
			name:    "OnDirectoryRecord",
			visitor: &abortVisitor{on: "dir", err: sentinel},
		},
		{
			name:    "OnFile",
			visitor: &abortVisitor{on: "file", err: sentinel},
		},
		{
			name:    "OnUnreferenced",
			visitor: &abortVisitor{on: "gap", err: sentinel},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			image := isotest.New().AddFile("/A.TXT", []byte("x")).Build()
			sut := iso.New(bytes.NewReader(image))

			// Act
			err := sut.Visit(tc.visitor)

			// Assert
			opts := cmpopts.EquateErrors()
			if got, want := err, sentinel; !cmp.Equal(got, want, opts) {
				t.Errorf("Visit(...) = error %v, want %v", got, want)
			}
		})
	}
}

// abortVisitor returns err from the callback named by on and is a no-op
// otherwise.
type abortVisitor struct {
	iso.BaseVisitor
	on  string
	err error
}

func (a *abortVisitor) VisitSystemArea(*iso.Region) error {
	return a.fail("system")
}
func (a *abortVisitor) VisitVolumeDescriptor(*iso.VolumeDescriptor) error {
	return a.fail("vd")
}
func (a *abortVisitor) VisitPathTableRecord(*iso.PathTableRecord) error {
	return a.fail("path")
}
func (a *abortVisitor) VisitDirectoryRecord(*iso.DirectoryRecord) error {
	return a.fail("dir")
}
func (a *abortVisitor) VisitFile(*iso.File) error {
	return a.fail("file")
}
func (a *abortVisitor) VisitUnreferenced(*iso.Region) error {
	return a.fail("gap")
}

// fail returns the visitor's error when kind matches the callback it is set to
// abort on.
func (a *abortVisitor) fail(kind string) error {
	if a.on == kind {
		return a.err
	}
	return nil
}
