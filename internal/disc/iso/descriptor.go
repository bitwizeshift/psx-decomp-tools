package iso

import (
	"encoding/binary"
	"time"
)

// logicalSectorSize is the fixed 2048-byte logical sector that addresses the
// volume descriptor set, independent of the logical block size a descriptor may
// declare for file and directory extents.
const logicalSectorSize = 2048

// systemAreaSectors is the number of logical sectors reserved before the volume
// descriptor set begins.
const systemAreaSectors = 16

// standardIdentifier is the value the standard identifier field of every volume
// descriptor must hold.
const standardIdentifier = "CD001"

// VolumeDescriptorType identifies the kind of a volume descriptor.
type VolumeDescriptorType uint8

// Recognized volume descriptor types.
const (
	// VolumeDescriptorBoot is a boot record.
	VolumeDescriptorBoot VolumeDescriptorType = 0

	// VolumeDescriptorPrimary is a primary volume descriptor.
	VolumeDescriptorPrimary VolumeDescriptorType = 1

	// VolumeDescriptorSupplementary is a supplementary or enhanced volume
	// descriptor.
	VolumeDescriptorSupplementary VolumeDescriptorType = 2

	// VolumeDescriptorPartition is a volume partition descriptor.
	VolumeDescriptorPartition VolumeDescriptorType = 3

	// VolumeDescriptorTerminator is a volume descriptor set terminator.
	VolumeDescriptorTerminator VolumeDescriptorType = 255
)

// VolumeDescriptor is one descriptor of a volume descriptor set. Every type is
// recognized so the set can be walked and terminated; only the primary
// descriptor is decoded in full, through [VolumeDescriptor.Primary].
type VolumeDescriptor struct {
	// Extent locates the descriptor's logical sector within the image.
	Extent Extent

	// Type is the descriptor's type.
	Type VolumeDescriptorType

	// Version is the descriptor's version byte.
	Version uint8

	// Primary holds the decoded descriptor when Type is
	// [VolumeDescriptorPrimary], and is nil for every other type.
	Primary *PrimaryVolumeDescriptor
}

// PrimaryVolumeDescriptor holds the decoded fields of a primary volume
// descriptor, including the location of the path tables and the root directory
// record.
type PrimaryVolumeDescriptor struct {
	// SystemIdentifier names the system that can act on the volume's system area.
	SystemIdentifier string

	// VolumeIdentifier names the volume.
	VolumeIdentifier string

	// VolumeSpaceSize is the size of the volume in logical blocks.
	VolumeSpaceSize uint32

	// VolumeSetSize is the number of volumes in the volume set.
	VolumeSetSize uint16

	// VolumeSequence is the volume's ordinal within its set.
	VolumeSequence uint16

	// LogicalBlockSize is the size in bytes of a logical block.
	LogicalBlockSize uint16

	// PathTableSize is the size in bytes of each path table.
	PathTableSize uint32

	// TypeLPathTable is the logical block of the little-endian path table.
	TypeLPathTable uint32

	// OptTypeLPathTable is the logical block of the optional little-endian path
	// table, or zero when absent.
	OptTypeLPathTable uint32

	// TypeMPathTable is the logical block of the big-endian path table.
	TypeMPathTable uint32

	// OptTypeMPathTable is the logical block of the optional big-endian path
	// table, or zero when absent.
	OptTypeMPathTable uint32

	// Root is the directory record of the root directory.
	Root DirectoryRecord

	// VolumeSetIdentifier names the volume set.
	VolumeSetIdentifier string

	// PublisherIdentifier names the volume's publisher.
	PublisherIdentifier string

	// DataPreparerIdentifier names the preparer of the volume's data.
	DataPreparerIdentifier string

	// ApplicationIdentifier names the application that recorded the volume.
	ApplicationIdentifier string

	// CopyrightFileIdentifier names the file containing the copyright statement.
	CopyrightFileIdentifier string

	// AbstractFileIdentifier names the file containing the abstract statement.
	AbstractFileIdentifier string

	// BibliographicFileIdentifier names the file containing bibliographic
	// information.
	BibliographicFileIdentifier string

	// Created is the volume's creation timestamp, or the zero [time.Time] when
	// unset.
	Created time.Time

	// Modified is the volume's last-modification timestamp, or the zero
	// [time.Time] when unset.
	Modified time.Time

	// Expires is the volume's expiration timestamp, or the zero [time.Time] when
	// unset.
	Expires time.Time

	// Effective is the volume's effective timestamp, or the zero [time.Time] when
	// unset.
	Effective time.Time
}

