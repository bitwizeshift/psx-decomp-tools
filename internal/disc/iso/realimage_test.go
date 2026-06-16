package iso_test

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/iso"
	"github.com/google/go-cmp/cmp"
)

// testdata/minimal.iso is a real ISO 9660 image produced by genisoimage. It was
// generated reproducibly, without installing any host tools, with:
//
//	docker run --rm -v "$PWD/internal/disc/iso/testdata:/out" ubuntu:latest bash -c '
//	  apt-get update && apt-get install -y genisoimage &&
//	  mkdir -p /src/DATA &&
//	  printf "BOOT=cdrom:\SLUS_000.01;1" > /src/SYSTEM.CNF &&
//	  printf "hello world" > /src/DATA/HELLO.TXT &&
//	  genisoimage -quiet -iso-level 1 -V PSXTEST -o /out/minimal.iso /src'
//
// The expectations below are derived by hand from those inputs, not from the
// extractor's own output.

// realImageVisitor records the primary volume descriptor and the files of a real
// image.
type realImageVisitor struct {
	iso.BaseVisitor
	primary *iso.PrimaryVolumeDescriptor
	files   []fileData
}

func (v *realImageVisitor) VisitVolumeDescriptor(d *iso.VolumeDescriptor) error {
	if d.Primary != nil {
		v.primary = d.Primary
	}
	return nil
}

func (v *realImageVisitor) VisitFile(f *iso.File) error {
	data, _ := io.ReadAll(f.Data)
	v.files = append(v.files, fileData{Path: f.Path, Name: f.Name, Data: string(data)})
	return nil
}

func TestRealImageFiles(t *testing.T) {
	t.Parallel()

	// Arrange
	image, err := os.ReadFile("testdata/minimal.iso")
	if err != nil {
		t.Fatalf("ReadFile(...) = unexpected error %v", err)
	}
	sut := &realImageVisitor{}

	// Act
	if err := iso.New(bytes.NewReader(image)).Visit(sut); err != nil {
		t.Fatalf("Visit(...) = unexpected error %v", err)
	}

	// Assert
	want := []fileData{
		{Path: "/SYSTEM.CNF", Name: "SYSTEM.CNF", Data: `BOOT=cdrom:\SLUS_000.01;1`},
		{Path: "/DATA/HELLO.TXT", Name: "HELLO.TXT", Data: `hello world`},
	}
	if got := sut.files; !cmp.Equal(got, want) {
		t.Errorf("Visit(...) files = mismatch (-want +got):\n%s", cmp.Diff(want, got))
	}
}

func TestRealImagePrimaryDescriptor(t *testing.T) {
	t.Parallel()

	// Arrange
	image, err := os.ReadFile("testdata/minimal.iso")
	if err != nil {
		t.Fatalf("ReadFile(...) = unexpected error %v", err)
	}
	sut := &realImageVisitor{}

	// Act
	if err := iso.New(bytes.NewReader(image)).Visit(sut); err != nil {
		t.Fatalf("Visit(...) = unexpected error %v", err)
	}

	// Assert
	if got, want := sut.primary.VolumeIdentifier, "PSXTEST"; !cmp.Equal(got, want) {
		t.Errorf("Visit(...) volume identifier = %q, want %q", got, want)
	}
	if got, want := sut.primary.LogicalBlockSize, uint16(2048); !cmp.Equal(got, want) {
		t.Errorf("Visit(...) logical block size = %d, want %d", got, want)
	}
	if got, want := sut.primary.VolumeSpaceSize, uint32(177); !cmp.Equal(got, want) {
		t.Errorf("Visit(...) volume space size = %d, want %d", got, want)
	}
}

func TestRealImageTiles(t *testing.T) {
	t.Parallel()

	// Arrange
	image, err := os.ReadFile("testdata/minimal.iso")
	if err != nil {
		t.Fatalf("ReadFile(...) = unexpected error %v", err)
	}

	// Act
	events := recordVisit(t, image)

	// Assert
	assertTiles(t, events, int64(len(image)))
}
