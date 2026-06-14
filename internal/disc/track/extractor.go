package track

import "github.com/bitwizeshift/psx-decomp-tools/internal/disc/cue"

// PayloadExtractor extracts the logical payload from a decoded [Sector].
type PayloadExtractor interface {
	Payload(sector Sector) ([]byte, error)
}

// Mode1Extractor yields the 2048-byte user data of a Mode 1 sector.
type Mode1Extractor struct{}

// Payload returns the sector's user data.
func (Mode1Extractor) Payload(sector Sector) ([]byte, error) {
	return sector.UserData, nil
}

// Mode2Extractor yields the user data of a bare Mode 2 sector.
type Mode2Extractor struct{}

// Payload returns the sector's user data.
func (Mode2Extractor) Payload(sector Sector) ([]byte, error) {
	return sector.UserData, nil
}

// XAExtractor yields the user data of an XA Mode 2 sector, which the decoder has
// already sized to 2048 bytes for Form 1 or 2324 bytes for Form 2.
type XAExtractor struct{}

// Payload returns the sector's user data.
func (XAExtractor) Payload(sector Sector) ([]byte, error) {
	return sector.UserData, nil
}

// RawExtractor yields the entire raw sector, used for audio and unknown modes.
type RawExtractor struct{}

// Payload returns the sector's raw bytes.
func (RawExtractor) Payload(sector Sector) ([]byte, error) {
	return sector.Raw, nil
}

// ExtractorFactory selects a [PayloadExtractor] for a track.
type ExtractorFactory interface {
	ExtractorFor(tr cue.Track) PayloadExtractor
}

// DispatchExtractorFactory selects an [ExtractorFactory] by the track's
// [cue.Mode], using Fallback for modes absent from Factories.
type DispatchExtractorFactory struct {
	// Factories maps a track mode to the factory that builds its extractor.
	Factories map[cue.Mode]ExtractorFactory

	// Fallback builds an extractor for modes absent from Factories.
	Fallback ExtractorFactory
}

// ExtractorFor returns the extractor for tr, dispatching on its mode and using
// Fallback when the mode is unknown.
func (f DispatchExtractorFactory) ExtractorFor(tr cue.Track) PayloadExtractor {
	if factory, ok := f.Factories[tr.Mode]; ok {
		return factory.ExtractorFor(tr)
	}
	return f.Fallback.ExtractorFor(tr)
}

// NewDispatchExtractorFactory returns a [DispatchExtractorFactory] wired with an
// extractor factory for every known [cue.Mode] and a [RawExtractorFactory]
// fallback.
func NewDispatchExtractorFactory() DispatchExtractorFactory {
	return DispatchExtractorFactory{
		Factories: map[cue.Mode]ExtractorFactory{
			cue.ModeAudio:      RawExtractorFactory{},
			cue.ModeCDG:        RawExtractorFactory{},
			cue.ModeMode1_2048: Mode1ExtractorFactory{},
			cue.ModeMode1_2352: Mode1ExtractorFactory{},
			cue.ModeMode2_2336: Mode2ExtractorFactory{},
			cue.ModeMode2_2352: XAExtractorFactory{},
			cue.ModeCDI_2336:   Mode2ExtractorFactory{},
			cue.ModeCDI_2352:   XAExtractorFactory{},
		},
		Fallback: RawExtractorFactory{},
	}
}

// Mode1ExtractorFactory builds a [Mode1Extractor].
type Mode1ExtractorFactory struct{}

// ExtractorFor returns a [Mode1Extractor].
func (Mode1ExtractorFactory) ExtractorFor(cue.Track) PayloadExtractor {
	return Mode1Extractor{}
}

// Mode2ExtractorFactory builds a [Mode2Extractor].
type Mode2ExtractorFactory struct{}

// ExtractorFor returns a [Mode2Extractor].
func (Mode2ExtractorFactory) ExtractorFor(cue.Track) PayloadExtractor {
	return Mode2Extractor{}
}

// XAExtractorFactory builds an [XAExtractor].
type XAExtractorFactory struct{}

// ExtractorFor returns an [XAExtractor].
func (XAExtractorFactory) ExtractorFor(cue.Track) PayloadExtractor {
	return XAExtractor{}
}

// RawExtractorFactory builds a [RawExtractor].
type RawExtractorFactory struct{}

// ExtractorFor returns a [RawExtractor].
func (RawExtractorFactory) ExtractorFor(cue.Track) PayloadExtractor {
	return RawExtractor{}
}

var (
	_ PayloadExtractor = Mode1Extractor{}
	_ PayloadExtractor = Mode2Extractor{}
	_ PayloadExtractor = XAExtractor{}
	_ PayloadExtractor = RawExtractor{}

	_ ExtractorFactory = DispatchExtractorFactory{}
	_ ExtractorFactory = Mode1ExtractorFactory{}
	_ ExtractorFactory = Mode2ExtractorFactory{}
	_ ExtractorFactory = XAExtractorFactory{}
	_ ExtractorFactory = RawExtractorFactory{}
)
