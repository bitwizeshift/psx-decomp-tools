package track

import "errors"

// SectorDecoder decodes a [RawSector] into a [Sector]. On a structural or
// checksum error it returns a best-effort [Sector] holding whatever could be
// decoded alongside the error, so that wrappers such as [LenientSectorDecoder]
// can still use the partial result.
type SectorDecoder interface {
	DecodeSector(sector RawSector) (Sector, error)
}

// LenientSectorDecoder wraps a [SectorDecoder] and tolerates selected decode
// errors. When the wrapped decoder fails with an error matching one of Allowed,
// the failure is passed to Callback (when set) and the best-effort [Sector] is
// returned with a nil error.
type LenientSectorDecoder struct {
	// Decoder is the underlying decoder.
	Decoder SectorDecoder

	// Callback, when non-nil, receives each swallowed error.
	Callback func(error)

	// Allowed lists the errors to tolerate, matched with [errors.Is].
	Allowed []error
}

// DecodeSector decodes sector with the wrapped decoder, swallowing any error
// that matches Allowed. It returns the underlying error unchanged otherwise.
func (d LenientSectorDecoder) DecodeSector(sector RawSector) (Sector, error) {
	decoded, err := d.Decoder.DecodeSector(sector)
	if err != nil && d.allows(err) {
		if d.Callback != nil {
			d.Callback(err)
		}
		return decoded, nil
	}
	return decoded, err
}

// allows reports whether err matches any of the tolerated errors.
func (d LenientSectorDecoder) allows(err error) bool {
	for _, allowed := range d.Allowed {
		if errors.Is(err, allowed) {
			return true
		}
	}
	return false
}

// bareSector wraps raw as a [Sector] whose user data is the entire sector and
// which carries no header.
func bareSector(raw RawSector) Sector {
	return Sector{Number: raw.Number, UserData: raw.Data, Raw: raw.Data}
}

var _ SectorDecoder = LenientSectorDecoder{}
