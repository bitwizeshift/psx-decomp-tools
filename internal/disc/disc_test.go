package disc_test

import (
	"bytes"
	"io"
	"io/fs"
	"strings"
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/cue"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/disctest"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// lines joins parts with newlines to form readable multi-line CUE text.
func lines(parts ...string) string {
	return strings.Join(parts, "\n")
}

// mustCue parses cueText into a [cue.File], failing the test if it cannot.
func mustCue(t *testing.T, cueText string) *cue.File {
	t.Helper()
	file, err := cue.FromReader(strings.NewReader(cueText))
	if err != nil {
		t.Fatalf("cue.FromReader(...) = %v, want nil", err)
	}
	return file
}

// dataBytes returns n bytes whose value increases with position, giving each
// 2048-byte sector distinct content.
func dataBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i)
	}
	return b
}

const bareTrack = `FILE "track.bin" BINARY
  TRACK 01 MODE1/2048
    INDEX 01 00:00:00`

func TestDiscOpenTrackErrors(t *testing.T) {
	t.Parallel()

	ioErr := fs.ErrPermission
	files := map[string][]byte{"track.bin": dataBytes(2048)}

	testCases := []struct {
		name    string
		cue     *cue.File
		open    disc.OpenFunc
		stat    disc.StatFunc
		number  int
		wantErr error
	}{
		{
			name:    "TrackNotFound",
			cue:     mustCue(t, bareTrack),
			open:    disctest.OpenFunc(files),
			stat:    disctest.StatFunc(files),
			number:  2,
			wantErr: disc.ErrTrackNotFound,
		},
		{
			name: "UnsupportedMode",
			cue: &cue.File{Tracks: []cue.Track{{
				Number: 1,
				Mode:   cue.Mode(99),
				File:   "track.bin",
			}}},
			open:    disctest.OpenFunc(files),
			stat:    disctest.StatFunc(files),
			number:  1,
			wantErr: track.ErrUnsupportedMode,
		},
		{
			name:    "StatFailure",
			cue:     mustCue(t, bareTrack),
			open:    disctest.OpenFunc(files),
			stat:    disctest.ErrStatFunc(ioErr),
			number:  1,
			wantErr: ioErr,
		},
		{
			name:    "OpenFailure",
			cue:     mustCue(t, bareTrack),
			open:    disctest.ErrOpenFunc(ioErr),
			stat:    disctest.StatFunc(files),
			number:  1,
			wantErr: ioErr,
		},
		{
			name:    "NotRandomAccessible",
			cue:     mustCue(t, bareTrack),
			open:    disctest.SequentialOpenFunc(files),
			stat:    disctest.StatFunc(files),
			number:  1,
			wantErr: disc.ErrRandomAccess,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := disc.NewDisc(tc.cue, disc.WithOpenFunc(tc.open), disc.WithStatFunc(tc.stat))

			// Act
			opened, err := sut.OpenTrack(tc.number)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Disc.OpenTrack(%d) = error %v, want %v", tc.number, got, want)
			}
			if got, want := opened, (*disc.Track)(nil); !cmp.Equal(got, want) {
				t.Errorf("Disc.OpenTrack(%d) = %v, want %v", tc.number, got, want)
			}
		})
	}
}

