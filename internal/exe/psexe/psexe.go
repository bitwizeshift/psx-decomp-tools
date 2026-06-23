package psexe

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// Magic is the eight-byte ASCII tag that begins a PS-X EXE header. Its fields are
// little-endian, matching the PlayStation's MIPS core.
const Magic = "PS-X EXE"

// HeaderSize is the fixed length of the header that precedes the text payload. The
// BIOS loads the bytes that follow it to the address in [Header.TextAddr].
const HeaderSize = 0x800

// regionOffset is the start of the ASCII region marker within the header.
const regionOffset = 0x4c

// Header is the decoded PS-X EXE header. All address and size fields are byte
// values in the PlayStation's address space.
type Header struct {
	// PC0 is the program counter the BIOS jumps to once the executable is loaded.
	PC0 uint32

	// GP0 is the initial global pointer (register r28).
	GP0 uint32

	// TextAddr is the address the text payload is loaded to.
	TextAddr uint32

	// TextSize is the length of the text payload in bytes. The total file size is
	// [HeaderSize] plus this value.
	TextSize uint32

	// DataAddr and DataSize describe the data region. They are zero in most
	// executables, whose data is carried within the text payload.
	DataAddr uint32
	DataSize uint32

	// BSSAddr and BSSSize describe the zero-initialized region cleared at load.
	BSSAddr uint32
	BSSSize uint32

	// StackAddr is the initial stack and frame pointer base.
	StackAddr uint32

	// StackSize is the offset added to [Header.StackAddr] to form the initial
	// stack pointer.
	StackSize uint32

	// Region is the ASCII marker string near the end of the header, with trailing
	// padding removed. The BIOS does not require it.
	Region string
}

// SectionKind identifies one of the regions described by a PS-X EXE header.
type SectionKind int

// The regions a PS-X EXE header describes. Only [Text] is backed by bytes in the
// file; the rest name address ranges established at load time.
const (
	Text SectionKind = iota
	Data
	BSS
	Stack
)

// noOffset marks a [Section] that has no bytes in the file.
const noOffset = -1

// Section describes one region of an executable: where it lives in memory, how
// large it is, and the file offset of its bytes when it has any.
type Section struct {
	// Kind is the region this section describes.
	Kind SectionKind

	// Addr is the region's base address in the PlayStation's address space.
	Addr uint32

	// Size is the region's length in bytes.
	Size uint32

	// Offset is the byte offset of the region's bytes within the file, or -1 when
	// the region has no bytes in the file.
	Offset int64
}

// File is a parsed PS-X EXE: its [Header] and the text payload that follows.
type File struct {
	Header Header
	text   []byte
}

// DecodeConfig reads and validates the PS-X EXE header from r without reading the
// text payload. It reports [ErrInvalidHeader] when the header is short or carries
// the wrong magic tag.
func DecodeConfig(r io.Reader) (Header, error) {
	var buf [HeaderSize]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return Header{}, fmt.Errorf("psexe: %w: %w", ErrInvalidHeader, err)
	}
	if string(buf[0:len(Magic)]) != Magic {
		return Header{}, fmt.Errorf("psexe: %q: %w", buf[0:len(Magic)], ErrInvalidHeader)
	}
	return Header{
		PC0:       binary.LittleEndian.Uint32(buf[0x10:0x14]),
		GP0:       binary.LittleEndian.Uint32(buf[0x14:0x18]),
		TextAddr:  binary.LittleEndian.Uint32(buf[0x18:0x1c]),
		TextSize:  binary.LittleEndian.Uint32(buf[0x1c:0x20]),
		DataAddr:  binary.LittleEndian.Uint32(buf[0x20:0x24]),
		DataSize:  binary.LittleEndian.Uint32(buf[0x24:0x28]),
		BSSAddr:   binary.LittleEndian.Uint32(buf[0x28:0x2c]),
		BSSSize:   binary.LittleEndian.Uint32(buf[0x2c:0x30]),
		StackAddr: binary.LittleEndian.Uint32(buf[0x30:0x34]),
		StackSize: binary.LittleEndian.Uint32(buf[0x34:0x38]),
		Region:    string(bytes.TrimRight(buf[regionOffset:], "\x00")),
	}, nil
}

// Decode reads a PS-X EXE header and its text payload from r. It reports
// [ErrInvalidHeader] for a malformed header and [ErrTruncated] when the payload
// ends before [Header.TextSize] bytes are read.
func Decode(r io.Reader) (*File, error) {
	header, err := DecodeConfig(r)
	if err != nil {
		return nil, err
	}
	text := make([]byte, header.TextSize)
	if _, err := io.ReadFull(r, text); err != nil {
		return nil, fmt.Errorf("psexe: %w: %w", ErrTruncated, err)
	}
	return &File{Header: header, text: text}, nil
}

// Sections returns the regions described by the executable in the fixed order
// text, data, BSS, stack. Only the text section reports a file [Section.Offset].
func (f *File) Sections() []Section {
	h := &f.Header
	return []Section{
		{Kind: Text, Addr: h.TextAddr, Size: h.TextSize, Offset: HeaderSize},
		{Kind: Data, Addr: h.DataAddr, Size: h.DataSize, Offset: noOffset},
		{Kind: BSS, Addr: h.BSSAddr, Size: h.BSSSize, Offset: noOffset},
		{Kind: Stack, Addr: h.StackAddr, Size: h.StackSize, Offset: noOffset},
	}
}

// Text returns a reader over the executable's text payload.
func (f *File) Text() *bytes.Reader {
	return bytes.NewReader(f.text)
}

// Section returns a reader over the bytes backing kind. ok is false for regions
// that have no bytes in the file, which is every region other than [Text].
func (f *File) Section(kind SectionKind) (r io.Reader, ok bool) {
	if kind != Text {
		return nil, false
	}
	return f.Text(), true
}
