package disc

import (
	"io"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/iso"
)

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
	return v, nil
}

// volume addresses the tracks of a disc as one continuous logical-sector space.
type volume struct {
	spans []trackSpan
}

// trackSpan is an opened track together with the absolute sector index at which
// it begins within its volume.
type trackSpan struct {
	track *Track
	start int
}

// ReadSector returns the user data of the volume's sector n, drawn from the track
// that contains it. It returns [io.EOF] once n is past the final track's last
// sector, or any error from reading or decoding the sector.
func (v *volume) ReadSector(n int) ([]byte, error) {
	for _, span := range v.spans {
		if n < span.start+span.track.SectorCount() {
			sector, err := span.track.ReadSector(n - span.start)
			if err != nil {
				return nil, err
			}
			return sector.UserData, nil
		}
	}
	return nil, io.EOF
}

var _ iso.SectorSource = (*volume)(nil)
