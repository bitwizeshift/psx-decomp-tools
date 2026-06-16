package iso

import "errors"

// Sentinel errors reported while parsing an ISO 9660 image.
var (
	// ErrBadMagic indicates a volume descriptor whose standard identifier is not
	// the required "CD001".
	ErrBadMagic = errors.New("iso: bad standard identifier")

	// ErrNoPrimaryDescriptor indicates the volume descriptor set contained no
	// primary volume descriptor.
	ErrNoPrimaryDescriptor = errors.New("iso: no primary volume descriptor")

	// ErrCorruptImage indicates a record whose length or bounds are inconsistent,
	// such as a both-endian field whose halves disagree or an extent that runs
	// past the end of the image.
	ErrCorruptImage = errors.New("iso: corrupt image")

	// ErrTruncated indicates a structural read reached the end of the image before
	// a complete record could be read.
	ErrTruncated = errors.New("iso: truncated image")
)
