package iso_test

import (
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/iso"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestBaseVisitor(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		call func(iso.Visitor) error
	}{
		{
			name: "SystemArea",
			call: func(v iso.Visitor) error { return v.VisitSystemArea(&iso.Region{}) },
		},
		{
			name: "VolumeDescriptor",
			call: func(v iso.Visitor) error { return v.VisitVolumeDescriptor(&iso.VolumeDescriptor{}) },
		},
		{
			name: "PathTableRecord",
			call: func(v iso.Visitor) error { return v.VisitPathTableRecord(&iso.PathTableRecord{}) },
		},
		{
			name: "DirectoryRecord",
			call: func(v iso.Visitor) error { return v.VisitDirectoryRecord(&iso.DirectoryRecord{}) },
		},
		{
			name: "File",
			call: func(v iso.Visitor) error { return v.VisitFile(&iso.File{}) },
		},
		{
			name: "Unreferenced",
			call: func(v iso.Visitor) error { return v.VisitUnreferenced(&iso.Region{}) },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			var sut iso.BaseVisitor

			// Act
			err := tc.call(sut)

			// Assert
			opts := cmpopts.EquateErrors()
			if got, want := err, error(nil); !cmp.Equal(got, want, opts) {
				t.Errorf("BaseVisitor.Visit%s(...) = error %v, want %v", tc.name, got, want)
			}
		})
	}
}
