package track

// RawSector is a complete, unaltered sector as read from a track. Data holds the
// sector's bytes exactly as stored, including any sync, header, and error-code
// regions.
type RawSector struct {
	// Number is the zero-based index of the sector within its track.
	Number int64

	// Data is the raw bytes of the sector.
	Data []byte
}

// SectorHeader is the decoded four-byte header of a CD-ROM sector: a BCD MM:SS:FF
// address followed by the sector mode.
type SectorHeader struct {
	Minute int
	Second int
	Frame  int
	Mode   SectorMode
}

// Form is the XA sector form, derived from the subheader submode bit
// [SubModeForm2].
type Form int

const (
	// FormOne is XA Form 1: 2048 bytes of user data protected by EDC and ECC.
	FormOne Form = 1
	// FormTwo is XA Form 2: 2324 bytes of user data with an optional EDC.
	FormTwo Form = 2
)

// Submode bit flags within the XA subheader submode byte.
const (
	SubModeEndOfRecord byte = 1 << 0
	SubModeVideo       byte = 1 << 1
	SubModeAudio       byte = 1 << 2
	SubModeData        byte = 1 << 3
	SubModeTrigger     byte = 1 << 4
	SubModeForm2       byte = 1 << 5
	SubModeRealTime    byte = 1 << 6
	SubModeEndOfFile   byte = 1 << 7
)

// Subheader is one decoded copy of the eight-byte XA subheader carried by a Mode
// 2 sector.
type Subheader struct {
	File    byte
	Channel byte
	SubMode byte
	Coding  byte
	Form    Form
}

// Sector is the decoded form of a [RawSector]. Header and Subheader are nil for
// layouts that lack them, such as audio, raw, and the bare 2048- and 2336-byte
// data layouts. Raw retains the full sector bytes.
type Sector struct {
	// Number is the zero-based index of the sector within its track.
	Number int64

	// Header is the decoded sector header, or nil when the layout has none.
	Header *SectorHeader

	// Subheader is the decoded XA subheader, or nil when the layout has none.
	Subheader *Subheader

	// UserData is the logical data region of the sector.
	UserData []byte

	// Raw is the full, unaltered sector bytes.
	Raw []byte
}
