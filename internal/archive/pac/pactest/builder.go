package pactest

import (
	"encoding/binary"

	"github.com/bitwizeshift/psx-decomp-tools/internal/archive/pac"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// Node specifies a PAC node to render into a test archive. Construct one with
// [Leaf], [Dir], or [Raw].
type Node struct {
	kind     uint32
	payload  []byte
	children []Node
	dir      bool
	raw      bool
}

// Leaf returns a leaf node carrying the given kind word and payload bytes.
func Leaf(kind uint32, payload []byte) Node {
	return Node{kind: kind, payload: payload}
}

// Dir returns a directory node containing the given children, laid out in order.
// A nested directory or [Raw] child must appear last, matching the layout the pac
// package expects.
func Dir(children ...Node) Node {
	return Node{dir: true, children: children}
}

// DirData returns a directory node that carries the given inline data in its
// header region, ahead of its children.
func DirData(data []byte, children ...Node) Node {
	return Node{dir: true, payload: data, children: children}
}

// Raw returns a non-PAC child of the given bytes, used to exercise the raw
// remainder handling of a directory. It must appear as the last child of its
// directory.
func Raw(payload []byte) Node {
	return Node{raw: true, payload: payload}
}

// Build renders root into a complete PAC archive in memory.
func Build(root Node) []byte {
	return root.render()
}

// render returns the sector-aligned block of bytes the node occupies, including
// its header.
func (n Node) render() []byte {
	switch {
	case n.raw:
		return pad(n.payload)
	case n.dir:
		return n.renderDir()
	default:
		return n.renderLeaf()
	}
}

// renderLeaf renders a leaf node: a 16-byte header whose total word counts the
// header and payload, followed by the payload, padded to a sector boundary.
func (n Node) renderLeaf() []byte {
	total := 16 + len(n.payload)
	buf := make([]byte, align(total))
	writeHeader(buf, n.kind, 0, uint32(total))
	copy(buf[16:], n.payload)
	return buf
}

// renderDir renders a directory node: a header sized to hold any inline data,
// followed by its children's blocks laid out contiguously. The header word counts
// the header sectors less one, matching how the pac package locates the children.
func (n Node) renderDir() []byte {
	headerSectors := headerSectorsFor(len(n.payload))
	headerBytes := (headerSectors + 1) * pac.SectorSize

	var children []byte
	for _, child := range n.children {
		children = append(children, child.render()...)
	}

	buf := make([]byte, headerBytes+len(children))
	writeHeader(buf, 0, uint32(headerSectors), uint32(headerBytes))
	copy(buf[16:], n.payload)
	copy(buf[headerBytes:], children)
	return buf
}

// headerSectorsFor returns the header word for a directory whose inline data is
// the given number of bytes: at least one, and enough that the header holds the
// 16-byte node header plus the data.
func headerSectorsFor(dataLen int) int {
	sectors := 1
	for (sectors+1)*pac.SectorSize < 16+dataLen {
		sectors++
	}
	return sectors
}

// CompareFiles returns a [cmp.Option] that compares [pac.File] values by the
// fields that identify a payload, ignoring the unexported reader they carry.
func CompareFiles() cmp.Option {
	return cmpopts.IgnoreUnexported(pac.File{})
}

// writeHeader writes a 16-byte PAC node header into buf.
func writeHeader(buf []byte, kind, hdr, total uint32) {
	copy(buf[0:4], []byte{'P', 'A', 'C', 0x00})
	binary.LittleEndian.PutUint32(buf[4:8], kind)
	binary.LittleEndian.PutUint32(buf[8:12], hdr)
	binary.LittleEndian.PutUint32(buf[12:16], total)
}

// pad returns data extended with zero bytes to the next sector boundary.
func pad(data []byte) []byte {
	buf := make([]byte, align(len(data)))
	copy(buf, data)
	return buf
}

// align rounds n up to the next [pac.SectorSize] boundary.
func align(n int) int {
	return (n + pac.SectorSize - 1) &^ (pac.SectorSize - 1)
}