func TestDiscOpenTrackSlicing(t *testing.T) {
	t.Parallel()

	merged := dataBytes(6 * 2048)
	mergedCue := lines(
		`FILE "game.bin" BINARY`,
		`  TRACK 01 MODE1/2048`,
		`    INDEX 01 00:00:00`,
		`  TRACK 02 MODE1/2048`,
		`    INDEX 01 00:00:02`,
		`  TRACK 03 MODE1/2048`,
		`    INDEX 01 00:00:04`,
	)
	mergedFiles := map[string][]byte{"game.bin": merged}

	fileA, fileB := dataBytes(2*2048), dataBytes(3*2048)
	splitCue := lines(
		`FILE "a.bin" BINARY`,
		`  TRACK 01 MODE1/2048`,
		`    INDEX 01 00:00:00`,
		`FILE "b.bin" BINARY`,
		`  TRACK 02 MODE1/2048`,
		`    INDEX 01 00:00:00`,
	)
	splitFiles := map[string][]byte{"a.bin": fileA, "b.bin": fileB}

	testCases := []struct {
		name        string
		cue         string
		files       map[string][]byte
		number      int
		wantSectors int
		wantPayload []byte
	}{
		{
			name:        "MergedFirstTrack",
			cue:         mergedCue,
			files:       mergedFiles,
			number:      1,
			wantSectors: 2,
			wantPayload: merged[0 : 2*2048],
		},
		{
			name:        "MergedMiddleTrack",
			cue:         mergedCue,
			files:       mergedFiles,
			number:      2,
			wantSectors: 2,
			wantPayload: merged[2*2048 : 4*2048],
		},
		{
			name:        "MergedLastTrack",
			cue:         mergedCue,
			files:       mergedFiles,
			number:      3,
			wantSectors: 2,
			wantPayload: merged[4*2048 : 6*2048],
		},
		{
			name:        "SplitFirstFile",
			cue:         splitCue,
			files:       splitFiles,
			number:      1,
			wantSectors: 2,
			wantPayload: fileA,
		},
		{
			name:        "SplitSecondFile",
			cue:         splitCue,
			files:       splitFiles,
			number:      2,
			wantSectors: 3,
			wantPayload: fileB,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := disctest.MustDisc(tc.cue, tc.files)

			// Act
			opened, openErr := sut.OpenTrack(tc.number)
			payload, readErr := io.ReadAll(opened)

			// Assert
			if got, want := openErr, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Disc.OpenTrack(%d) = error %v, want %v", tc.number, got, want)
			}
			if got, want := readErr, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("io.ReadAll(track) = error %v, want %v", got, want)
			}
			if got, want := opened.SectorCount(), tc.wantSectors; !cmp.Equal(got, want) {
				t.Errorf("Track.SectorCount() = %d, want %d", got, want)
			}
			if got, want := payload, tc.wantPayload; !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
				t.Errorf("io.ReadAll(track) = mismatch (-want +got):\n%s", cmp.Diff(want, got))
			}
		})
	}
}

func TestDisc_OpenTrackSharedFile_ReadsBothTracks(t *testing.T) {
	t.Parallel()

	// Arrange
	merged := dataBytes(4 * 2048)
	cueText := lines(
		`FILE "game.bin" BINARY`,
		`  TRACK 01 MODE1/2048`,
		`    INDEX 01 00:00:00`,
		`  TRACK 02 MODE1/2048`,
		`    INDEX 01 00:00:02`,
	)
	sut := disctest.MustDisc(cueText, map[string][]byte{"game.bin": merged})

	// Act
	first, firstErr := sut.OpenTrack(1)
	second, secondErr := sut.OpenTrack(2)
	firstPayload, firstReadErr := io.ReadAll(first)
	secondPayload, secondReadErr := io.ReadAll(second)

	// Assert
	if got, want := firstErr, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("Disc.OpenTrack(1) = error %v, want %v", got, want)
	}
	if got, want := secondErr, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("Disc.OpenTrack(2) = error %v, want %v", got, want)
	}
	if got, want := firstReadErr, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("io.ReadAll(first) = error %v, want %v", got, want)
	}
	if got, want := secondReadErr, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("io.ReadAll(second) = error %v, want %v", got, want)
	}
	if got, want := firstPayload, merged[0:2*2048]; !cmp.Equal(got, want) {
		t.Errorf("io.ReadAll(first) = mismatch (-want +got):\n%s", cmp.Diff(want, got))
	}
	if got, want := secondPayload, merged[2*2048:4*2048]; !cmp.Equal(got, want) {
		t.Errorf("io.ReadAll(second) = mismatch (-want +got):\n%s", cmp.Diff(want, got))
	}
}

// sectorVisit records one observation made by a [disc.SectorVisitor].
type sectorVisit struct {
	Track  int
	Sector int64
	Data   []byte
}

