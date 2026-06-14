package track

// CD-ROM sector layout sizes and offsets, in bytes. The "full" layouts are the
// 2352-byte raw sectors that carry sync, header, and error-detection regions;
// the "bare" layouts carry only user data.
const (
	rawSectorSize = 2352
	bareMode1Size = 2048
	bareMode2Size = 2336

	syncSize   = 12
	headerSize = 4

	headerOffset    = syncSize              // 12
	subheaderOffset = syncSize + headerSize // 16

	modeByteOffset = syncSize + headerSize - 1 // 15

	mode1DataOffset = 16 // Mode 1 user data starts after sync+header.
	mode1DataEnd    = 16 + 2048
	mode2DataOffset = 24 // Mode 2 user data starts after sync+header+subheader.
	form1DataEnd    = 24 + 2048
	form2DataEnd    = 24 + 2324

	mode1EDCOffset = 2064 // Mode 1 EDC protects bytes [0:2064].
	form1EDCOffset = 2072 // XA Form 1 EDC protects bytes [16:2072].
	form2EDCOffset = 2348 // XA Form 2 EDC protects bytes [16:2348].

	edcSize = 4
)

// syncPattern is the 12-byte sync field at the start of every full sector.
var syncPattern = [syncSize]byte{
	0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x00,
}
