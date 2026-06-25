package sqv

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/bitwizeshift/psx-decomp-tools/internal/media/midi"
	"github.com/bitwizeshift/psx-decomp-tools/internal/media/vab"
)

// Magic is the four-byte tag that begins an SQV container.
const Magic = ".sqv"

// headerSize is the length of the fixed container header that precedes the
// sequence region.
const headerSize = 16

// Sentinel errors reported while decoding.
var (
	// ErrInvalidHeader indicates content that does not begin with the SQV tag.
	ErrInvalidHeader = errors.New("sqv: invalid header")

	// ErrMissingBank indicates a container with no embedded VAB bank.
	ErrMissingBank = errors.New("sqv: missing bank")
)

// Header describes an SQV container.
type Header struct {
	// NextSequence is the header word at offset 4. It locates the second
	// sequence as that sequence's offset less four, and is zero for a container
	// holding a single sequence. The decoder does not rely on it; sequences are
	// found by their tags.
	NextSequence int
}

// File is a decoded SQV container: its header, its sequences in order, and the
// instrument bank they play.
type File struct {
	Header    Header
	Sequences []*midi.Sequence
	Bank      *vab.Bank
}

// DecodeConfig reads the container header from r without decoding its contents.
// It reports [ErrInvalidHeader] when the content does not begin with [Magic].
func DecodeConfig(r io.Reader) (Header, error) {
	var buf [headerSize]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return Header{}, fmt.Errorf("sqv: %w: %w", ErrInvalidHeader, err)
	}
	if string(buf[0:4]) != Magic {
		return Header{}, fmt.Errorf("sqv: %q: %w", buf[0:4], ErrInvalidHeader)
	}
	return Header{NextSequence: int(binary.LittleEndian.Uint32(buf[4:8]))}, nil
}

// Decode reads a whole SQV container from r. It reports [ErrInvalidHeader] for a
// malformed tag and [ErrMissingBank] when no VAB bank is present. A sequence
// whose bytes are truncated is still kept, with its recovered events.
func Decode(r io.Reader) (*File, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("sqv: %w", err)
	}
	header, err := DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	bankOffset := bytes.Index(data, []byte(vab.Magic))
	if bankOffset < 0 {
		return nil, fmt.Errorf("sqv: %w", ErrMissingBank)
	}
	bank, err := vab.Decode(bytes.NewReader(data[bankOffset:]))
	if err != nil {
		return nil, fmt.Errorf("sqv: %w", err)
	}

	return &File{
		Header:    header,
		Sequences: decodeSequences(data[:bankOffset]),
		Bank:      bank,
	}, nil
}

// decodeSequences decodes each MThd-tagged sequence in the region preceding the
// bank. A sequence spans from its tag to the next one. Truncated sequences are
// kept with whatever events were recovered.
func decodeSequences(region []byte) []*midi.Sequence {
	starts := tagOffsets(region, []byte(midi.HeaderMagic))
	sequences := make([]*midi.Sequence, 0, len(starts))
	for i, start := range starts {
		end := len(region)
		if i+1 < len(starts) {
			end = starts[i+1]
		}
		sequence, err := midi.Decode(bytes.NewReader(region[start:end]))
		if err != nil && !errors.Is(err, midi.ErrTruncated) {
			continue
		}
		sequences = append(sequences, sequence)
	}
	return sequences
}

// tagOffsets returns every offset in data at which tag occurs.
func tagOffsets(data, tag []byte) []int {
	var offsets []int
	for off := 0; ; {
		i := bytes.Index(data[off:], tag)
		if i < 0 {
			return offsets
		}
		offsets = append(offsets, off+i)
		off += i + len(tag)
	}
}