func TestDiscVisitSectors(t *testing.T) {
	t.Parallel()

	merged := dataBytes(4 * 2048)
	mergedCue := lines(
		`FILE "game.bin" BINARY`,
		`  TRACK 01 MODE1/2048`,
		`    INDEX 01 00:00:00`,
		`  TRACK 02 MODE1/2048`,
		`    INDEX 01 00:00:02`,
	)

	testCases := []struct {
		name        string
		cue         string
		files       map[string][]byte
		opts        []disc.Option
		wantVisited []sectorVisit
		wantErr     error
	}{
		{
			name:  "EverySectorOfEveryTrack",
			cue:   mergedCue,
			files: map[string][]byte{"game.bin": merged},
			wantVisited: []sectorVisit{
				{Track: 1, Sector: 0, Data: merged[0:2048]},
				{Track: 1, Sector: 1, Data: merged[2048:4096]},
				{Track: 2, Sector: 0, Data: merged[4096:6144]},
				{Track: 2, Sector: 1, Data: merged[6144:8192]},
			},
			wantErr: nil,
		},
		{
			name:        "OpenFailure",
			cue:         bareTrack,
			files:       map[string][]byte{"track.bin": dataBytes(2048)},
			opts:        []disc.Option{disc.WithOpenFunc(disctest.ErrOpenFunc(fs.ErrPermission))},
			wantVisited: nil,
			wantErr:     fs.ErrPermission,
		},
		{
			name: "ReadFailure",
			cue: lines(
				`FILE "track.bin" BINARY`,
				`  TRACK 01 MODE1/2352`,
				`    INDEX 01 00:00:00`,
			),
			files:       map[string][]byte{"track.bin": corruptMode1Raw()},
			wantVisited: nil,
			wantErr:     track.ErrChecksum,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := disctest.MustDisc(tc.cue, tc.files, tc.opts...)
			var visited []sectorVisit
			visitor := func(tr *cue.Track, sector *track.Sector) {
				visited = append(visited, sectorVisit{
					Track:  tr.Number,
					Sector: sector.Number,
					Data:   sector.UserData,
				})
			}
			opts := cmp.Options{cmpopts.EquateEmpty()}

			// Act
			err := sut.VisitSectors(visitor)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Disc.VisitSectors(...) = error %v, want %v", got, want)
			}
			if got, want := visited, tc.wantVisited; !cmp.Equal(got, want, opts) {
				t.Errorf("Disc.VisitSectors(...) visited = mismatch (-want +got):\n%s", cmp.Diff(want, got, opts))
			}
		})
	}
}

func TestDisc_OpenTrackWithoutIndex_StartsAtZero(t *testing.T) {
	t.Parallel()

	// Arrange
	want := dataBytes(2 * 2048)
	cueText := lines(
		`FILE "track.bin" BINARY`,
		`  TRACK 01 MODE1/2048`,
	)
	sut := disctest.MustDisc(cueText, map[string][]byte{"track.bin": want})

	// Act
	opened, openErr := sut.OpenTrack(1)
	payload, readErr := io.ReadAll(opened)

	// Assert
	if got, want := openErr, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("Disc.OpenTrack(1) = error %v, want %v", got, want)
	}
	if got, want := readErr, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("io.ReadAll(track) = error %v, want %v", got, want)
	}
	if got, want := payload, want; !cmp.Equal(got, want) {
		t.Errorf("io.ReadAll(track) = mismatch (-want +got):\n%s", cmp.Diff(want, got))
	}
}

func TestDiscClose(t *testing.T) {
	t.Parallel()

	closeErr := fs.ErrClosed
	files := map[string][]byte{"track.bin": dataBytes(2048)}

	testCases := []struct {
		name    string
		open    disc.OpenFunc
		wantErr error
	}{
		{
			name:    "ClosesOpenedFile",
			open:    disctest.OpenFunc(files),
			wantErr: nil,
		},
		{
			name:    "ReportsCloseError",
			open:    disctest.CloseErrOpenFunc(files, closeErr),
			wantErr: closeErr,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := disc.NewDisc(
				mustCue(t, bareTrack),
				disc.WithOpenFunc(tc.open),
				disc.WithStatFunc(disctest.StatFunc(files)),
			)
			openTrack(t, sut, 1)

			// Act
			err := sut.Close()

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Disc.Close() = error %v, want %v", got, want)
			}
		})
	}
}

