package isotest

import (
	"encoding/binary"
	"strings"
)

// parts splits an absolute path into its components.
func parts(p string) []string {
	return strings.Split(strings.Trim(p, "/"), "/")
}

// parentParts returns the components of the directory containing the path p.
func parentParts(p string) []string {
	all := parts(p)
	return all[:len(all)-1]
}

// breadthFirstDirs returns root and its descendant directories in breadth-first
// order.
func breadthFirstDirs(root *node) []*node {
	dirs := []*node{root}
	for i := 0; i < len(dirs); i++ {
		for _, child := range dirs[i].children {
			if child.isDir {
				dirs = append(dirs, child)
			}
		}
	}
	return dirs
}

// depthFirstFiles returns the files beneath n in depth-first order.
func depthFirstFiles(n *node) []*node {
	var files []*node
	for _, child := range n.children {
		if child.isDir {
			files = append(files, depthFirstFiles(child)...)
		} else {
			files = append(files, child)
		}
	}
	return files
}

// setParents records each directory's parent block, size, and path table number,
// treating the root as its own parent.
func setParents(n, parent *node) {
	n.parentBlk = parent.block
	n.parentSize = parent.size
	n.parentNum = parent.number
	for _, child := range n.children {
		if child.isDir {
			setParents(child, n)
		}
	}
}

// directoryByteSize returns the block-aligned size of dir's extent, accounting
// for the "." and ".." entries, its child records, and per-block padding.
func directoryByteSize(dir *node) uint32 {
	lengths := []int{recordLen(1), recordLen(1)}
	for _, child := range dir.children {
		lengths = append(lengths, recordLen(len(childIdentifier(child))))
	}
	pos := 0
	for _, length := range lengths {
		if pos%BlockSize+length > BlockSize {
			pos = (pos/BlockSize + 1) * BlockSize
		}
		pos += length
	}
	return uint32(sectorsFor(pos) * BlockSize)
}

// sectorsFor returns the number of whole sectors needed to hold n bytes.
func sectorsFor(n int) int {
	return (n + BlockSize - 1) / BlockSize
}

// recordLen returns the length of a directory record whose identifier is idLen
// bytes, padded so the record occupies an even number of bytes.
func recordLen(idLen int) int {
	length := dirRecordFixed + idLen
	if length%2 == 1 {
		length++
	}
	return length
}

// childIdentifier returns the file identifier under which n appears in its
// parent's directory extent, appending the ";1" version to files.
func childIdentifier(n *node) string {
	if n.isDir {
		return n.name
	}
	return n.name + ";1"
}

// pathIdentifier returns the identifier under which n appears in a path table,
// mapping the root to its "\x00" identifier.
func pathIdentifier(n *node) string {
	if n.name == "" {
		return "\x00"
	}
	return n.name
}

// renderDirectory returns dir's extent: the "." and ".." records followed by one
// record per child, packed into logical blocks.
func renderDirectory(dir *node) []byte {
	w := blockWriter{buf: make([]byte, dir.size)}
	w.write(encodeDirRecord("\x00", dir.block, dir.size, true))
	w.write(encodeDirRecord("\x01", dir.parentBlk, dir.parentSize, true))
	for _, child := range dir.children {
		w.write(encodeDirRecord(childIdentifier(child), child.block, child.size, child.isDir))
	}
	return w.buf
}

// blockWriter packs variable-length records into a buffer without letting a
// record straddle a logical block boundary.
type blockWriter struct {
	buf []byte
	pos int
}

// write copies rec into the buffer, advancing to the next block first when rec
// would otherwise straddle a block boundary.
func (w *blockWriter) write(rec []byte) {
	if w.pos%BlockSize+len(rec) > BlockSize {
		w.pos = (w.pos/BlockSize + 1) * BlockSize
	}
	copy(w.buf[w.pos:], rec)
	w.pos += len(rec)
}

