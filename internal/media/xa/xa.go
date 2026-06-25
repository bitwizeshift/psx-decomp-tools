package xa

import (
	"errors"
	"fmt"
)

// Sector geometry for four-bit XA ADPCM.
const (
	adpcmBytes    = 2304 // 18 sound groups of 128 bytes
	groupSize     = 128
	groupCount    = 18
	unitsPerGroup = 8
	samplesPerUnit = 28
)

// ADPCM predictor coefficients, scaled by 64. Index by the filter field of a
// sound unit.
var (
	filterK0 = [...]int32{0, 60, 115, 98, 122}
	filterK1 = [...]int32{0, 0, -52, -55, -60}
)

// Sentinel errors reported while decoding.
var (
	// ErrShortSector indicates a sector with fewer than the 2304 ADPCM bytes a
	// Form 2 sector carries.
	ErrShortSector = errors.New("xa: short sector")

	// ErrUnsupportedCoding indicates a coding this package does not decode, such as
	// eight-bit samples.
	ErrUnsupportedCoding = errors.New("xa: unsupported coding")
)

// predictor holds the two previous samples of one channel's ADPCM stream.
type predictor struct {
	old   int32
	older int32
}

// Decoder decodes the sectors of a single XA channel in order, carrying predictor
// state across sectors.
type Decoder struct {
	coding Coding
	left   predictor
	right  predictor
}

// NewDecoder returns a [Decoder] for the given coding. It reports
// [ErrUnsupportedCoding] for a coding that is not four-bit.
func NewDecoder(coding Coding) (*Decoder, error) {
	if !coding.fourBit() {
		return nil, fmt.Errorf("xa: %#x is not four-bit: %w", byte(coding), ErrUnsupportedCoding)
	}
	return &Decoder{coding: coding}, nil
}

// DecodeSector decodes one sector's user data into interleaved PCM samples. data
// must hold at least the 2304 ADPCM bytes of a Form 2 sector; any trailing bytes
// are ignored. Stereo output interleaves left and right; mono output is a single
// channel. It reports [ErrShortSector] for an undersized sector.
func (d *Decoder) DecodeSector(data []byte) ([]int16, error) {
	if len(data) < adpcmBytes {
		return nil, fmt.Errorf("xa: %d bytes: %w", len(data), ErrShortSector)
	}
	if d.coding.Stereo() {
		return d.decodeStereo(data), nil
	}
	return d.decodeMono(data), nil
}

// decodeMono decodes every sound unit of every group into one channel, in order.
func (d *Decoder) decodeMono(data []byte) []int16 {
	out := make([]int16, 0, groupCount*unitsPerGroup*samplesPerUnit)
	for g := range groupCount {
		base := g * groupSize
		for unit := range unitsPerGroup {
			out = decodeUnit(out, data, base, unit, &d.left)
		}
	}
	return out
}

// decodeStereo decodes the even sound units as the left channel and the odd units
// as the right, interleaving them in time order.
func (d *Decoder) decodeStereo(data []byte) []int16 {
	out := make([]int16, 0, groupCount*unitsPerGroup*samplesPerUnit)
	var left, right [samplesPerUnit]int16
	for g := range groupCount {
		base := g * groupSize
		for pair := range unitsPerGroup / 2 {
			decodeUnitInto(left[:], data, base, pair*2, &d.left)
			decodeUnitInto(right[:], data, base, pair*2+1, &d.right)
			for s := range samplesPerUnit {
				out = append(out, left[s], right[s])
			}
		}
	}
	return out
}

// decodeUnit decodes one sound unit, appending its samples to out.
func decodeUnit(out []int16, data []byte, base, unit int, p *predictor) []int16 {
	var samples [samplesPerUnit]int16
	decodeUnitInto(samples[:], data, base, unit, p)
	return append(out, samples[:]...)
}

// decodeUnitInto decodes the 28 samples of one sound unit into dst, advancing the
// predictor.
func decodeUnitInto(dst []int16, data []byte, base, unit int, p *predictor) {
	param := data[base+4+unit]
	shift := uint(param & 0x0f)
	filter := int(param>>4) & 0x07
	if filter >= len(filterK0) {
		filter = 0
	}
	for s := range samplesPerUnit {
		raw := data[base+16+s*4+unit/2]
		nibble := raw & 0x0f
		if unit&1 == 1 {
			nibble = raw >> 4
		}
		sample := int32(int16(uint16(nibble)<<12)) >> shift
		sample += (p.old*filterK0[filter] + p.older*filterK1[filter] + 32) >> 6
		sample = clamp16(sample)
		p.older = p.old
		p.old = sample
		dst[s] = int16(sample)
	}
}

// clamp16 limits sample to the range of a signed 16-bit integer.
func clamp16(sample int32) int32 {
	if sample > 32767 {
		return 32767
	}
	if sample < -32768 {
		return -32768
	}
	return sample
}
