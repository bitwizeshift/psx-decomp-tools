// Command dump-image reads the CUE sheet named on the command line and extracts
// the ISO 9660 volume that spans its tracks into the directory named by --output.
// Files are written under iso/ as their raw, form-aware user data (with a
// .xa subheader sidecar for XA files), the bytes of the system area and each
// descriptor and record under regions/, non-zero unreferenced ranges under
// unreferenced/, and the trailing bytes of sectors belonging to no file under
// unexpected/.
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
	return iso.FromSectorSource(source).Visit(&extractVisitor{reader: reader, root: output, index: map[string]int{}})
}

// extractVisitor writes the parts of an ISO 9660 image beneath root: raw
// files under iso/, the system area and each descriptor and record under regions/,
// each non-zero unreferenced range under unreferenced/, and each non-file sector
// tail under unexpected/.
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

// VisitUnexpected writes a sector's trailing bytes, beyond its ISO logical block
// and belonging to no file, into unexpected/ named by the sector's logical block,
// unless every byte is zero.
func (v *extractVisitor) VisitUnexpected(u *iso.Unexpected) error {
	data, err := io.ReadAll(u.Data)
	if err != nil {
		return err
	}
	if allZero(data) {
		return nil
	}
	return v.writeBytes("unexpected", fmt.Sprintf("%d.bin", u.Block), data)
}

// VisitFile writes the file's raw contents to its path beneath iso/, creating
// any parent directories, and writes a per-sector XA subheader sidecar (.xa) when
// the file carries any. It returns any error from creating the directories,
// writing, or reading the file's sectors.
func (v *extractVisitor) VisitFile(f *iso.File, s *iso.FileStream) (err error) {
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
	var written int64
	var subheaders []*iso.Subheader
	xa := false
	listener := iso.ListenerFunc(func(data []byte, sub *iso.Subheader) error {
		n, err := out.Write(data)
		written += int64(n)
		if err != nil {
			return err
		}
		subheaders = append(subheaders, sub)
		xa = xa || sub != nil
		return nil
	})
	if err := s.Stream(listener); err != nil {
		return err
	}
	if xa {
		if err := os.WriteFile(dest+".xa", encodeSubheaders(subheaders), 0o644); err != nil {
			return err
		}
	}
	log.Printf("  extracted %s (%d bytes)", rel, written)
	return nil
}

// encodeSubheaders packs each sector's File, Channel, SubMode, and Coding bytes
// (zeros where a sector has no subheader) into a compact sidecar.
func encodeSubheaders(subheaders []*iso.Subheader) []byte {
	buf := make([]byte, 0, len(subheaders)*4)
	for _, sub := range subheaders {
		if sub == nil {
			buf = append(buf, 0, 0, 0, 0)
			continue
		}
		buf = append(buf, sub.File, sub.Channel, sub.SubMode, sub.Coding)
	}
	return buf
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
