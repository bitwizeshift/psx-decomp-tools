package sqv

import (
	"encoding/binary"
	"errors"
	"math"

	"github.com/bitwizeshift/psx-decomp-tools/internal/media/midi"
	"github.com/bitwizeshift/psx-decomp-tools/internal/media/vab"
	"github.com/bitwizeshift/psx-decomp-tools/internal/media/vag"
)

// ErrNoInput indicates [Render] was given no sequence or no bank.
var ErrNoInput = errors.New("sqv: no sequence or bank")

// Synthesis defaults.
const (
	defaultSampleRate = 44100
	defaultMicros     = 500000 // 120 BPM until a tempo event
	outputChannels    = 2
	attackSeconds     = 0.004 // gain ramp in to avoid a click at note start
	releaseSeconds    = 0.10  // gain ramp out after note-off
	maxSeconds        = 600   // guard against runaway timing in malformed tracks
)

// Option configures [Render].
type Option interface {
	apply(*config)
}

type option func(*config)

func (o option) apply(c *config) { o(c) }

type config struct {
	sampleRate int
}

// WithSampleRate sets the output sample rate in Hz.
func WithSampleRate(hz int) Option {
	return option(func(c *config) { c.sampleRate = hz })
}

// Render synthesizes seq against bank into interleaved stereo little-endian
// 16-bit PCM, mixing each note as a pitch-shifted sample with a gating
// envelope. It reports [ErrNoInput] when seq or bank is nil.
func Render(seq *midi.Sequence, bank *vab.Bank, opts ...Option) ([]byte, error) {
	if seq == nil || bank == nil {
		return nil, ErrNoInput
	}
	cfg := config{sampleRate: defaultSampleRate}
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	r := newRenderer(cfg, bank)
	return r.render(r.notes(seq)), nil
}

// voice is one sounded note: when it starts and ends, the note that triggered
// it, the sample it plays, and its per-channel mix attributes.
type voice struct {
	start, end int
	note       uint8
	tone       vab.Tone
	velocity   uint8
	programVol uint8
}

// renderer holds the state shared while turning a sequence into samples.
type renderer struct {
	cfg        config
	bank       *vab.Bank
	slotTones  map[int][]vab.Tone
	slotVolume map[int]uint8
	cache      map[int][]int16
}

func newRenderer(cfg config, bank *vab.Bank) *renderer {
	slotTones := map[int][]vab.Tone{}
	slotVolume := map[int]uint8{}
	for _, program := range bank.Programs {
		for _, tone := range program.Tones {
			slotTones[tone.Program] = append(slotTones[tone.Program], tone)
			slotVolume[tone.Program] = program.Volume
		}
	}
	return &renderer{
		cfg:        cfg,
		bank:       bank,
		slotTones:  slotTones,
		slotVolume: slotVolume,
		cache:      map[int][]int16{},
	}
}

// notes walks every track and returns the sounded notes with sample-accurate
// start and end times. Each track carries its own delta-time clock.
func (r *renderer) notes(seq *midi.Sequence) []voice {
	division := seq.Division
	if division <= 0 {
		division = 1
	}
	var voices []voice
	for _, track := range seq.Tracks {
		voices = append(voices, r.trackNotes(track, division)...)
	}
	return voices
}

func (r *renderer) trackNotes(track midi.Track, division int) []voice {
	var (
		voices  []voice
		program [16]int
		active  = map[uint16]voice{}
		sample  float64
		micros  = float64(defaultMicros)
	)
	for _, event := range track.Events {
		sample += float64(event.Delta) * micros / 1e6 * float64(r.cfg.sampleRate) / float64(division)
		switch msg := event.Message.(type) {
		case midi.SetTempo:
			micros = float64(msg.MicrosPerQuarter)
		case midi.ProgramChange:
			program[msg.Channel] = int(msg.Program)
		case midi.NoteOn:
			key := uint16(msg.Channel)<<8 | uint16(msg.Note)
			if tone, ok := r.selectTone(program[msg.Channel], msg.Note); ok {
				active[key] = voice{
					start:      int(sample),
					note:       msg.Note,
					tone:       tone,
					velocity:   msg.Velocity,
					programVol: r.slotVolume[tone.Program],
				}
			}
		case midi.NoteOff:
			key := uint16(msg.Channel)<<8 | uint16(msg.Note)
			if v, ok := active[key]; ok {
				v.end = int(sample)
				voices = append(voices, v)
				delete(active, key)
			}
		}
	}
	for _, v := range active {
		v.end = int(sample)
		voices = append(voices, v)
	}
	return voices
}

