package disc

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/cue"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track"
)

// ErrTrackNotFound is returned when a track number is requested that no track
// in the disc declares.
type SectorVisitor func(track *cue.Track, sector *track.Sector)

// OpenFunc opens the backing file named path for reading. The returned
// [fs.File] must also implement [io.ReaderAt]; one that does not yields
// [ErrRandomAccess] when a track on it is opened.
type OpenFunc = func(path string) (fs.File, error)

// StatFunc reports the [fs.FileInfo] for the backing file named path, whose
// size bounds the final track in that file.
type StatFunc = func(path string) (fs.FileInfo, error)

// Disc is a CD image described by a CUE sheet. It opens the tracks the sheet
// declares, reading their sectors from the backing files on demand, and owns
// every file it opens until [Disc.Close] is called.
type Disc struct {
	cue            *cue.File
	open           OpenFunc
	stat           StatFunc
	sectorCallback func(error)
	decoders       track.DecoderFactory
	extractors     track.ExtractorFactory
	files          map[string]*openFile
}

// openFile is a backing file the disc has opened, retained so it can be reused
// by other tracks and closed with the disc.
type openFile struct {
	ra     io.ReaderAt
	size   int64
	closer io.Closer
}

// toOpenFunc adapts an open function returning a concrete [fs.File] type to an
// [OpenFunc].
func toOpenFunc[T fs.File](fn func(path string) (T, error)) OpenFunc {
	return func(path string) (fs.File, error) {
		return fn(path)
	}
}

// NewDisc returns a [Disc] backed by cue, configured by opts. By default it
// reads backing files from the local filesystem; [WithOpenFunc] and
// [WithStatFunc] override that, and [WithSectorCallback] tolerates checksum
// errors.
func NewDisc(cue *cue.File, opts ...Option) *Disc {
	d := Disc{
		cue:        cue,
		open:       toOpenFunc(os.Open),
		stat:       os.Stat,
		decoders:   track.NewDispatchFactory(),
		extractors: track.NewDispatchExtractorFactory(),
		files:      map[string]*openFile{},
	}
	for _, opt := range opts {
		opt.apply(&d)
	}
	return &d
}

// FromCueFile reads and parses the CUE sheet at path and returns a [Disc] whose
// backing files are resolved relative to the sheet's directory. It returns any
// error from parsing the sheet.
func FromCueFile(path string) (*Disc, error) {
	file, err := cue.FromFile(path)
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(path)
	open := func(name string) (fs.File, error) {
		return os.Open(filepath.Join(dir, name))
	}
	stat := func(name string) (fs.FileInfo, error) {
		return os.Stat(filepath.Join(dir, name))
	}
	return NewDisc(file, WithOpenFunc(open), WithStatFunc(stat)), nil
}

// Cue returns the CUE sheet describing the disc.
func (d *Disc) Cue() *cue.File {
	return d.cue
}

// Tracks returns the disc's tracks in CUE declaration order.
func (d *Disc) Tracks() []cue.Track {
	return d.cue.Tracks
}

// OpenTrack opens the track with the given number for reading. It returns
// [ErrTrackNotFound] when no such track exists, [track.ErrUnsupportedMode] when
// the track's mode has no known sector layout, [ErrRandomAccess] when its
// backing file is not randomly accessible, or any error from opening or stating
// that file.
func (d *Disc) OpenTrack(number int) (*Track, error) {
	tr, ok := d.trackByNumber(number)
	if !ok {
		return nil, ErrTrackNotFound
	}
	size, err := track.SectorSize(tr.Mode)
	if err != nil {
		return nil, err
	}
	file, err := d.fileFor(tr.File)
	if err != nil {
		return nil, err
	}
	base, count := d.bounds(tr, int64(size), file.size)
	section := io.NewSectionReader(file.ra, base*int64(size), count*int64(size))
	decoder := d.decoders.DecoderFor(tr)
	if d.sectorCallback != nil {
		decoder = track.LenientSectorDecoder{
			Decoder:  decoder,
			Callback: d.sectorCallback,
			Allowed:  []error{track.ErrChecksum},
		}
	}
	return &Track{
		cue:        tr,
		sectorSize: size,
		count:      int(count),
		sectors:    track.BinSectorReader{Reader: section, SectorSize: size},
		decoder:    decoder,
		extractor:  d.extractors.ExtractorFor(tr),
		index:      []int64{0},
	}, nil
}

// Close closes every backing file the disc has opened, joining any errors. It
// is safe to call when no files have been opened.
func (d *Disc) Close() error {
	var errs []error
	for name, file := range d.files {
		errs = append(errs, file.closer.Close())
		delete(d.files, name)
	}
	return errors.Join(errs...)
}

var _ io.Closer = (*Disc)(nil)

// VisitSectors calls visitor for every sector of every track in the disc, in
// CUE declaration order. It returns any error from reading the sectors.
func (d *Disc) VisitSectors(visitor SectorVisitor) error {
	for _, tr := range d.cue.Tracks {
		track, err := d.OpenTrack(tr.Number)
		if err != nil {
			return err
		}
		for n := range track.SectorCount() {
			sector, err := track.ReadSector(n)
			if err != nil {
				return err
			}
			visitor(&tr, &sector)
		}
	}
	return nil
}

// trackByNumber returns the track declared with the given number.
func (d *Disc) trackByNumber(number int) (cue.Track, bool) {
	for _, tr := range d.cue.Tracks {
		if tr.Number == number {
			return tr, true
		}
	}
	return cue.Track{}, false
}

// fileFor returns the opened backing file named name, opening and caching it on
// first use. It returns [ErrRandomAccess] if the opened file is not an
// [io.ReaderAt], or any error from opening or stating it.
func (d *Disc) fileFor(name string) (*openFile, error) {
	if file, ok := d.files[name]; ok {
		return file, nil
	}
	info, err := d.stat(name)
	if err != nil {
		return nil, err
	}
	f, err := d.open(name)
	if err != nil {
		return nil, err
	}
	ra, ok := f.(io.ReaderAt)
	if !ok {
		return nil, errors.Join(ErrRandomAccess, f.Close())
	}
	file := &openFile{ra: ra, size: info.Size(), closer: f}
	d.files[name] = file
	return file, nil
}

// bounds returns the base sector index and sector count of tr within its
// backing file. A track spans from its own start frame up to the next
// same-file track's start frame, or to the end of a file of fileSize bytes for
// the last track.
func (d *Disc) bounds(tr cue.Track, sectorSize, fileSize int64) (base, count int64) {
	base = int64(startFrame(tr))
	end := fileSize / sectorSize
	for _, other := range d.cue.Tracks {
		next := int64(startFrame(other))
		if other.File == tr.File && next > base && next < end {
			end = next
		}
	}
	return base, end - base
}

// startFrame returns the absolute sector offset at which tr's data begins
// within its backing file, taken from its first INDEX, or zero when it declares
// none.
func startFrame(tr cue.Track) int {
	if len(tr.Indices) == 0 {
		return 0
	}
	return tr.Indices[0].Offset.FrameCount()
}
