package track

// AudioDecoder decodes CD-DA audio sectors, whose 2352 bytes are PCM samples
// with no header or error-detection regions.
type AudioDecoder struct{}

// DecodeSector returns sector as a [Sector] whose user data is the entire PCM
// frame. It never fails.
func (AudioDecoder) DecodeSector(sector RawSector) (Sector, error) {
	return bareSector(sector), nil
}

var _ SectorDecoder = AudioDecoder{}
