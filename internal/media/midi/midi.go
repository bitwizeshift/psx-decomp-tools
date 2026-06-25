package midi

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// HeaderMagic is the four-byte tag that begins a file's header chunk.
const HeaderMagic = "MThd"

// trackMagic is the four-byte tag that begins each track chunk.
const trackMagic = "MTrk"

// headerDataSize is the minimum length of the MThd chunk body: the format,
// track count, and division words.
const headerDataSize = 6

// Meta event types relevant to playback.
const (
	metaSetTempo   = 0x51
	metaEndOfTrack = 0x2f
)

// Sentinel errors reported while decoding.
var (
	// ErrInvalidHeader indicates content that does not begin with a valid MThd
	// header chunk.
	ErrInvalidHeader = errors.New("midi: invalid header")

	// ErrTruncated indicates a track whose bytes end in the middle of an event.
	// [Decode] returns it alongside the events it managed to read.
	ErrTruncated = errors.New("midi: truncated track")
)

// errIncomplete marks an event the remaining track bytes cannot complete.
var errIncomplete = errors.New("midi: incomplete event")

// Header describes a Standard MIDI File.
type Header struct {
	// Format is the file's format word: 0 for a single track, 1 for several
	// simultaneous tracks, 2 for several independent tracks.
	Format int

	// NumTracks is the number of MTrk chunks declared by the header.
	NumTracks int

	// Division is the timing resolution. When its top bit is clear it counts
	// ticks per quarter note.
	Division int
}

// Sequence is a decoded Standard MIDI File: its header and its tracks.
type Sequence struct {
	Header
	Tracks []Track
}

// Track is one MTrk chunk: an ordered list of timed events.
type Track struct {
	Events []Event
}

// Event is a single message preceded by its delta time.
type Event struct {
	// Delta is the number of ticks since the previous event in the track.
	Delta int

	// Message is the decoded message.
	Message Message
}

// Message is one MIDI message. The set of concrete implementations is closed;
// messages the decoder does not model are reported as [Unknown].
type Message interface {
	isMessage()
}

// NoteOn starts a note on a channel. A note-on with zero velocity is decoded as
// a [NoteOff].
type NoteOn struct {
	Channel  uint8
	Note     uint8
	Velocity uint8
}

// NoteOff stops a note on a channel. In this dialect a note-off message carries
// no release velocity.
type NoteOff struct {
	Channel uint8
	Note    uint8
}

// ProgramChange selects the instrument program for a channel.
type ProgramChange struct {
	Channel uint8
	Program uint8
}

// ControlChange sets a controller's value on a channel.
type ControlChange struct {
	Channel    uint8
	Controller uint8
	Value      uint8
}

// SetTempo sets the playback tempo as the number of microseconds per quarter
// note.
type SetTempo struct {
	MicrosPerQuarter uint32
}

// EndOfTrack marks the final event of a track.
type EndOfTrack struct{}

// Unknown preserves a message the decoder does not model, including system
// exclusive and unhandled meta events. Status is the message's status byte and
// Data is its payload.
type Unknown struct {
	Status uint8
	Data   []byte
}

func (NoteOn) isMessage()        {}
func (NoteOff) isMessage()       {}
func (ProgramChange) isMessage() {}
func (ControlChange) isMessage() {}
func (SetTempo) isMessage()      {}
func (EndOfTrack) isMessage()    {}
func (Unknown) isMessage()       {}

// DecodeConfig reads only the header chunk from r. It reports [ErrInvalidHeader]
// when the chunk is short or carries the wrong tag.
func DecodeConfig(r io.Reader) (Header, error) {
	var tag [8]byte
	if _, err := io.ReadFull(r, tag[:]); err != nil {
		return Header{}, fmt.Errorf("midi: %w: %w", ErrInvalidHeader, err)
	}
	if string(tag[0:4]) != HeaderMagic {
		return Header{}, fmt.Errorf("midi: %q: %w", tag[0:4], ErrInvalidHeader)
	}
	length := binary.BigEndian.Uint32(tag[4:8])
	if length < headerDataSize {
		return Header{}, fmt.Errorf("midi: header length %d: %w", length, ErrInvalidHeader)
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(r, body); err != nil {
		return Header{}, fmt.Errorf("midi: %w: %w", ErrInvalidHeader, err)
	}
	return Header{
		Format:    int(binary.BigEndian.Uint16(body[0:2])),
		NumTracks: int(binary.BigEndian.Uint16(body[2:4])),
		Division:  int(binary.BigEndian.Uint16(body[4:6])),
	}, nil
}

// Decode reads a whole file from r. It reports [ErrInvalidHeader] for a
// malformed header. When a track's bytes end in the middle of an event it
// returns the sequence read so far wrapped with [ErrTruncated]; a track that
// merely omits its end-of-track event, padded with trailing zeros, is not
// truncated.
func Decode(r io.Reader) (*Sequence, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("midi: %w", err)
	}
	header, err := DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	seq := &Sequence{Header: header}
	truncated := false
	pos := 8 + 6 // skip the header chunk; only its first 6 body bytes are defined
	for pos+8 <= len(data) {
		if string(data[pos:pos+4]) != trackMagic {
			break
		}
		length := int(binary.BigEndian.Uint32(data[pos+4 : pos+8]))
		pos += 8
		// The chunk length may overrun the available data; the body bounds it.
		end := min(pos+length, len(data))
		track, short := parseTrack(data[pos:end])
		seq.Tracks = append(seq.Tracks, track)
		truncated = truncated || short
		pos = end
	}
	if truncated {
		return seq, fmt.Errorf("midi: %w", ErrTruncated)
	}
	return seq, nil
}

