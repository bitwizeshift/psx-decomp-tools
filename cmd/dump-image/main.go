// Command dump-image reads the CUE sheet named on the command line and extracts
// the ISO 9660 volume that spans its tracks into the directory named by --output.
// Files are written under iso/, the bytes of the system area and each descriptor
// and record under regions/, and any non-zero unreferenced byte range under
// unreferenced/.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/iso"
)

func main() {
	log.SetFlags(0)
	output := flag.String("output", "", "directory to extract the image into")
	flag.Parse()
	if *output == "" || flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: dump-image --output <dir> <image.cue>")
		os.Exit(2)
	}
	if err := run(flag.Arg(0), *output); err != nil {
		log.Fatal(err)
	}
}

// run opens the disc described by the CUE sheet at path and extracts its ISO 9660
// volume, which spans every track, beneath output. It returns any error from
// opening the disc, parsing the volume, writing a file, or closing the disc's
// backing files.
func run(path, output string) (err error) {
	d, err := disc.FromCueFile(path)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := d.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	source, err := d.Volume()
	if err != nil {
		return err
	}
	reader := iso.SectorReader{Source: source}
	return iso.New(reader).Visit(&extractVisitor{reader: reader, root: output, index: map[string]int{}})
}

// extractVisitor writes the parts of an ISO 9660 image beneath root: files under
// iso/, the system area and each descriptor and record under regions/, and each
// non-zero unreferenced range under unreferenced/.
type extractVisitor struct {
	iso.BaseVisitor
	reader io.ReaderAt
	root   string
	index  map[string]int
}

// VisitSystemArea writes the reserved system area into regions/.
func (v *extractVisitor) VisitSystemArea(r *iso.Region) error {
	return v.writeRegion("system", r.Extent)
}

// VisitVolumeDescriptor writes the descriptor into regions/, logging the volume
// the primary descriptor names.
func (v *extractVisitor) VisitVolumeDescriptor(d *iso.VolumeDescriptor) error {
	if d.Primary != nil {
		log.Printf("volume %q", d.Primary.VolumeIdentifier)
	}
	return v.writeRegion("volume-descriptor", d.Extent)
}

// VisitPathTableRecord writes the path table record into regions/.
func (v *extractVisitor) VisitPathTableRecord(rec *iso.PathTableRecord) error {
	return v.writeRegion("path-table-record", rec.Extent)
}

// VisitDirectoryRecord writes the directory record into regions/.
func (v *extractVisitor) VisitDirectoryRecord(rec *iso.DirectoryRecord) error {
	return v.writeRegion("directory-record", rec.Extent)
}

// VisitUnreferenced writes the range into unreferenced/, unless every byte of it
// is zero.
func (v *extractVisitor) VisitUnreferenced(r *iso.Region) error {
	data, err := io.ReadAll(r.Data)
	if err != nil {
		return err
	}
	if allZero(data) {
		return nil
	}
	return v.writeBytes("unreferenced", fmt.Sprintf("%d.bin", v.next("unreferenced")), data)
}

// VisitFile writes the file's contents to its path beneath iso/, creating any
// parent directories. It returns any error from creating the directories, writing
// the file, or reading the file's data.
func (v *extractVisitor) VisitFile(f *iso.File) (err error) {
	rel := filepath.Join("iso", filepath.FromSlash(f.Path))
	dest := filepath.Join(v.root, rel)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := out.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	written, err := io.Copy(out, f.Data)
	if err != nil {
		return err
	}
	if written < f.Extent.Length {
		log.Printf("  extracted %s (%d of %d bytes; extent runs past the end of the image)", rel, written, f.Extent.Length)
		return nil
	}
	log.Printf("  extracted %s (%d bytes)", rel, written)
	return nil
}

// writeRegion reads the bytes of extent from the image and writes them into
// regions/ as kind-N.bin, where N counts the regions of that kind.
func (v *extractVisitor) writeRegion(kind string, extent iso.Extent) error {
	buf := make([]byte, extent.Length)
	n, err := v.reader.ReadAt(buf, extent.Offset)
	if err != nil && err != io.EOF {
		return err
	}
	return v.writeBytes("regions", fmt.Sprintf("%s-%d.bin", kind, v.next(kind)), buf[:n])
}

// writeBytes writes data to name within subdir of the visitor's root, creating
// the subdirectory.
func (v *extractVisitor) writeBytes(subdir, name string, data []byte) error {
	dir := filepath.Join(v.root, subdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name), data, 0o644)
}

// next returns the current count for kind and advances it.
func (v *extractVisitor) next(kind string) int {
	n := v.index[kind]
	v.index[kind]++
	return n
}

// allZero reports whether every byte of b is zero.
func allZero(b []byte) bool {
	for _, c := range b {
		if c != 0 {
			return false
		}
	}
	return true
}
