package cue

import (
	"encoding"
	"fmt"
	"strconv"
	"strings"
)

// framesPerSecond is the number of CD frames (sectors) in one second.
const framesPerSecond = 75

// secondsPerMinute is the number of seconds in one minute.
const secondsPerMinute = 60

// MSF is a position on a CD expressed as minutes, seconds, and frames, where a
// frame is a single sector and there are 75 frames per second. It is the form
// used by INDEX, PREGAP, and POSTGAP timecodes in a CUE sheet.
type MSF struct {
	Minute int
	Second int
	Frame  int
}

// FrameCount returns the absolute frame (sector) offset represented by m,
// computed as (Minute*60 + Second)*75 + Frame.
func (m MSF) FrameCount() int {
	return (m.Minute*secondsPerMinute+m.Second)*framesPerSecond + m.Frame
}

// String returns m as the canonical "MM:SS:FF" timecode.
func (m MSF) String() string {
	return fmt.Sprintf("%02d:%02d:%02d", m.Minute, m.Second, m.Frame)
}

// MarshalText returns m as the canonical "MM:SS:FF" timecode.
func (m MSF) MarshalText() ([]byte, error) {
	return []byte(m.String()), nil
}

// UnmarshalText parses a "MM:SS:FF" timecode in text into m. It returns
// [ErrInvalidMSF] if text is not three colon-separated, non-negative integers
// with seconds below 60 and frames below 75.
func (m *MSF) UnmarshalText(text []byte) error {
	fields := strings.Split(string(text), ":")
	if len(fields) != 3 {
		return fmt.Errorf("%w: %q", ErrInvalidMSF, text)
	}
	values := make([]int, len(fields))
	for i, field := range fields {
		value, err := strconv.Atoi(field)
		if err != nil || value < 0 {
			return fmt.Errorf("%w: %q", ErrInvalidMSF, text)
		}
		values[i] = value
	}
	if values[1] >= secondsPerMinute || values[2] >= framesPerSecond {
		return fmt.Errorf("%w: %q", ErrInvalidMSF, text)
	}
	m.Minute, m.Second, m.Frame = values[0], values[1], values[2]
	return nil
}

var (
	_ fmt.Stringer             = MSF{}
	_ encoding.TextMarshaler   = MSF{}
	_ encoding.TextUnmarshaler = (*MSF)(nil)
)
