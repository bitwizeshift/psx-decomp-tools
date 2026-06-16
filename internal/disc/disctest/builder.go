package disctest

import (
	"strings"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/cue"
)

// TrackFileName is the backing file name used by [OpenTrack] and [MustOpenTrack].
const TrackFileName = "track.bin"

// Disc builds an in-memory [disc.Disc] from cueText and the named file buffers
// in files, applying opts after the in-memory I/O. It returns any error from
// parsing cueText.
func Disc(cueText string, files map[string][]byte, opts ...disc.Option) (*disc.Disc, error) {
	file, err := cue.FromReader(strings.NewReader(cueText))
	if err != nil {
		return nil, err
	}
	all := append([]disc.Option{
		disc.WithOpenFunc(OpenFunc(files)),
		disc.WithStatFunc(StatFunc(files)),
	}, opts...)
	return disc.NewDisc(file, all...), nil
}

// MustDisc is like [Disc] but panics if cueText cannot be parsed.
func MustDisc(cueText string, files map[string][]byte, opts ...disc.Option) *disc.Disc {
	d, err := Disc(cueText, files, opts...)
	if err != nil {
		panic(err)
	}
	return d
}

// OpenTrack builds a one-track in-memory disc whose single track has the given
// mode and is backed by raw, then opens that track. It returns any error from
// building the disc or opening the track.
func OpenTrack(mode cue.Mode, raw []byte, opts ...disc.Option) (*disc.Track, error) {
	d := MustDisc(singleTrackCue(mode), map[string][]byte{TrackFileName: raw}, opts...)
	return d.OpenTrack(1)
}

// MustOpenTrack is like [OpenTrack] but panics on any error.
func MustOpenTrack(mode cue.Mode, raw []byte, opts ...disc.Option) *disc.Track {
	track, err := OpenTrack(mode, raw, opts...)
	if err != nil {
		panic(err)
	}
	return track
}

// singleTrackCue returns CUE text declaring one track of the given mode backed
// by [TrackFileName].
func singleTrackCue(mode cue.Mode) string {
	return strings.Join([]string{
		`FILE "` + TrackFileName + `" BINARY`,
		`  TRACK 01 ` + mode.String(),
		`    INDEX 01 00:00:00`,
	}, "\n")
}
