package iso

import (
	"encoding/binary"
	"errors"
	"io"
	"sort"
)

// segment is a claimed byte range together with the action that reports it to a
// [Visitor]. The visit closure captures the parsed value the callback needs.
type segment struct {
	extent Extent
	visit  func(i *ISO, v Visitor) error
}

// builder discovers the claimed segments of an image by parsing its volume
// descriptors, path tables, and directory tree through random access.
type builder struct {
	r         io.ReaderAt
	blockSize int64
	segments  []segment
	files     []fileRange
	walked    map[int64]bool
}

// layout returns the image's claimed segments in ascending offset order and its
// files' sector ranges in ascending order. It returns any error encountered while
// parsing the image structure.
func (i *ISO) layout() ([]segment, []fileRange, error) {
	b := builder{r: i.cooked, walked: map[int64]bool{}}
	if err := b.scan(); err != nil {
		return nil, nil, err
	}
	sort.Slice(b.segments, func(lhs, rhs int) bool {
		return b.segments[lhs].extent.Offset < b.segments[rhs].extent.Offset
	})
	sort.Slice(b.files, func(lhs, rhs int) bool {
		return b.files[lhs].start < b.files[rhs].start
	})
	return b.segments, b.files, nil
}

// scan claims the system area, the volume descriptor set, and then, from the
// primary descriptor, the path tables and the directory tree.
func (b *builder) scan() error {
	b.claimSystemArea()
	primary, err := b.scanDescriptors()
	if err != nil {
		return err
	}
	b.blockSize = int64(primary.LogicalBlockSize)
	if err := b.scanPathTables(primary); err != nil {
		return err
	}
	return b.scanDirectory(&primary.Root, "/")
}

// claimSystemArea claims the reserved sectors that precede the volume descriptor
// set.
func (b *builder) claimSystemArea() {
	extent := Extent{Offset: 0, Length: systemAreaSectors * logicalSectorSize, Block: 0}
	b.add(extent, func(rd *ISO, v Visitor) error {
		return v.VisitSystemArea(rd.region(extent))
	})
}

// scanDescriptors claims each descriptor of the volume descriptor set, reading
// sectors from the system area's end until a set terminator. It returns the
// primary volume descriptor, or [ErrNoPrimaryDescriptor] when the set declares
// none.
func (b *builder) scanDescriptors() (*PrimaryVolumeDescriptor, error) {
	var primary *PrimaryVolumeDescriptor
	for offset := int64(systemAreaSectors * logicalSectorSize); ; offset += logicalSectorSize {
		sector, err := b.read(offset, logicalSectorSize)
		if err != nil {
			return nil, err
		}
		descriptor, err := parseVolumeDescriptor(sector, offset)
		if err != nil {
			return nil, err
		}
		claimed := descriptor
		b.add(descriptor.Extent, func(_ *ISO, v Visitor) error {
			return v.VisitVolumeDescriptor(&claimed)
		})
		if descriptor.Primary != nil {
			primary = descriptor.Primary
		}
		if descriptor.Type == VolumeDescriptorTerminator {
			break
		}
	}
	if primary == nil {
		return nil, ErrNoPrimaryDescriptor
	}
	return primary, nil
}

// scanPathTables claims the records of every path table the primary descriptor
// locates, skipping the optional tables when they are absent.
func (b *builder) scanPathTables(primary *PrimaryVolumeDescriptor) error {
	tables := []struct {
		block uint32
		order binary.ByteOrder
	}{
		{primary.TypeLPathTable, binary.LittleEndian},
		{primary.OptTypeLPathTable, binary.LittleEndian},
		{primary.TypeMPathTable, binary.BigEndian},
		{primary.OptTypeMPathTable, binary.BigEndian},
	}
	for _, table := range tables {
		if table.block == 0 {
			continue
		}
		if err := b.scanPathTable(table.block, int(primary.PathTableSize), table.order); err != nil {
			return err
		}
	}
	return nil
}