func TestDisc_CloseWithoutOpenFiles_ReturnsNil(t *testing.T) {
	t.Parallel()

	// Arrange
	sut := disctest.MustDisc(bareTrack, map[string][]byte{"track.bin": dataBytes(2048)})

	// Act
	err := sut.Close()

	// Assert
	if got, want := err, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("Disc.Close() = error %v, want %v", got, want)
	}
}

func TestDiscCue(t *testing.T) {
	t.Parallel()

	// Arrange
	want := mustCue(t, bareTrack)
	sut := disc.NewDisc(want)

	// Act
	got := sut.Cue()

	// Assert
	if got, want := got, want; !cmp.Equal(got, want) {
		t.Errorf("Disc.Cue() = mismatch (-want +got):\n%s", cmp.Diff(want, got))
	}
}

func TestDiscTracks(t *testing.T) {
	t.Parallel()

	// Arrange
	file := mustCue(t, bareTrack)
	sut := disc.NewDisc(file)

	// Act
	got := sut.Tracks()

	// Assert
	if got, want := got, file.Tracks; !cmp.Equal(got, want) {
		t.Errorf("Disc.Tracks() = mismatch (-want +got):\n%s", cmp.Diff(want, got))
	}
}

func TestDisc_OpenTrackDefaultOpen_ReportsOSError(t *testing.T) {
	t.Parallel()

	// Arrange
	cueText := lines(
		`FILE "missing.bin" BINARY`,
		`  TRACK 01 MODE1/2048`,
		`    INDEX 01 00:00:00`,
	)
	files := map[string][]byte{"missing.bin": dataBytes(2048)}
	sut := disc.NewDisc(mustCue(t, cueText), disc.WithStatFunc(disctest.StatFunc(files)))

	// Act
	opened, err := sut.OpenTrack(1)

	// Assert
	if got, want := err, fs.ErrNotExist; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("Disc.OpenTrack(1) = error %v, want %v", got, want)
	}
	if got, want := opened, (*disc.Track)(nil); !cmp.Equal(got, want) {
		t.Errorf("Disc.OpenTrack(1) = %v, want %v", got, want)
	}
}

func TestFromCueFile(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		path    string
		wantErr error
	}{
		{
			name:    "MissingFile",
			path:    "testdata/missing.cue",
			wantErr: fs.ErrNotExist,
		},
		{
			name:    "ParsesSample",
			path:    "testdata/sample.cue",
			wantErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Act
			_, err := disc.FromCueFile(tc.path)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("FromCueFile(%q) = error %v, want %v", tc.path, got, want)
			}
		})
	}
}

func TestFromCueFile_OpensSampleTrack_ReadsPayload(t *testing.T) {
	t.Parallel()

	// Arrange
	sut, fromErr := disc.FromCueFile("testdata/sample.cue")

	// Act
	opened, openErr := sut.OpenTrack(1)
	payload, readErr := io.ReadAll(opened)
	closeErr := sut.Close()

	// Assert
	if got, want := fromErr, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("FromCueFile(...) = error %v, want %v", got, want)
	}
	if got, want := openErr, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("Disc.OpenTrack(1) = error %v, want %v", got, want)
	}
	if got, want := readErr, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("io.ReadAll(track) = error %v, want %v", got, want)
	}
	if got, want := closeErr, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("Disc.Close() = error %v, want %v", got, want)
	}
	if got, want := payload, bytes.Repeat([]byte("A"), 2048); !cmp.Equal(got, want) {
		t.Errorf("io.ReadAll(track) = mismatch (-want +got):\n%s", cmp.Diff(want, got))
	}
}

// openTrack opens the numbered track, failing the test on error.
func openTrack(t *testing.T, d *disc.Disc, number int) *disc.Track {
	t.Helper()
	opened, err := d.OpenTrack(number)
	if err != nil {
		t.Fatalf("Disc.OpenTrack(%d) = error %v, want nil", number, err)
	}
	return opened
}
