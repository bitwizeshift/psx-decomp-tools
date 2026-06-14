package track

import (
	"errors"
	"fmt"
	"io"
)

// SectorReader reads the raw sectors of a track by zero-based index.
type SectorReader interface {
	// ReadSector returns the sector at index n. It returns [io.EOF] once n is at
	// or past the end of the track's data.
	ReadSector(n int) (RawSector, error)
}

// BinSectorReader reads fixed-size sectors from the binary data of a track.
type BinSectorReader struct {
	// Reader supplies the track's binary data.
	Reader io.ReaderAt

	// SectorSize is the size, in bytes, of each sector.
	SectorSize int
}

// ReadSector reads sector n at byte offset n*SectorSize. It returns [io.EOF]
// when n is at or past the end of the data, [ErrShortSector] when the data ends
// partway through the sector, or the underlying read error otherwise.
func (r BinSectorReader) ReadSector(n int) (RawSector, error) {
	data := make([]byte, r.SectorSize)
	read, err := r.Reader.ReadAt(data, int64(n)*int64(r.SectorSize))
	switch {
	case read == r.SectorSize:
		return RawSector{Number: int64(n), Data: data}, nil
	case read == 0 && errors.Is(err, io.EOF):
		return RawSector{}, io.EOF
	case errors.Is(err, io.EOF):
		return RawSector{}, fmt.Errorf("%w: sector %d", ErrShortSector, n)
	default:
		return RawSector{}, err
	}
}

// ReadAllSectors reads every sector from reader in order until [io.EOF],
// returning them as a slice. The terminating [io.EOF] is not reported.
func ReadAllSectors(reader SectorReader) ([]RawSector, error) {
	var sectors []RawSector
	for n := 0; ; n++ {
		sector, err := reader.ReadSector(n)
		if errors.Is(err, io.EOF) {
			return sectors, nil
		}
		if err != nil {
			return nil, err
		}
		sectors = append(sectors, sector)
	}
}

var _ SectorReader = BinSectorReader{}
