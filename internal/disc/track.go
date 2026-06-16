package disc

import (
	"errors"
	"io"
	"sort"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/cue"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track"
)

// Track is a single track of a [Disc]. Its [io.Reader], [io.ReaderAt],
// [io.Seeker], and [io.WriterTo] operate over the track's logical payload: the
// user data of its sectors, concatenated in order. Decoded sectors, which also
// retain their raw bytes, are available through [Track.ReadSector].
//
// Sectors are decoded lazily, the first time a payload read reaches them, and
// each decoded payload is retained so it is decoded at most once.
type Track struct {
	cue        cue.Track
	sectorSize int
	count      int
	sectors    track.SectorReader
	decoder    track.SectorDecoder
	extractor  track.PayloadExtractor

	pos      int64
	index    []int64
	payloads [][]byte
	done     bool
}

// Number returns the track's number from its TRACK declaration.
func (t *Track) Number() int {
	return t.cue.Number
}

// Mode returns the track's sector mode from its TRACK declaration.
func (t *Track) Mode() cue.Mode {
	return t.cue.Mode
}

// SectorSize returns the on-disc size, in bytes, of one of the track's sectors.
func (t *Track) SectorSize() int {
	return t.sectorSize
}

// SectorCount returns the number of sectors the track spans.
func (t *Track) SectorCount() int {
	return t.count
}

// CueTrack returns the track's CUE declaration.
func (t *Track) CueTrack() cue.Track {
	return t.cue
}

// ReadSector returns the decoded sector at track-relative index n. It returns
// [io.EOF] once n is at or past the track's end, or a decode error from the
// track's mode alongside a best-effort sector.
func (t *Track) ReadSector(n int) (track.Sector, error) {
	raw, err := t.sectors.ReadSector(n)
	if err != nil {
		return track.Sector{}, err
	}
	return t.decoder.DecodeSector(raw)
}

// Read reads the track's payload into p, advancing the read position. It
// returns [io.EOF] once the position is at the end of the payload.
func (t *Track) Read(p []byte) (int, error) {
	read, err := t.ReadAt(p, t.pos)
	t.pos += int64(read)
	return read, err
}

// ReadAt reads len(p) bytes of the track's payload starting at byte offset off.
// It returns [io.EOF] when it reaches the end before filling p,
// [ErrNegativeOffset] when off is negative, or a decode error from the track's
// mode.
func (t *Track) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 {
		return 0, ErrNegativeOffset
	}
	read := 0
	for read < len(p) {
		payload, base, ok, err := t.payloadAt(off + int64(read))
		if err != nil {
			return read, err
		}
		if !ok {
			return read, io.EOF
		}
		read += copy(p[read:], payload[off+int64(read)-base:])
	}
	return read, nil
}

// Seek moves the payload read position and returns the new absolute offset. It
// returns [ErrWhence] for an unrecognized whence, [ErrNegativeOffset] when the
// resulting position is negative, or a decode error when sizing the payload for
// [io.SeekEnd].
func (t *Track) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = t.pos + offset
	case io.SeekEnd:
		size, err := t.payloadSize()
		if err != nil {
			return 0, err
		}
		abs = size + offset
	default:
		return 0, ErrWhence
	}
	if abs < 0 {
		return 0, ErrNegativeOffset
	}
	t.pos = abs
	return abs, nil
}

// WriteTo writes the track's remaining payload, from the current read position
// to the end, to w. It returns the number of bytes written and any decode or
// write error.
func (t *Track) WriteTo(w io.Writer) (int64, error) {
	var written int64
	for {
		payload, base, ok, err := t.payloadAt(t.pos)
		if err != nil {
			return written, err
		}
		if !ok {
			return written, nil
		}
		n, err := w.Write(payload[t.pos-base:])
		written += int64(n)
		t.pos += int64(n)
		if err != nil {
			return written, err
		}
	}
}

// payloadAt returns the payload of the sector whose region contains byte offset
// off, together with the payload offset at which that sector begins. ok is false
// when off is at or past the end of the track's payload.
func (t *Track) payloadAt(off int64) (payload []byte, base int64, ok bool, err error) {
	for !t.done && t.index[len(t.index)-1] <= off {
		if err := t.grow(); err != nil {
			return nil, 0, false, err
		}
	}
	if off >= t.index[len(t.index)-1] {
		return nil, 0, false, nil
	}
	sector := sort.Search(len(t.index), func(i int) bool {
		return t.index[i] > off
	}) - 1
	return t.payloads[sector], t.index[sector], true, nil
}

// payloadSize returns the total size of the track's payload, decoding every
// sector.
func (t *Track) payloadSize() (int64, error) {
	for !t.done {
		if err := t.grow(); err != nil {
			return 0, err
		}
	}
	return t.index[len(t.index)-1], nil
}

// grow decodes the next sector, appending its payload and cumulative offset, or
// marks the track done once its sectors are exhausted. It returns a read or
// decode error from the track's mode.
func (t *Track) grow() error {
	next := len(t.payloads)
	raw, err := t.sectors.ReadSector(next)
	if err != nil {
		if errors.Is(err, io.EOF) {
			t.done = true
			return nil
		}
		return err
	}
	sector, err := t.decoder.DecodeSector(raw)
	if err != nil {
		return err
	}
	payload, err := t.extractor.Payload(sector)
	t.payloads = append(t.payloads, payload)
	t.index = append(t.index, t.index[next]+int64(len(payload)))
	return err
}

var (
	_ io.Reader   = (*Track)(nil)
	_ io.ReaderAt = (*Track)(nil)
	_ io.Seeker   = (*Track)(nil)
	_ io.WriterTo = (*Track)(nil)
)
