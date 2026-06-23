package disc

import (
	"io"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/cue"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/iso"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track"
)

// logicalBlockSize is the fixed size, in bytes, of an ISO 9660 logical block.
const logicalBlockSize = 2048

// Volume opens every track of the disc and returns an [iso.SectorSource] that
// addresses them as one continuous logical-sector space: sector indices run from
// zero through the end of the final track, so a file whose extent lies on a later
// track, such as XA audio referenced from the data track, is reachable. It
// returns any error from opening a track.
func (d *Disc) Volume() (iso.SectorSource, error) {
	v := &volume{}
	start := 0
	for _, tr := range d.cue.Tracks {
		track, err := d.OpenTrack(tr.Number)
		if err != nil {
			return nil, err
		}
		v.spans = append(v.spans, trackSpan{track: track, start: start})
		start += track.SectorCount()
	}
	v.sectors = start
	return v, nil
}

// volume addresses the tracks of a disc as one continuous logical-sector space.
type volume struct {
	spans   []trackSpan
	sectors int
}

// Size returns the number of bytes the volume spans across all its tracks, as the
// cooked 2048-byte logical block addressing the [iso.SectorSource] reads through.
func (v *volume) Size() int64 {
	return int64(v.sectors) * logicalBlockSize
}

// trackSpan is an opened track together with the absolute sector index at which
// it begins within its volume.
type trackSpan struct {
	track *Track
	start int
}

// ReadSector returns the decoded sector at the volume's index n, drawn from the
// track that contains it and split into its ISO logical block and any tail
// according to the track's mode and the sector's form. It returns [io.EOF] once n
// is past the final track's last sector, or any error from reading or decoding
// the sector.
func (v *volume) ReadSector(n int) (iso.Sector, error) {
	for _, span := range v.spans {
		if n < span.start+span.track.SectorCount() {
			sector, err := span.track.ReadSector(n - span.start)
			if err != nil {
				return iso.Sector{}, err
			}
			return decodeSector(n, sector, span.track.Mode()), nil
		}
	}
	return iso.Sector{}, io.EOF
}

// decodeSector maps a decoded track sector at volume index n to an [iso.Sector],
// splitting its user data into the 2048-byte ISO logical block and any tail. The
// tail is a stream continuation for audio tracks and XA Form 2 sectors, and
// opaque otherwise.
func decodeSector(n int, sector track.Sector, mode cue.Mode) iso.Sector {
	out := iso.Sector{Index: int64(n)}
	if data := sector.UserData; len(data) > logicalBlockSize {
		out.Block = data[:logicalBlockSize]
		out.Tail = data[logicalBlockSize:]
	} else {
		out.Block = data
	}
	out.TailIsStream = isStreamMode(mode) || (sector.Subheader != nil && sector.Subheader.Form == track.FormTwo)
	if sub := sector.Subheader; sub != nil {
		out.Subheader = &iso.Subheader{File: sub.File, Channel: sub.Channel, SubMode: sub.SubMode, Coding: sub.Coding}
	}
	return out
}

// isStreamMode reports whether a track of the given mode carries continuous
// stream data (CD-DA audio) rather than addressable file data.
func isStreamMode(mode cue.Mode) bool {
	return mode == cue.ModeAudio || mode == cue.ModeCDG
}

var _ iso.SectorSource = (*volume)(nil)
