package cue_test

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/cue"
)

// gameFile is the expected parse of testdata/game.cue.
func gameFile() *cue.File {
	return &cue.File{
		Catalog:    "1234567890123",
		CDTextFile: "game.cdt",
		Comments:   []string{"GENRE Action", "DATE 1999"},
		Tracks: []cue.Track{
			{
				Number:  1,
				Mode:    cue.ModeMode2_2352,
				Type:    cue.TypeBinary,
				File:    "game (Track 1).bin",
				ISRC:    "ABCDE1234567",
				Flags:   []cue.Flag{cue.FlagDCP, cue.Flag4CH},
				Pregap:  &cue.MSF{Second: 2},
				Indices: []cue.Index{{Number: 1, Offset: cue.MSF{}}},
			},
			{
				Number:   2,
				Mode:     cue.ModeAudio,
				Type:     cue.TypeBinary,
				File:     "game (Track 2).bin",
				Comments: []string{"track two comment"},
				Indices: []cue.Index{
					{Number: 0, Offset: cue.MSF{Minute: 2, Second: 30}},
					{Number: 1, Offset: cue.MSF{Minute: 2, Second: 32}},
				},
				Postgap: &cue.MSF{Second: 1},
			},
		},
	}
}

func TestFromReader(t *testing.T) {
	t.Parallel()

	testErr := errors.New("read failure")

	testCases := []struct {
		name    string
		reader  io.Reader
		want    *cue.File
		wantErr error
	}{
		{
			name:   "Empty",
			reader: strings.NewReader(""),
			want:   &cue.File{},
		},
		{
			name: "BlankLinesAndDiscComments",
			reader: strings.NewReader(strings.Join([]string{
				"",
				"   ",
				"REM GENRE Action",
				"REM",
			}, "\n")),
			want: &cue.File{Comments: []string{"GENRE Action", ""}},
		},
		{
			name: "MinimalSingleTrack",
			reader: strings.NewReader(`
FILE "a.bin" BINARY
TRACK 01 AUDIO
INDEX 01 00:00:00`),
			want: &cue.File{
				Tracks: []cue.Track{{
					Number:  1,
					Mode:    cue.ModeAudio,
					Type:    cue.TypeBinary,
					File:    "a.bin",
					Indices: []cue.Index{{Number: 1, Offset: cue.MSF{}}},
				}},
			},
		},
		{
			name: "IgnoredCDTextCommands",
			reader: strings.NewReader(`
TITLE Disc
PERFORMER Studio
SONGWRITER Writer`),
			want: &cue.File{},
		},
		{
			name:    "UnterminatedQuote",
			reader:  strings.NewReader(`FILE "a.bin BINARY`),
			wantErr: cue.ErrSyntax,
		},
		{
			name: "UnknownCommand",
			reader: strings.NewReader(`
WIBBLE 1`),
			wantErr: cue.ErrSyntax,
		},
		{
			name: "CatalogWrongArguments",
			reader: strings.NewReader(`
CATALOG one two`),
			wantErr: cue.ErrSyntax,
		},
		{
			name: "CDTextFileWrongArguments",
			reader: strings.NewReader(`
CDTEXTFILE`),
			wantErr: cue.ErrSyntax,
		},
		{
			name: "FileWrongArguments",
			reader: strings.NewReader(`
FILE "a.bin"`),
			wantErr: cue.ErrSyntax,
		},
		{
			name: "FileInvalidType",
			reader: strings.NewReader(`
FILE "a.bin" NONSENSE`),
			wantErr: cue.ErrInvalidType,
		},
		{
			name: "TrackWrongArguments",
			reader: strings.NewReader(`
FILE "a.bin" BINARY
TRACK 01`),
			wantErr: cue.ErrSyntax,
		},
		{
			name: "TrackBeforeFile",
			reader: strings.NewReader(`
TRACK 01 AUDIO`),
			wantErr: cue.ErrSyntax,
		},
		{
			name: "TrackInvalidNumber",
			reader: strings.NewReader(`
FILE "a.bin" BINARY
TRACK xx AUDIO`),
			wantErr: cue.ErrSyntax,
		},
		{
			name: "TrackInvalidMode",
			reader: strings.NewReader(`
FILE "a.bin" BINARY
TRACK 01 NONSENSE`),
			wantErr: cue.ErrInvalidMode,
		},
		{
			name: "FlagsBeforeTrack",
			reader: strings.NewReader(`
FLAGS DCP`),
			wantErr: cue.ErrSyntax,
		},
		{
			name: "FlagsWithoutArguments",
			reader: strings.NewReader(`
FILE "a.bin" BINARY
TRACK 01 AUDIO
FLAGS`),
			wantErr: cue.ErrSyntax,
		},
		{
			name: "FlagsInvalidValue",
			reader: strings.NewReader(`
FILE "a.bin" BINARY
TRACK 01 AUDIO
FLAGS NONSENSE`),
			wantErr: cue.ErrInvalidFlag,
		},
		{
			name: "ISRCBeforeTrack",
			reader: strings.NewReader(`
ISRC ABCDE1234567`),
			wantErr: cue.ErrSyntax,
		},
		{
			name: "ISRCWrongArguments",
			reader: strings.NewReader(`
FILE "a.bin" BINARY
TRACK 01 AUDIO
ISRC one two`),
			wantErr: cue.ErrSyntax,
		},
		{
			name: "PregapBeforeTrack",
			reader: strings.NewReader(`
PREGAP 00:02:00`),
			wantErr: cue.ErrSyntax,
		},
		{
			name: "PregapWrongArguments",
			reader: strings.NewReader(`
FILE "a.bin" BINARY
TRACK 01 AUDIO
PREGAP`),
			wantErr: cue.ErrSyntax,
		},
		{
			name: "PregapInvalidTimecode",
			reader: strings.NewReader(`
FILE "a.bin" BINARY
TRACK 01 AUDIO
PREGAP 00:99:00`),
			wantErr: cue.ErrInvalidMSF,
		},
		{
			name: "PostgapBeforeTrack",
			reader: strings.NewReader(`
POSTGAP 00:01:00`),
			wantErr: cue.ErrSyntax,
		},
		{
			name: "PostgapInvalidTimecode",
			reader: strings.NewReader(`
FILE "a.bin" BINARY
TRACK 01 AUDIO
POSTGAP bad`),
			wantErr: cue.ErrInvalidMSF,
		},
		{
			name: "IndexBeforeTrack",
			reader: strings.NewReader(`
INDEX 01 00:00:00`),
			wantErr: cue.ErrSyntax,
		},
		{
			name: "IndexWrongArguments",
			reader: strings.NewReader(`
FILE "a.bin" BINARY
TRACK 01 AUDIO
INDEX 01`),
			wantErr: cue.ErrSyntax,
		},
		{
			name: "IndexInvalidNumber",
			reader: strings.NewReader(`
FILE "a.bin" BINARY
TRACK 01 AUDIO
INDEX xx 00:00:00`),
			wantErr: cue.ErrSyntax,
		},
		{
			name: "IndexInvalidTimecode",
			reader: strings.NewReader(`
FILE "a.bin" BINARY
TRACK 01 AUDIO
INDEX 01 00:00:99`),
			wantErr: cue.ErrInvalidMSF,
		},
		{
			name:    "ReaderError",
			reader:  iotest.ErrReader(testErr),
			wantErr: testErr,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			opts := []cmp.Option{cmpopts.EquateEmpty()}

			// Act
			file, err := cue.FromReader(tc.reader)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("FromReader(...) = error %v, want %v", got, want)
			}
			if got, want := file, tc.want; !cmp.Equal(got, want, opts...) {
				t.Errorf("FromReader(...) = mismatch (-want +got):\n%s", cmp.Diff(want, got, opts...))
			}
		})
	}
}

func TestFromFile(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		path    string
		want    *cue.File
		wantErr error
	}{
		{
			name: "ValidSheet",
			path: "testdata/game.cue",
			want: gameFile(),
		},
		{
			name:    "MissingFile",
			path:    "testdata/missing.cue",
			wantErr: os.ErrNotExist,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			opts := []cmp.Option{cmpopts.EquateEmpty()}

			// Act
			file, err := cue.FromFile(tc.path)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("FromFile(%q) = error %v, want %v", tc.path, got, want)
			}
			if got, want := file, tc.want; !cmp.Equal(got, want, opts...) {
				t.Errorf("FromFile(%q) = mismatch (-want +got):\n%s", tc.path, cmp.Diff(want, got, opts...))
			}
		})
	}
}
