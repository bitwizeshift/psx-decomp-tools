package vabtest

import "encoding/binary"

// Layout constants mirroring the vab package's bank geometry.
const (
	headerSize      = 32
	programCount    = 128
	programAttrSize = 16
	tonesPerProgram = 16
	toneAttrSize    = 32
	vagTableSize    = 512
)

// Tone describes one tone attribute: its mix attributes, the note range it
// covers, the note it is centered on, and the zero-based waveform it plays.
type Tone struct {
	Volume     byte
	Pan        byte
	CenterNote byte
	NoteMin    byte
	NoteMax    byte
	ADSR1      uint16
	ADSR2      uint16
	Waveform   int
}

// Program describes one instrument program: its volume and its tones.
type Program struct {
	Volume byte
	Tones  []Tone
}

// Option customizes a [Bank] beyond its programs and waveforms.
type Option interface {
	apply(*options)
}

type option func(*options)

func (o option) apply(opts *options) { o(opts) }

type options struct {
	masterVolume byte
}

// WithMasterVolume sets the bank's master volume byte.
func WithMasterVolume(volume byte) Option {
	return option(func(o *options) { o.masterVolume = volume })
}

// Bank assembles a complete VAB byte stream. programs are laid out as slots in
// order; a slot with no tones is left empty, and the tone blocks of the remaining
// programs are packed after the program table. Each waveform's length must be a
// multiple of eight bytes.
func Bank(programs []Program, waveforms [][]byte, opts ...Option) []byte {
	var o options
	for _, opt := range opts {
		opt.apply(&o)
	}
	present := presentCount(programs)
	toneTable := headerSize + programCount*programAttrSize
	vagTable := toneTable + present*tonesPerProgram*toneAttrSize
	dataStart := vagTable + vagTableSize

	out := make([]byte, dataStart)
	copy(out[0:4], "pBAV")
	binary.LittleEndian.PutUint32(out[4:8], 7)
	binary.LittleEndian.PutUint16(out[16:18], 0xeeee)
	binary.LittleEndian.PutUint16(out[18:20], uint16(present))
	binary.LittleEndian.PutUint16(out[20:22], uint16(totalTones(programs)))
	binary.LittleEndian.PutUint16(out[22:24], uint16(len(waveforms)))
	out[24] = o.masterVolume

	block := 0
	for slot, program := range programs {
		out[headerSize+slot*programAttrSize] = byte(len(program.Tones))
		if len(program.Tones) == 0 {
			continue
		}
		out[headerSize+slot*programAttrSize+1] = program.Volume
		base := toneTable + block*tonesPerProgram*toneAttrSize
		for index, tone := range program.Tones {
			attr := out[base+index*toneAttrSize:]
			attr[2] = tone.Volume
			attr[3] = tone.Pan
			attr[4] = tone.CenterNote
			attr[6] = tone.NoteMin
			attr[7] = tone.NoteMax
			binary.LittleEndian.PutUint16(attr[16:18], tone.ADSR1)
			binary.LittleEndian.PutUint16(attr[18:20], tone.ADSR2)
			binary.LittleEndian.PutUint16(attr[20:22], uint16(slot))
			binary.LittleEndian.PutUint16(attr[22:24], uint16(tone.Waveform+1))
		}
		block++
	}

	for i, waveform := range waveforms {
		binary.LittleEndian.PutUint16(out[vagTable+(i+1)*2:], uint16(len(waveform)/8))
		out = append(out, waveform...)
	}

	binary.LittleEndian.PutUint32(out[12:16], uint32(len(out)))
	return out
}

// presentCount returns the number of programs that carry at least one tone.
func presentCount(programs []Program) int {
	present := 0
	for _, program := range programs {
		if len(program.Tones) > 0 {
			present++
		}
	}
	return present
}

// totalTones sums the tone counts across programs.
func totalTones(programs []Program) int {
	total := 0
	for _, program := range programs {
		total += len(program.Tones)
	}
	return total
}