// scanPathTable claims each record of the path table of size bytes located at
// block, reading its multi-byte fields with order.
func (b *builder) scanPathTable(block uint32, size int, order binary.ByteOrder) error {
	base := int64(block) * b.blockSize
	data, err := b.read(base, size)
	if err != nil {
		return err
	}
	for pos := 0; pos < size; {
		record, n, err := parsePathTableRecord(data[pos:], base+int64(pos), order)
		if err != nil {
			return err
		}
		if n == 0 {
			return ErrCorruptImage
		}
		claimed := record
		b.add(record.Extent, func(_ *ISO, v Visitor) error {
			return v.VisitPathTableRecord(&claimed)
		})
		pos += n
	}
	return nil
}

// scanDirectory claims the records of the directory located by record, recursing
// into its subdirectories and claiming the files it declares. path is the
// directory's absolute path, used to qualify the files within it. Already-walked
// extents are skipped so that cyclic or duplicated references terminate.
func (b *builder) scanDirectory(record *DirectoryRecord, path string) error {
	base := int64(record.DataBlock) * b.blockSize
	if b.walked[base] {
		return nil
	}
	b.walked[base] = true
	for off := int64(0); off < int64(record.DataLength); off += b.blockSize {
		block, err := b.read(base+off, int(b.blockSize))
		if err != nil {
			return err
		}
		if err := b.scanDirectoryBlock(block, base+off, path); err != nil {
			return err
		}
	}
	return nil
}

// scanDirectoryBlock claims the records packed into a single directory block that
// begins at base, descending into each as it is parsed.
func (b *builder) scanDirectoryBlock(block []byte, base int64, path string) error {
	for pos := 0; pos < len(block); {
		record, n, err := parseDirectoryRecord(block[pos:], base+int64(pos))
		if err != nil {
			return err
		}
		if n == 0 {
			break
		}
		claimed := record
		b.add(record.Extent, func(_ *ISO, v Visitor) error {
			return v.VisitDirectoryRecord(&claimed)
		})
		if err := b.descend(&claimed, path); err != nil {
			return err
		}
		pos += n
	}
	return nil
}

// descend recurses into a subdirectory record or claims a file record, ignoring
// the "." and ".." entries.
func (b *builder) descend(record *DirectoryRecord, path string) error {
	if record.isSpecial() {
		return nil
	}
	if record.IsDir() {
		return b.scanDirectory(record, path+record.Name+"/")
	}
	b.claimFile(record, path)
	return nil
}

// claimFile claims the data extent of the file declared by record, qualified by
// path, and records its sector range. Empty files and already-walked extents are
// not claimed.
func (b *builder) claimFile(record *DirectoryRecord, path string) {
	offset := int64(record.DataBlock) * b.blockSize
	if record.DataLength == 0 || b.walked[offset] {
		return
	}
	b.walked[offset] = true
	start := int(record.DataBlock)
	end := start + int((int64(record.DataLength)+b.blockSize-1)/b.blockSize)
	b.files = append(b.files, fileRange{start: start, end: end})
	file := File{
		Path:   path + record.Name,
		Name:   record.Name,
		Record: record,
		Extent: Extent{Offset: offset, Length: int64(record.DataLength), Block: int64(record.DataBlock)},
	}
	b.add(file.Extent, func(i *ISO, v Visitor) error {
		stream := FileStream{source: i.sectors, start: start, end: end, length: int64(record.DataLength)}
		return v.VisitFile(&file, &stream)
	})
}

// add appends a claimed segment over extent with the given visit action.
func (b *builder) add(extent Extent, visit func(rd *ISO, v Visitor) error) {
	b.segments = append(b.segments, segment{extent: extent, visit: visit})
}

// read returns size bytes at offset, or [ErrTruncated] when the image ends before
// size bytes are available.
func (b *builder) read(offset int64, size int) ([]byte, error) {
	buf := make([]byte, size)
	n, err := b.r.ReadAt(buf, offset)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if n < size {
		return nil, ErrTruncated
	}
	return buf, nil
}
