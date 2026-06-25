package miditest

import "encoding/binary"

// Event is one track event: its delta time in ticks and its raw message bytes.
type Event struct {
	Delta   int
	Message []byte
}

// NoteOn returns a note-on message for the given channel.
func NoteOn(channel, note, velocity byte) []byte {
	return []byte{0x90 | channel&0x0f, note, velocity}
}

// NoteOff returns a note-off message for the given channel. The dialect's
// note-off carries only a note number.
func NoteOff(channel, note byte) []byte {
	return []byte{0x80 | channel&0x0f, note}
}

// ProgramChange returns a program-change message for the given channel.
func ProgramChange(channel, program byte) []byte {
	return []byte{0xc0 | channel&0x0f, program}
}

// ControlChange returns a control-change message for the given channel.
func ControlChange(channel, controller, value byte) []byte {
	return []byte{0xb0 | channel&0x0f, controller, value}
}

// SetTempo returns a set-tempo meta event of micros microseconds per quarter
// note.
func SetTempo(micros uint32) []byte {
	return []byte{0xff, 0x51, 0x03, byte(micros >> 16), byte(micros >> 8), byte(micros)}
}

// EndOfTrack returns the end-of-track meta event.
func EndOfTrack() []byte {
	return []byte{0xff, 0x2f, 0x00}
}

// Track frames events into an MTrk chunk.
func Track(events ...Event) []byte {
	var body []byte
	for _, event := range events {
		body = append(body, varLen(event.Delta)...)
		body = append(body, event.Message...)
	}
	out := make([]byte, 8, 8+len(body))
	copy(out[0:4], "MTrk")
	binary.BigEndian.PutUint32(out[4:8], uint32(len(body)))
	return append(out, body...)
}

// File frames tracks into a complete Standard MIDI File with the given division.
// The header format is 0 for a single track and 1 for more.
func File(division int, tracks ...[]byte) []byte {
	format := 0
	if len(tracks) > 1 {
		format = 1
	}
	out := make([]byte, 14)
	copy(out[0:4], "MThd")
	binary.BigEndian.PutUint32(out[4:8], 6)
	binary.BigEndian.PutUint16(out[8:10], uint16(format))
	binary.BigEndian.PutUint16(out[10:12], uint16(len(tracks)))
	binary.BigEndian.PutUint16(out[12:14], uint16(division))
	for _, track := range tracks {
		out = append(out, track...)
	}
	return out
}

// Sequence frames events as a single-track file with the given division.
func Sequence(division int, events ...Event) []byte {
	return File(division, Track(events...))
}

// varLen encodes value as a MIDI variable-length quantity.
func varLen(value int) []byte {
	buf := []byte{byte(value & 0x7f)}
	for value >>= 7; value > 0; value >>= 7 {
		buf = append([]byte{byte(value&0x7f) | 0x80}, buf...)
	}
	return buf
}