// selectTone returns the tone of program slot that covers note, preferring one
// whose range includes it and falling back to the slot's first playable tone.
func (r *renderer) selectTone(slot int, note uint8) (vab.Tone, bool) {
	tones := r.slotTones[slot]
	for _, tone := range tones {
		if tone.Waveform >= 0 && note >= tone.NoteMin && note <= tone.NoteMax {
			return tone, true
		}
	}
	for _, tone := range tones {
		if tone.Waveform >= 0 {
			return tone, true
		}
	}
	return vab.Tone{}, false
}

// render mixes every voice into a stereo buffer and returns it as little-endian
// PCM bytes, bounded against runaway timing.
func (r *renderer) render(voices []voice) []byte {
	release := int(releaseSeconds * float64(r.cfg.sampleRate))
	limit := maxSeconds * r.cfg.sampleRate
	total := 0
	for _, v := range voices {
		if reach := v.end + release; reach > total {
			total = reach
		}
	}
	total = min(total, limit)
	mix := make([]int32, total*outputChannels)
	for _, v := range voices {
		r.mixVoice(mix, v, release)
	}
	return encodePCM(mix)
}

// mixVoice renders one voice into the stereo accumulator. The note plays its
// sample once, ramped in over a short attack and out over the release after
// note-off, and stops when the sample data runs out.
func (r *renderer) mixVoice(mix []int32, v voice, release int) {
	pcm, ok := r.waveform(v.tone.Waveform)
	if !ok {
		return
	}
	ratio := math.Pow(2, float64(int(v.note)-int(v.tone.CenterNote))/12)
	gain := r.voiceGain(v)
	left, right := pan(v.tone.Pan)
	attack := int(attackSeconds * float64(r.cfg.sampleRate))
	duration := v.end - v.start
	frames := len(mix) / outputChannels

	for i := 0; v.start+i < frames; i++ {
		level := gain * envelope(i, duration, attack, release)
		if level == 0 && i >= duration+release {
			break
		}
		pos := float64(i) * ratio
		if int(pos) >= len(pcm) {
			break // sample finished
		}
		value := interpolate(pcm, pos) * level
		at := (v.start + i) * outputChannels
		mix[at] += int32(value * left)
		mix[at+1] += int32(value * right)
	}
}

// envelope returns the gain at sample i of a note gated for duration samples,
// ramped in over attack samples and out over release samples.
func envelope(i, duration, attack, release int) float64 {
	switch {
	case i < attack:
		return float64(i) / float64(attack)
	case i < duration:
		return 1
	case i < duration+release:
		return 1 - float64(i-duration)/float64(release)
	default:
		return 0
	}
}

// voiceGain combines the note velocity with the tone and program volumes and the
// bank master volume into a linear gain in [0, 1].
func (r *renderer) voiceGain(v voice) float64 {
	return norm(v.velocity) * norm(v.tone.Volume) * norm(v.programVol) * norm(r.bank.Header.MasterVolume)
}

// waveform decodes and caches the PCM of bank sample i.
func (r *renderer) waveform(i int) ([]int16, bool) {
	if pcm, ok := r.cache[i]; ok {
		return pcm, len(pcm) > 0
	}
	pcm, err := vag.Decode(r.bank.Waveform(i))
	if err != nil {
		pcm = nil
	}
	r.cache[i] = pcm
	return pcm, len(pcm) > 0
}

// pan splits a 0..127 pan position into left and right gains, 64 being center.
func pan(position uint8) (left, right float64) {
	right = float64(position) / 127
	return 1 - right, right
}

// interpolate samples pcm at the fractional position pos.
func interpolate(pcm []int16, pos float64) float64 {
	i := int(pos)
	if i+1 >= len(pcm) {
		return float64(pcm[i])
	}
	frac := pos - float64(i)
	return float64(pcm[i])*(1-frac) + float64(pcm[i+1])*frac
}

// norm maps a 0..127 attribute to a 0..1 gain.
func norm(v uint8) float64 {
	return float64(v) / 127
}

// encodePCM clamps the stereo accumulator to 16 bits and serializes it as
// little-endian bytes.
func encodePCM(mix []int32) []byte {
	out := make([]byte, len(mix)*2)
	for i, sample := range mix {
		binary.LittleEndian.PutUint16(out[i*2:], uint16(clamp16(sample)))
	}
	return out
}

// clamp16 limits sample to the range of a signed 16-bit integer.
func clamp16(sample int32) int16 {
	if sample > math.MaxInt16 {
		return math.MaxInt16
	}
	if sample < math.MinInt16 {
		return math.MinInt16
	}
	return int16(sample)
}
