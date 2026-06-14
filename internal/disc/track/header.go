package track

import (
	"bytes"
	"fmt"
)

// fromBCD decodes a two-digit binary-coded decimal byte. It returns
// [ErrInvalidBCD] if either nibble is not a decimal digit.
func fromBCD(b byte) (int, error) {
	hi, lo := b>>4, b&0x0F
	if hi > 9 || lo > 9 {
		return 0, fmt.Errorf("%w: %#02x", ErrInvalidBCD, b)
	}
	return int(hi)*10 + int(lo), nil
}

// decodeHeader validates the sync pattern and decodes the four-byte header of a
// full sector. It returns [ErrBadSync] for a mismatched sync pattern,
// [ErrInvalidBCD] for a malformed address, or [ErrInvalidSectorMode] for an
// unknown mode byte.
func decodeHeader(raw []byte) (*SectorHeader, error) {
	if !bytes.Equal(raw[:syncSize], syncPattern[:]) {
		return nil, fmt.Errorf("%w: %#x", ErrBadSync, raw[:syncSize])
	}
	minute, err := fromBCD(raw[headerOffset])
	if err != nil {
		return nil, err
	}
	second, err := fromBCD(raw[headerOffset+1])
	if err != nil {
		return nil, err
	}
	frame, err := fromBCD(raw[headerOffset+2])
	if err != nil {
		return nil, err
	}
	mode, err := parseSectorMode(raw[modeByteOffset])
	if err != nil {
		return nil, err
	}
	return &SectorHeader{Minute: minute, Second: second, Frame: frame, Mode: mode}, nil
}

// decodeSubheader decodes the first copy of the eight-byte XA subheader of a
// full Mode 2 sector and derives its [Form] from the submode byte.
func decodeSubheader(raw []byte) *Subheader {
	submode := raw[subheaderOffset+2]
	form := FormOne
	if submode&SubModeForm2 != 0 {
		form = FormTwo
	}
	return &Subheader{
		File:    raw[subheaderOffset],
		Channel: raw[subheaderOffset+1],
		SubMode: submode,
		Coding:  raw[subheaderOffset+3],
		Form:    form,
	}
}