// parseTrack decodes the event stream of a single MTrk body, ignoring the
// trailing zero padding the chunk length may include. It reports whether the
// body ended in the middle of an event rather than at an end-of-track event or
// a complete final event.
func parseTrack(body []byte) (Track, bool) {
	var events []Event
	var status uint8
	content := contentEnd(body)
	for pos := 0; pos < content; {
		delta, n, ok := readVarint(body[pos:])
		if !ok || pos+n >= len(body) {
			return Track{Events: events}, true
		}
		pos += n

		if body[pos]&0x80 != 0 {
			status = body[pos]
			pos++
		} else if status == 0 {
			return Track{Events: events}, true
		}

		msg, n, err := parseMessage(status, body[pos:])
		if err != nil {
			return Track{Events: events}, true
		}
		pos += n
		if status >= 0xf0 {
			status = 0 // system and meta messages clear running status
		}
		events = append(events, Event{Delta: delta, Message: msg})
		if _, end := msg.(EndOfTrack); end {
			break
		}
	}
	return Track{Events: events}, false
}

// parseMessage decodes the message bodied by data under the given running
// status. It returns the message and the number of bytes consumed from data.
func parseMessage(status uint8, data []byte) (Message, int, error) {
	channel := status & 0x0f
	switch status & 0xf0 {
	case 0x80:
		return channelMessage(data, 1, func() Message {
			return NoteOff{Channel: channel, Note: data[0]}
		})
	case 0x90:
		return channelMessage(data, 2, func() Message {
			if data[1] == 0 {
				return NoteOff{Channel: channel, Note: data[0]}
			}
			return NoteOn{Channel: channel, Note: data[0], Velocity: data[1]}
		})
	case 0xb0:
		return channelMessage(data, 2, func() Message {
			return ControlChange{Channel: channel, Controller: data[0], Value: data[1]}
		})
	case 0xc0:
		return channelMessage(data, 1, func() Message {
			return ProgramChange{Channel: channel, Program: data[0]}
		})
	case 0xa0:
		return unknownChannel(status, data, 2)
	case 0xd0, 0xe0:
		return unknownChannel(status, data, 1)
	case 0xf0:
		return parseSystemMessage(status, data)
	}
	return nil, 0, fmt.Errorf("midi: status %#02x: %w", status, errIncomplete)
}

// unknownChannel decodes a size-byte channel message the package does not model
// into an [Unknown], preserving its status and data.
func unknownChannel(status uint8, data []byte, size int) (Message, int, error) {
	return channelMessage(data, size, func() Message {
		return Unknown{Status: status, Data: append([]byte(nil), data[:size]...)}
	})
}

// channelMessage validates that data holds size bytes and, when it does, builds
// the message with make.
func channelMessage(data []byte, size int, make func() Message) (Message, int, error) {
	if len(data) < size {
		return nil, 0, fmt.Errorf("midi: short channel message: %w", errIncomplete)
	}
	return make(), size, nil
}

// parseSystemMessage decodes a system-exclusive or meta message. Meta tempo and
// end-of-track are modeled; everything else is reported as [Unknown].
func parseSystemMessage(status uint8, data []byte) (Message, int, error) {
	if status == 0xff {
		if len(data) < 1 {
			return nil, 0, fmt.Errorf("midi: meta type: %w", errIncomplete)
		}
		meta := data[0]
		length, n, ok := readVarint(data[1:])
		if !ok || 1+n+length > len(data) {
			return nil, 0, fmt.Errorf("midi: meta length: %w", errIncomplete)
		}
		payload := data[1+n : 1+n+length]
		consumed := 1 + n + length
		switch meta {
		case metaSetTempo:
			if length != 3 {
				return Unknown{Status: status, Data: append([]byte(nil), payload...)}, consumed, nil
			}
			tempo := uint32(payload[0])<<16 | uint32(payload[1])<<8 | uint32(payload[2])
			return SetTempo{MicrosPerQuarter: tempo}, consumed, nil
		case metaEndOfTrack:
			return EndOfTrack{}, consumed, nil
		default:
			return Unknown{Status: status, Data: append([]byte(nil), payload...)}, consumed, nil
		}
	}

	// System exclusive: a variable-length payload.
	length, n, ok := readVarint(data)
	if !ok || n+length > len(data) {
		return nil, 0, fmt.Errorf("midi: sysex length: %w", errIncomplete)
	}
	return Unknown{Status: status, Data: append([]byte(nil), data[n:n+length]...)}, n + length, nil
}

// contentEnd returns the length of body with its trailing zero padding removed.
func contentEnd(body []byte) int {
	end := len(body)
	for end > 0 && body[end-1] == 0 {
		end--
	}
	return end
}

// readVarint reads a MIDI variable-length quantity from the front of b. It
// returns the value, the number of bytes read, and whether a complete quantity
// was present.
func readVarint(b []byte) (value int, read int, ok bool) {
	for read < len(b) {
		c := b[read]
		read++
		value = value<<7 | int(c&0x7f)
		if c&0x80 == 0 {
			return value, read, true
		}
	}
	return 0, read, false
}
