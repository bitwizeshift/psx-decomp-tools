package track

import "github.com/bitwizeshift/psx-decomp-tools/internal/disc/cue"

// DecoderFactory selects and configures a [SectorDecoder] for a track.
type DecoderFactory interface {
	DecoderFor(tr cue.Track) SectorDecoder
}

// DispatchFactory selects a [DecoderFactory] by the track's [cue.Mode], using
// Fallback for modes absent from Factories.
type DispatchFactory struct {
	// Factories maps a track mode to the factory that builds its decoder.
	Factories map[cue.Mode]DecoderFactory

	// Fallback builds a decoder for modes absent from Factories.
	Fallback DecoderFactory
}

// DecoderFor returns the decoder for tr, dispatching on its mode and using
// Fallback when the mode is unknown.
func (f DispatchFactory) DecoderFor(tr cue.Track) SectorDecoder {
	if factory, ok := f.Factories[tr.Mode]; ok {
		return factory.DecoderFor(tr)
	}
	return f.Fallback.DecoderFor(tr)
}

// NewDispatchFactory returns a [DispatchFactory] wired with a decoder factory
// for every known [cue.Mode], each with its matching verifier, and a
// [RawFactory] fallback.
func NewDispatchFactory() DispatchFactory {
	return DispatchFactory{
		Factories: map[cue.Mode]DecoderFactory{
			cue.ModeAudio:      AudioFactory{},
			cue.ModeCDG:        AudioFactory{},
			cue.ModeMode1_2048: Mode1Factory{Verifier: Mode1Verifier{}},
			cue.ModeMode1_2352: Mode1Factory{Verifier: Mode1Verifier{}},
			cue.ModeMode2_2336: Mode2Factory{Verifier: Mode2Verifier{}},
			cue.ModeMode2_2352: Mode2Factory{Verifier: Mode2Verifier{}},
			cue.ModeCDI_2336:   Mode2Factory{Verifier: Mode2Verifier{}},
			cue.ModeCDI_2352:   Mode2Factory{Verifier: Mode2Verifier{}},
		},
		Fallback: RawFactory{},
	}
}

// Mode1Factory builds a [Mode1Decoder] sized for the track's mode.
type Mode1Factory struct {
	// Verifier is passed to each decoder it builds.
	Verifier SectorVerifier
}

// DecoderFor returns a [Mode1Decoder] sized for tr's mode.
func (f Mode1Factory) DecoderFor(tr cue.Track) SectorDecoder {
	size, _ := SectorSize(tr.Mode)
	return Mode1Decoder{SectorSize: size, Verifier: f.Verifier}
}

// Mode2Factory builds a [Mode2Decoder] sized for the track's mode.
type Mode2Factory struct {
	// Verifier is passed to each decoder it builds.
	Verifier SectorVerifier
}

// DecoderFor returns a [Mode2Decoder] sized for tr's mode.
func (f Mode2Factory) DecoderFor(tr cue.Track) SectorDecoder {
	size, _ := SectorSize(tr.Mode)
	return Mode2Decoder{SectorSize: size, Verifier: f.Verifier}
}

// AudioFactory builds an [AudioDecoder].
type AudioFactory struct{}

// DecoderFor returns an [AudioDecoder].
func (AudioFactory) DecoderFor(cue.Track) SectorDecoder {
	return AudioDecoder{}
}

// RawFactory builds a [RawDecoder].
type RawFactory struct{}

// DecoderFor returns a [RawDecoder].
func (RawFactory) DecoderFor(cue.Track) SectorDecoder {
	return RawDecoder{}
}

var (
	_ DecoderFactory = DispatchFactory{}
	_ DecoderFactory = Mode1Factory{}
	_ DecoderFactory = Mode2Factory{}
	_ DecoderFactory = AudioFactory{}
	_ DecoderFactory = RawFactory{}
)
