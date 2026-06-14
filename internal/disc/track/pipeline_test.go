package track_test

import (
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/cue"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track/tracktest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// extractAll decodes and extracts the payload of every sector in order.
func extractAll(decoder track.SectorDecoder, extractor track.PayloadExtractor, sectors []track.RawSector) ([][]byte, error) {
	var payloads [][]byte
	for _, raw := range sectors {
		sector, err := decoder.DecodeSector(raw)
		if err != nil {
			return nil, err
		}
		data, err := extractor.Payload(sector)
		if err != nil {
			return nil, err
		}
		payloads = append(payloads, data)
	}
	return payloads, nil
}

func TestPipelineExtractsPayloads(t *testing.T) {
	t.Parallel()

	// Arrange
	sub := track.Subheader{File: 1, Channel: 1}
	first := payload(2048)
	second := payload(2048)
	for i := range second {
		second[i] = 0xAA
	}
	reader := tracktest.StaticSectorReader(
		tracktest.BuildMode2Form1Sector(sub, testMSF, first),
		tracktest.BuildMode2Form1Sector(sub, testMSF, second),
	)
	tr := cue.Track{Mode: cue.ModeMode2_2352}
	decoder := track.NewDispatchFactory().DecoderFor(tr)
	extractor := track.NewDispatchExtractorFactory().ExtractorFor(tr)
	wantPayloads := [][]byte{first, second}
	opts := cmp.Options{cmpopts.EquateEmpty()}

	// Act
	sectors, readErr := track.ReadAllSectors(reader)
	payloads, err := extractAll(decoder, extractor, sectors)

	// Assert
	if got, want := readErr, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("ReadAllSectors(...) = error %v, want %v", got, want)
	}
	if got, want := err, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("extractAll(...) = error %v, want %v", got, want)
	}
	if got, want := payloads, wantPayloads; !cmp.Equal(got, want, opts) {
		t.Errorf("pipeline payloads = mismatch (-want +got):\n%s", cmp.Diff(want, got, opts))
	}
}

func TestPipelineLenientSwallowsChecksum(t *testing.T) {
	t.Parallel()

	// Arrange
	sub := track.Subheader{File: 1, Channel: 1}
	raw := corrupt(tracktest.BuildMode2Form1Sector(sub, testMSF, payload(2048)), 30)
	var recorded []error
	sut := track.LenientSectorDecoder{
		Decoder:  track.Mode2Decoder{SectorSize: 2352, Verifier: track.Mode2Verifier{}},
		Callback: func(err error) { recorded = append(recorded, err) },
		Allowed:  []error{track.ErrChecksum},
	}
	opts := cmp.Options{cmpopts.EquateErrors(), cmpopts.EquateEmpty()}
	wantUserData := raw[24:2072]

	// Act
	sector, err := sut.DecodeSector(track.RawSector{Number: 0, Data: raw})

	// Assert
	if got, want := err, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("LenientSectorDecoder.DecodeSector(...) = error %v, want %v", got, want)
	}
	if got, want := recorded, []error{track.ErrChecksum}; !cmp.Equal(got, want, opts) {
		t.Errorf("LenientSectorDecoder.DecodeSector(...) recorded = %v, want %v", got, want)
	}
	if got, want := sector.UserData, wantUserData; !cmp.Equal(got, want, opts) {
		t.Errorf("LenientSectorDecoder.DecodeSector(...) UserData = mismatch (-want +got):\n%s", cmp.Diff(want, got, opts))
	}
}
