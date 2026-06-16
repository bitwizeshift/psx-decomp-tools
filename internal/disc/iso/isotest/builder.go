package isotest

import (
	"encoding/binary"
	"path"
)

// BlockSize is the logical block and sector size of every image isotest builds.
const BlockSize = 2048

// Fixed sector assignments shared by every image.
const (
	pvdSector        = 16
	terminatorSector = 17
	typeLSector      = 18
	typeMSector      = 19
	firstDataSector  = 20
)

// dirRecordFixed and pathRecordFixed are the sizes of the fixed portions of a
// directory record and a path table record.
const (
	dirRecordFixed  = 33
	pathRecordFixed = 8
)

// node is a file or directory within a builder's tree.
type node struct {
	name     string
	isDir    bool
	data     []byte
	children []*node

	block      uint32
	size       uint32
	sectors    int
	number     int
	parentNum  int
	parentBlk  uint32
	parentSize uint32
}

// Builder assembles an ISO 9660 image in memory from a directory tree.
type Builder struct {
	root      *node
	gap       int
	trailing  []byte
	endSector uint32
}

// New returns a [Builder] whose tree contains only the root directory.
func New() *Builder {
	return &Builder{root: &node{name: "", isDir: true}}
}

// AddFile adds a file at the given absolute path with the given contents,
// creating any missing parent directories. It returns the builder for chaining.
func (b *Builder) AddFile(p string, data []byte) *Builder {
	dir := b.mkdirAll(parentParts(p))
	dir.children = append(dir.children, &node{name: path.Base(p), data: data})
	return b
}

// AddDir adds an empty directory at the given absolute path, creating any
// missing parent directories. It returns the builder for chaining.
func (b *Builder) AddDir(p string) *Builder {
	b.mkdirAll(parts(p))
	return b
}

// Gap inserts n empty sectors between the directory region and the file region,
// yielding an unreferenced range in the built image. It returns the builder for
// chaining.
func (b *Builder) Gap(n int) *Builder {
	b.gap = n
	return b
}

// Trailing appends data to the image after the final extent, yielding an
// unreferenced range that runs to the end of the image. It returns the builder
// for chaining.
func (b *Builder) Trailing(data []byte) *Builder {
	b.trailing = data
	return b
}

// Build lays out the tree and renders the complete, well-formed image.
func (b *Builder) Build() []byte {
	return b.build(nil)
}

// BuildDuplicateFileExtent renders an image whose first two root files share a
// single data extent, as a hard link would. It panics unless the root directory
// declares at least two files.
func (b *Builder) BuildDuplicateFileExtent() []byte {
	return b.build(func(dirs []*node, files []*node) {
		files[1].block = files[0].block
	})
}

// BuildDuplicateDirectoryExtent renders an image whose first two subdirectories
// share a single extent, as a cyclic or duplicated reference would. It panics
// unless the root directory declares at least two subdirectories.
func (b *Builder) BuildDuplicateDirectoryExtent() []byte {
	return b.build(func(dirs []*node, files []*node) {
		dirs[2].block = dirs[1].block
		dirs[2].size = dirs[1].size
		dirs[2].sectors = dirs[1].sectors
	})
}

// NoPrimaryImage returns the smallest image whose volume descriptor set holds
// only a terminator, declaring no primary volume descriptor.
func NoPrimaryImage() []byte {
	image := make([]byte, (pvdSector+1)*BlockSize)
	writeTerminator(image[pvdSector*BlockSize:])
	return image
}

// build lays out the tree, applies mutate to the assigned directories and files,
// then renders the image.
func (b *Builder) build(mutate func(dirs []*node, files []*node)) []byte {
	dirs, files := b.layout()
	if mutate != nil {
		mutate(dirs, files)
	}
	return b.render(dirs, files)
}

// layout assigns block addresses and sizes to every directory and file in the
// tree, returning the directories in breadth-first order and the files in
// depth-first order.
func (b *Builder) layout() (dirs []*node, files []*node) {
	b.root.isDir = true
	dirs = breadthFirstDirs(b.root)
	for _, dir := range dirs {
		dir.size = directoryByteSize(dir)
		dir.sectors = int(dir.size) / BlockSize
	}
	sector := uint32(firstDataSector)
	for i, dir := range dirs {
		dir.number = i + 1
		dir.block = sector
		sector += uint32(dir.sectors)
	}
	setParents(b.root, b.root)
	sector += uint32(b.gap)
	files = depthFirstFiles(b.root)
	for _, file := range files {
		file.size = uint32(len(file.data))
		if len(file.data) == 0 {
			continue
		}
		file.sectors = sectorsFor(len(file.data))
		file.block = sector
		sector += uint32(file.sectors)
	}
	b.endSector = sector
	return dirs, files
}

// render writes the laid-out image: system area, primary volume descriptor,
// terminator, path tables, directory extents, file extents, and trailing data.
func (b *Builder) render(dirs []*node, files []*node) []byte {
	image := make([]byte, int(b.endSector)*BlockSize+len(b.trailing))
	typeL := pathTableBytes(dirs, binary.LittleEndian)
	typeM := pathTableBytes(dirs, binary.BigEndian)
	writePrimary(image[pvdSector*BlockSize:], b.root, b.endSector, len(typeL))
	writeTerminator(image[terminatorSector*BlockSize:])
	copy(image[typeLSector*BlockSize:], typeL)
	copy(image[typeMSector*BlockSize:], typeM)
	for _, dir := range dirs {
		copy(image[int(dir.block)*BlockSize:], renderDirectory(dir))
	}
	for _, file := range files {
		copy(image[int(file.block)*BlockSize:], file.data)
	}
	copy(image[int(b.endSector)*BlockSize:], b.trailing)
	return image
}

// mkdirAll returns the directory named by the path parts, creating intermediate
// directories under the root as needed.
func (b *Builder) mkdirAll(names []string) *node {
	dir := b.root
	for _, name := range names {
		dir = dir.child(name)
	}
	return dir
}

// child returns the subdirectory of n with the given name, creating it when
// absent.
func (n *node) child(name string) *node {
	for _, c := range n.children {
		if c.isDir && c.name == name {
			return c
		}
	}
	created := &node{name: name, isDir: true}
	n.children = append(n.children, created)
	return created
}