// encodeDirRecord encodes a directory record naming id whose data begins at block
// and spans length bytes, marking it a directory when isDir is set.
func encodeDirRecord(id string, block, length uint32, isDir bool) []byte {
	rec := make([]byte, recordLen(len(id)))
	rec[0] = byte(len(rec))
	putBoth32(rec[2:10], block)
	putBoth32(rec[10:18], length)
	if isDir {
		rec[25] = byte(0x02)
	}
	putBoth16(rec[28:32], 1)
	rec[32] = byte(len(id))
	copy(rec[33:], id)
	return rec
}

// pathTableBytes encodes the path table for dirs with the given byte order, one
// record per directory in breadth-first order.
func pathTableBytes(dirs []*node, order binary.ByteOrder) []byte {
	var out []byte
	for _, dir := range dirs {
		out = append(out, encodePathRecord(pathIdentifier(dir), dir.block, uint16(dir.parentNum), order)...)
	}
	return out
}

// encodePathRecord encodes a path table record naming id, whose directory begins
// at block and whose parent has the given path table number.
func encodePathRecord(id string, block uint32, parent uint16, order binary.ByteOrder) []byte {
	rec := make([]byte, pathRecordFixed+len(id)+(len(id)&1))
	rec[0] = byte(len(id))
	order.PutUint32(rec[2:6], block)
	order.PutUint16(rec[6:8], parent)
	copy(rec[8:], id)
	return rec
}

// writePrimary writes a primary volume descriptor into dst describing a volume of
// totalBlocks blocks whose path tables are pathTableSize bytes.
func writePrimary(dst []byte, root *node, totalBlocks uint32, pathTableSize int) {
	dst[0] = byte(1)
	copy(dst[1:6], standardIdentifier)
	dst[6] = byte(1)
	space(dst[8:40])
	space(dst[40:72])
	putBoth32(dst[80:88], totalBlocks)
	putBoth16(dst[120:124], 1)
	putBoth16(dst[124:128], 1)
	putBoth16(dst[128:132], BlockSize)
	putBoth32(dst[132:140], uint32(pathTableSize))
	binary.LittleEndian.PutUint32(dst[140:144], typeLSector)
	binary.BigEndian.PutUint32(dst[148:152], typeMSector)
	copy(dst[156:190], encodeDirRecord("\x00", root.block, root.size, true))
	for _, field := range [][2]int{{190, 318}, {318, 446}, {446, 574}, {574, 702}, {702, 739}, {739, 776}, {776, 813}} {
		space(dst[field[0]:field[1]])
	}
	for _, field := range [][2]int{{813, 830}, {830, 847}, {847, 864}, {864, 881}} {
		unsetDate(dst[field[0]:field[1]])
	}
}

// writeTerminator writes a volume descriptor set terminator into dst.
func writeTerminator(dst []byte) {
	dst[0] = byte(255)
	copy(dst[1:6], standardIdentifier)
	dst[6] = byte(1)
}

// standardIdentifier is the value of every volume descriptor's standard
// identifier field.
const standardIdentifier = "CD001"

// space fills b with the ASCII spaces ISO 9660 uses to pad fixed-width strings.
func space(b []byte) {
	for i := range b {
		b[i] = byte(' ')
	}
}

// unsetDate fills b with the 16 ASCII zeros and zero offset that denote an unset
// volume timestamp.
func unsetDate(b []byte) {
	for i := range 16 {
		b[i] = byte('0')
	}
	b[16] = byte(0)
}

// putBoth16 writes v into b in ISO 9660 both-byte order: little-endian then
// big-endian.
func putBoth16(b []byte, v uint16) {
	binary.LittleEndian.PutUint16(b[0:2], v)
	binary.BigEndian.PutUint16(b[2:4], v)
}

// putBoth32 writes v into b in ISO 9660 both-byte order: little-endian then
// big-endian.
func putBoth32(b []byte, v uint32) {
	binary.LittleEndian.PutUint32(b[0:4], v)
	binary.BigEndian.PutUint32(b[4:8], v)
}
