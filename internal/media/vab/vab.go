package vab

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// Magic is the four-byte tag that begins a VAB bank.
const Magic = "pBAV"

// Bank geometry. The program table is a fixed 128 entries; each program owns a
// block of 16 tones; the size table is a fixed 256 entries.
const (
	headerSize      = 32
	programCount    = 128
	programAttrSize = 16
	tonesPerProgram = 16
	toneAttrSize    = 32
	vagTableSize    = 512
	vagSizeUnit     = 8 // size table entries count 8-byte units

	reserved0 = 0xeeee // header word observed before the program count
)

// Sentinel errors reported while decoding.
var (
	// ErrInvalidHeader indicates content that does not begin with a valid VAB
	// header.
	ErrInvalidHeader = errors.New("vab: invalid header")

	// ErrTruncated indicates a bank shorter than its header and size table imply.
	ErrTruncated = errors.New("vab: truncated bank")
)

// Header describes a VAB bank.
type Header struct {
	// Version is the format version word from the header.
	Version uint32

	// ID is the bank identifier.
	ID uint32

	// Size is the total bank length in bytes, header and waveforms included.
	Size int

	// MasterVolume is the bank's master volume.
	MasterVolume uint8

	// MasterPan is the bank's master pan.
	MasterPan uint8

	// Programs is the number of instrument programs.
	Programs int

	// Tones is the total number of tones across all programs.
	Tones int

	// Waveforms is the number of VAG samples in the bank.
	Waveforms int
}

// Tone maps a note range to a VAG sample within a program.
type Tone struct {
	Volume     uint8
	Pan        uint8
	CenterNote uint8
	Shift      uint8
	NoteMin    uint8
	NoteMax    uint8
	Priority   uint8
	Mode       uint8
	ADSR1      uint16
	ADSR2      uint16

	// Program is the index of the parent program.
	Program int

	// Waveform is the zero-based index of the VAG sample this tone plays, or -1
	// when the tone references no sample.
	Waveform int
}

// Program is one instrument: its mix attributes and its tones.
type Program struct {
	Volume   uint8
	Priority uint8
	Mode     uint8
	Pan      uint8
	Tones    []Tone
}

// Bank is a decoded VAB instrument bank.
type Bank struct {
	Header    Header
	Programs  []Program
	waveforms [][]byte
}

// NumWaveforms returns the number of VAG samples in the bank.
func (b *Bank) NumWaveforms() int {
	return len(b.waveforms)
}

// Waveform returns the raw SPU ADPCM bytes of sample i, for i in the range
// [0, [Bank.NumWaveforms]).
func (b *Bank) Waveform(i int) []byte {
	return b.waveforms[i]
}

// DecodeConfig reads only the bank header from r. It reports [ErrInvalidHeader]
// when the header is short or malformed.
func DecodeConfig(r io.Reader) (Header, error) {
	var buf [headerSize]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return Header{}, fmt.Errorf("vab: %w: %w", ErrInvalidHeader, err)
	}
	return parseHeader(buf[:])
}

// Decode reads a whole VAB bank from r. It reports [ErrInvalidHeader] for a
// malformed header and [ErrTruncated] when the content is shorter than the
// header and size table require.
func Decode(r io.Reader) (*Bank, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("vab: %w", err)
	}
	header, err := parseHeader(buf)
	if err != nil {
		return nil, err
	}

	toneTable := headerSize + programCount*programAttrSize
	vagTable := toneTable + header.Programs*tonesPerProgram*toneAttrSize
	dataStart := vagTable + vagTableSize
	if len(buf) < dataStart {
		return nil, fmt.Errorf("vab: %d bytes, need %d: %w", len(buf), dataStart, ErrTruncated)
	}

	waveforms, err := sliceWaveforms(buf, vagTable, dataStart, header.Waveforms)
	if err != nil {
		return nil, err
	}
	return &Bank{
		Header:    header,
		Programs:  parsePrograms(buf, header.Programs, toneTable),
		waveforms: waveforms,
	}, nil
}

// parseHeader reads and validates the 32-byte bank header from the head of buf.
func parseHeader(buf []byte) (Header, error) {
	if len(buf) < headerSize || string(buf[0:4]) != Magic {
		return Header{}, fmt.Errorf("vab: %w", ErrInvalidHeader)
	}
	if binary.LittleEndian.Uint16(buf[16:18]) != reserved0 {
		return Header{}, fmt.Errorf("vab: %w", ErrInvalidHeader)
	}
	return Header{
		Version:      binary.LittleEndian.Uint32(buf[4:8]),
		ID:           binary.LittleEndian.Uint32(buf[8:12]),
		Size:         int(binary.LittleEndian.Uint32(buf[12:16])),
		Programs:     int(binary.LittleEndian.Uint16(buf[18:20])),
		Tones:        int(binary.LittleEndian.Uint16(buf[20:22])),
		Waveforms:    int(binary.LittleEndian.Uint16(buf[22:24])),
		MasterVolume: buf[24],
		MasterPan:    buf[25],
	}, nil
}

// parsePrograms reads count program-tone blocks. Each block belongs to the next
// program slot whose attribute records a non-zero tone count; the blocks follow
// the table in that order.
func parsePrograms(buf []byte, count, toneTable int) []Program {
	programs := make([]Program, 0, count)
	for slot, block := 0, 0; slot < programCount && block < count; slot++ {
		attr := buf[headerSize+slot*programAttrSize:]
		tones := int(attr[0])
		if tones == 0 {
			continue
		}
		base := toneTable + block*tonesPerProgram*toneAttrSize
		programs = append(programs, Program{
			Volume:   attr[1],
			Priority: attr[2],
			Mode:     attr[3],
			Pan:      attr[4],
			Tones:    parseTones(buf, base, min(tones, tonesPerProgram)),
		})
		block++
	}
	return programs
}

// parseTones reads count 32-byte tone attributes starting at base.
func parseTones(buf []byte, base, count int) []Tone {
	tones := make([]Tone, count)
	for i := range tones {
		attr := buf[base+i*toneAttrSize:]
		tones[i] = Tone{
			Priority:   attr[0],
			Mode:       attr[1],
			Volume:     attr[2],
			Pan:        attr[3],
			CenterNote: attr[4],
			Shift:      attr[5],
			NoteMin:    attr[6],
			NoteMax:    attr[7],
			ADSR1:      binary.LittleEndian.Uint16(attr[16:18]),
			ADSR2:      binary.LittleEndian.Uint16(attr[18:20]),
			Program:    int(binary.LittleEndian.Uint16(attr[20:22])),
			Waveform:   int(binary.LittleEndian.Uint16(attr[22:24])) - 1,
		}
	}
	return tones
}

// sliceWaveforms splits the waveform region into count samples using the size
// table at vagTable, whose entries count 8-byte units and whose first entry is
// reserved.
func sliceWaveforms(buf []byte, vagTable, dataStart, count int) ([][]byte, error) {
	waveforms := make([][]byte, 0, count)
	offset := dataStart
	for i := 1; i <= count; i++ {
		size := int(binary.LittleEndian.Uint16(buf[vagTable+i*2:])) * vagSizeUnit
		if offset+size > len(buf) {
			return nil, fmt.Errorf("vab: waveform %d exceeds %d bytes: %w", i-1, len(buf), ErrTruncated)
		}
		waveforms = append(waveforms, buf[offset:offset+size])
		offset += size
	}
	return waveforms, nil
}
