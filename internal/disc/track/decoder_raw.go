package track

// RawDecoder passes sectors through without interpretation, treating the entire
// sector as user data. It is the fallback for modes with no known layout.
type RawDecoder struct{}

// DecodeSector returns sector as a [Sector] whose user data is the entire
// sector. It never fails.
func (RawDecoder) DecodeSector(sector RawSector) (Sector, error) {
	return bareSector(sector), nil
}

var _ SectorDecoder = RawDecoder{}
