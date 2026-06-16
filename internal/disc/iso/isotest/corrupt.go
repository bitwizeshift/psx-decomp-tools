package isotest

import "encoding/binary"

// rootDirOffset is the byte offset of the root directory extent, which every
// image places immediately after the path tables.
const rootDirOffset = firstDataSector * BlockSize

// farBlock is a logical block address far beyond the end of any image isotest
// builds, used to make an extent point past the image.
const farBlock = 0x00100000

// BuildZeroBlockSize renders an image whose primary volume descriptor declares a
// logical block size of zero.
func (b *Builder) BuildZeroBlockSize() []byte {
	image := b.Build()
	field := pvdSector*BlockSize + 128
	for i := range 4 {
		image[field+i] = 0
	}
	return image
}

// BuildShortDirectoryRecord renders an image whose first root directory record
// declares a length too small to be a record.
func (b *Builder) BuildShortDirectoryRecord() []byte {
	image := b.Build()
	image[rootDirOffset] = 5
	return image
}

// BuildOverlongIdentifier renders an image whose first root directory record
// declares an identifier longer than the record itself.
func (b *Builder) BuildOverlongIdentifier() []byte {
	image := b.Build()
	image[rootDirOffset+32] = 200
	return image
}

// BuildTruncatedPathTableRecord renders an image whose first path table record
// declares an identifier that overruns the path table.
func (b *Builder) BuildTruncatedPathTableRecord() []byte {
	image := b.Build()
	image[typeLSector*BlockSize] = 200
	return image
}

// BuildZeroPathTableRecord renders an image whose first path table record
// declares a zero-length identifier.
func (b *Builder) BuildZeroPathTableRecord() []byte {
	image := b.Build()
	image[typeLSector*BlockSize] = 0
	return image
}

// BuildBadRootRecord renders an image whose primary volume descriptor embeds a
// malformed root directory record.
func (b *Builder) BuildBadRootRecord() []byte {
	image := b.Build()
	image[pvdSector*BlockSize+156] = 5
	return image
}

// BuildPathTableBeyondEnd renders an image whose primary volume descriptor points
// its first path table past the end of the image.
func (b *Builder) BuildPathTableBeyondEnd() []byte {
	image := b.Build()
	binary.LittleEndian.PutUint32(image[pvdSector*BlockSize+140:], farBlock)
	return image
}

// BuildSubdirectoryBeyondEnd renders an image whose first root subdirectory
// record points its extent past the end of the image. It panics unless the root
// directory declares a subdirectory as its third record.
func (b *Builder) BuildSubdirectoryBeyondEnd() []byte {
	image := b.Build()
	record := rootDirOffset + recordLen(1) + recordLen(1)
	binary.LittleEndian.PutUint32(image[record+2:], farBlock)
	return image
}

// ErrorReaderAt is an [io.ReaderAt] whose every read fails with Err.
type ErrorReaderAt struct {
	// Err is the error every read reports.
	Err error
}

// ReadAt reports Err without reading.
func (e ErrorReaderAt) ReadAt([]byte, int64) (int, error) {
	return 0, e.Err
}

// TailErrorReaderAt reads from Data like a [bytes.Reader] but reports Err in
// place of [io.EOF] once a read reaches the end of Data, modelling a source that
// fails at its tail.
type TailErrorReaderAt struct {
	// Data is the readable content before the failing tail.
	Data []byte

	// Err is the error reported at and beyond the end of Data.
	Err error
}

// ReadAt copies bytes of Data at off into p, reporting Err once the read reaches
// the end of Data.
func (t TailErrorReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(len(t.Data)) {
		return 0, t.Err
	}
	n := copy(p, t.Data[off:])
	if n < len(p) {
		return n, t.Err
	}
	return n, nil
}
