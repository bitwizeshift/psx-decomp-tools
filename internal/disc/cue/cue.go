package cue

import (
	"io"
	"os"

	"github.com/bitwizeshift/psx-decomp-tools/internal/ioerr"
)

// File is the parsed representation of a CUE sheet: the disc-level metadata
// followed by the ordered tracks it declares.
type File struct {
	// Catalog is the disc Media Catalog Number from a CATALOG command, or empty
	// when absent.
	Catalog string

	// CDTextFile is the external CD-TEXT filename from a CDTEXTFILE command, or
	// empty when absent.
	CDTextFile string

	// Comments holds the text of disc-level REM commands in the order they
	// appear.
	Comments []string

	// Tracks holds the tracks in declaration order.
	Tracks []Track
}

// Track is a single track within a CUE sheet, combining the governing FILE that
// supplies its data with the TRACK declaration and its sub-commands.
type Track struct {
	// Number is the track number from the TRACK command.
	Number int

	// Mode is the sector layout from the TRACK command.
	Mode Mode

	// Type is the layout of the FILE that supplies this track's data.
	Type Type

	// File is the name of the FILE that supplies this track's data.
	File string

	// ISRC is the track's International Standard Recording Code from an ISRC
	// command, or empty when absent.
	ISRC string

	// Flags holds the subcode flags from a FLAGS command in declaration order.
	Flags []Flag

	// Pregap is the silent pre-gap from a PREGAP command, or nil when absent.
	Pregap *MSF

	// Postgap is the silent post-gap from a POSTGAP command, or nil when absent.
	Postgap *MSF

	// Comments holds the text of REM commands declared within this track in the
	// order they appear.
	Comments []string

	// Indices holds the track's indices in declaration order.
	Indices []Index
}

// Index is a single INDEX point within a track.
type Index struct {
	// Number is the index number from the INDEX command.
	Number int

	// Offset is the position of the index, relative to the start of the track's
	// FILE.
	Offset MSF
}

// FromFile reads and parses the CUE sheet at path. It returns the parsed [File],
// or an error from opening the file or one of the parsing sentinels documented
// on [FromReader].
func FromFile(path string) (file *File, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer ioerr.CloseAndReport(f, &file, &err)
	return FromReader(f)
}

// FromReader reads and parses a CUE sheet from r. It returns the parsed [File],
// or one of [ErrSyntax], [ErrInvalidType], [ErrInvalidMode], [ErrInvalidFlag],
// or [ErrInvalidMSF] when the sheet is malformed.
func FromReader(r io.Reader) (*File, error) {
	return newParser(r).parse()
}