// parseVolumeDescriptor parses the volume descriptor occupying the logical sector
// b, reporting its position with offset. It returns [ErrBadMagic] when the
// standard identifier is not "CD001", or any error from decoding a primary
// descriptor's both-byte-order fields or root directory record.
func parseVolumeDescriptor(b []byte, offset int64) (VolumeDescriptor, error) {
	if string(b[1:6]) != standardIdentifier {
		return VolumeDescriptor{}, ErrBadMagic
	}
	descriptor := VolumeDescriptor{
		Extent:  Extent{Offset: offset, Length: logicalSectorSize, Block: offset / logicalSectorSize},
		Type:    VolumeDescriptorType(b[0]),
		Version: b[6],
	}
	if descriptor.Type != VolumeDescriptorPrimary {
		return descriptor, nil
	}
	primary, err := parsePrimaryVolumeDescriptor(b, offset)
	if err != nil {
		return descriptor, err
	}
	descriptor.Primary = &primary
	return descriptor, nil
}

// parsePrimaryVolumeDescriptor decodes the primary volume descriptor in the
// logical sector b, whose own position is offset. It returns [ErrCorruptImage]
// when the logical block size is zero or the root directory record is malformed.
func parsePrimaryVolumeDescriptor(b []byte, offset int64) (PrimaryVolumeDescriptor, error) {
	blockSize := bothUint16(b[128:132])
	if blockSize == 0 {
		return PrimaryVolumeDescriptor{}, ErrCorruptImage
	}
	root, _, err := parseDirectoryRecord(b[156:190], offset+156)
	if err != nil {
		return PrimaryVolumeDescriptor{}, err
	}
	return PrimaryVolumeDescriptor{
		SystemIdentifier:            trimField(b[8:40]),
		VolumeIdentifier:            trimField(b[40:72]),
		VolumeSpaceSize:             bothUint32(b[80:88]),
		VolumeSetSize:               bothUint16(b[120:124]),
		VolumeSequence:              bothUint16(b[124:128]),
		LogicalBlockSize:            blockSize,
		PathTableSize:               bothUint32(b[132:140]),
		TypeLPathTable:              binary.LittleEndian.Uint32(b[140:144]),
		OptTypeLPathTable:           binary.LittleEndian.Uint32(b[144:148]),
		TypeMPathTable:              binary.BigEndian.Uint32(b[148:152]),
		OptTypeMPathTable:           binary.BigEndian.Uint32(b[152:156]),
		Root:                        root,
		VolumeSetIdentifier:         trimField(b[190:318]),
		PublisherIdentifier:         trimField(b[318:446]),
		DataPreparerIdentifier:      trimField(b[446:574]),
		ApplicationIdentifier:       trimField(b[574:702]),
		CopyrightFileIdentifier:     trimField(b[702:739]),
		AbstractFileIdentifier:      trimField(b[739:776]),
		BibliographicFileIdentifier: trimField(b[776:813]),
		Created:                     decodeVolumeTime(b[813:830]),
		Modified:                    decodeVolumeTime(b[830:847]),
		Expires:                     decodeVolumeTime(b[847:864]),
		Effective:                   decodeVolumeTime(b[864:881]),
	}, nil
}
