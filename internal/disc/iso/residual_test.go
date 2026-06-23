package iso_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/iso"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/iso/isotest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// fileSector is the logical block of the single file in an image built by
// isotest.New().AddFile(...): system area (0-15), PVD, terminator, two path
// tables, root directory, then the file.
const fileSector = 21

// unexpectedData is a recorded unexpected tail: its block and contents.
type unexpectedData struct {
	Block int64
	Data  string
}

// errAtSource serves view but fails with err at sector at, modelling a read
// failure during streaming or the unexpected pass.
type errAtSource struct {
	view isotest.SectorView
	at   int
	err  error
}

func (s errAtSource) ReadSector(n int) (iso.Sector, error) {
	if n == s.at {
		return iso.Sector{}, s.err
	}
	return s.view.ReadSector(n)
}

// unexpectedsOf returns the unexpected events recorded among events.
func unexpectedsOf(events []visitEvent) []unexpectedData {
	var out []unexpectedData
	for _, e := range events {
		if e.Kind == "unexpected" {
			out = append(out, unexpectedData{Block: e.Block, Data: e.Data})
		}
	}
	return out
}

// fileDataOf returns the recorded raw contents of the named file.
func fileDataOf(events []visitEvent, path string) string {
	for _, e := range events {
		if e.Kind == "file" && e.Path == path {
			return e.Data
		}
	}
	return ""
}

func TestFromSectorSourcerawStreamFile(t *testing.T) {
	t.Parallel()

	// Arrange
	image := isotest.New().AddFile("/A.TXT", []byte("hi")).Build()
	tail := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	source := isotest.SectorView{Image: image, Tails: map[int]isotest.Tail{
		fileSector: {Bytes: tail, IsStream: true},
	}}
	r := &recorder{}

	// Act
	if err := iso.FromSectorSource(source).Visit(r); err != nil {
		t.Fatalf("Visit(...) = unexpected error %v", err)
	}

	// Assert
	if got, want := fileDataOf(r.events, "/A.TXT"), "hi"+string(tail); !cmp.Equal(got, want) {
		t.Errorf("Visit(...) file = data %q, want %q", got, want)
	}
	if got, want := unexpectedsOf(r.events), []unexpectedData(nil); !cmp.Equal(got, want) {
		t.Errorf("Visit(...) unexpected = %v, want %v", got, want)
	}
}

func TestFromSectorSourceUnexpected(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		tails map[int]isotest.Tail
		want  []unexpectedData
	}{
		{
			name:  "NonFileSector",
			tails: map[int]isotest.Tail{5: {Bytes: []byte{1, 2, 3}}},
			want:  []unexpectedData{{Block: 5, Data: string([]byte{1, 2, 3})}},
		},
		{
			name:  "FileSectorNonStreamTail",
			tails: map[int]isotest.Tail{fileSector: {Bytes: []byte{7, 8}, IsStream: false}},
			want:  []unexpectedData{{Block: fileSector, Data: string([]byte{7, 8})}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			image := isotest.New().AddFile("/A.TXT", []byte("hi")).Build()
			source := isotest.SectorView{Image: image, Tails: tc.tails}
			r := &recorder{}

			// Act
			if err := iso.FromSectorSource(source).Visit(r); err != nil {
				t.Fatalf("Visit(...) = unexpected error %v", err)
			}

			// Assert
			if got, want := unexpectedsOf(r.events), tc.want; !cmp.Equal(got, want) {
				t.Errorf("Visit(...) unexpected = mismatch (-want +got):\n%s", cmp.Diff(want, got))
			}
			if got, want := fileDataOf(r.events, "/A.TXT"), "hi"; !cmp.Equal(got, want) {
				t.Errorf("Visit(...) file = data %q, want %q", got, want)
			}
		})
	}
}

func TestFromReaderAtHasNoUnexpected(t *testing.T) {
	t.Parallel()

	// Arrange
	image := isotest.New().AddFile("/A.TXT", []byte("hi")).Build()
	r := &recorder{}

	// Act
	if err := iso.FromReaderAt(bytes.NewReader(image)).Visit(r); err != nil {
		t.Fatalf("Visit(...) = unexpected error %v", err)
	}

	// Assert
	if got, want := unexpectedsOf(r.events), []unexpectedData(nil); !cmp.Equal(got, want) {
		t.Errorf("Visit(...) unexpected = %v, want %v", got, want)
	}
}

func TestFromSectorSourceUnexpectedAbort(t *testing.T) {
	t.Parallel()

	// Arrange
	sentinel := errors.New("abort")
	image := isotest.New().AddFile("/A.TXT", []byte("hi")).Build()
	source := isotest.SectorView{Image: image, Tails: map[int]isotest.Tail{5: {Bytes: []byte{1}}}}

	// Act
	err := iso.FromSectorSource(source).Visit(&abortVisitor{on: "unexpected", err: sentinel})

	// Assert
	opts := cmpopts.EquateErrors()
	if got, want := err, sentinel; !cmp.Equal(got, want, opts) {
		t.Errorf("Visit(...) = error %v, want %v", got, want)
	}
}

func TestFromSectorSourceUnexpectedReadError(t *testing.T) {
	t.Parallel()

	// Arrange
	sentinel := errors.New("sector read failure")
	image := isotest.New().AddFile("/A.TXT", []byte("hi")).Build()
	source := errAtSource{view: isotest.SectorView{Image: image}, at: 0, err: sentinel}

	// Act
	err := iso.FromSectorSource(source).Visit(iso.BaseVisitor{})

	// Assert
	opts := cmpopts.EquateErrors()
	if got, want := err, sentinel; !cmp.Equal(got, want, opts) {
		t.Errorf("Visit(...) = error %v, want %v", got, want)
	}
}

func TestFromSectorSourceFileStreamError(t *testing.T) {
	t.Parallel()

	// Arrange
	sentinel := errors.New("file read failure")
	image := isotest.New().AddFile("/A.TXT", []byte("hi")).Build()
	source := errAtSource{view: isotest.SectorView{Image: image}, at: fileSector, err: sentinel}

	// Act
	err := iso.FromSectorSource(source).Visit(&recorder{})

	// Assert
	opts := cmpopts.EquateErrors()
	if got, want := err, sentinel; !cmp.Equal(got, want, opts) {
		t.Errorf("Visit(...) = error %v, want %v", got, want)
	}
}

// listenerErrVisitor fails the file stream's listener with err.
type listenerErrVisitor struct {
	iso.BaseVisitor
	err error
}

func (l *listenerErrVisitor) VisitFile(_ *iso.File, s *iso.FileStream) error {
	return s.Stream(iso.ListenerFunc(func([]byte, *iso.Subheader) error {
		return l.err
	}))
}

func TestFileStreamListenerError(t *testing.T) {
	t.Parallel()

	// Arrange
	sentinel := errors.New("listener failure")
	image := isotest.New().AddFile("/A.TXT", []byte("hi")).Build()

	// Act
	err := iso.FromSectorSource(isotest.SectorView{Image: image}).Visit(&listenerErrVisitor{err: sentinel})

	// Assert
	opts := cmpopts.EquateErrors()
	if got, want := err, sentinel; !cmp.Equal(got, want, opts) {
		t.Errorf("Visit(...) = error %v, want %v", got, want)
	}
}
