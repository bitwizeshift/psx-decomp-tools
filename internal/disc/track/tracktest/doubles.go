package tracktest

import (
	"io"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track"
)

// ErrSectorReader returns a [track.SectorReader] whose ReadSector always fails
// with err.
func ErrSectorReader(err error) track.SectorReader {
	return sectorReaderFunc(func(int) (track.RawSector, error) {
		return track.RawSector{}, err
	})
}

// StaticSectorReader returns a [track.SectorReader] that serves the given
// sectors by index, reporting [io.EOF] once the index is past the last sector.
func StaticSectorReader(sectors ...[]byte) track.SectorReader {
	return sectorReaderFunc(func(n int) (track.RawSector, error) {
		if n < 0 || n >= len(sectors) {
			return track.RawSector{}, io.EOF
		}
		return track.RawSector{Number: int64(n), Data: sectors[n]}, nil
	})
}

type sectorReaderFunc func(n int) (track.RawSector, error)

func (f sectorReaderFunc) ReadSector(n int) (track.RawSector, error) {
	return f(n)
}

// ErrVerifier returns a [track.SectorVerifier] that always reports err. A nil
// err yields a verifier that accepts every sector.
func ErrVerifier(err error) track.SectorVerifier {
	return verifierFunc(func(track.Sector) error {
		return err
	})
}

type verifierFunc func(track.Sector) error

func (f verifierFunc) Verify(sector track.Sector) error {
	return f(sector)
}
