package iso_test

import (
	"bytes"
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/iso"
	"github.com/google/go-cmp/cmp"
)

func TestLooksLikeImage(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		content []byte
		want    bool
	}{
		{
			name:    "ValidImage",
			content: imageWithIdentifier("CD001"),
			want:    true,
		}, {
			name:    "WrongIdentifier",
			content: imageWithIdentifier("XX001"),
			want:    false,
		}, {
			name:    "TooSmall",
			content: make([]byte, 1024),
			want:    false,
		}, {
			name:    "Empty",
			content: nil,
			want:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			reader := bytes.NewReader(tc.content)

			// Act
			got := iso.LooksLikeImage(reader)

			// Assert
			if got, want := got, tc.want; !cmp.Equal(got, want) {
				t.Errorf("LooksLikeImage() = %v, want %v", got, want)
			}
		})
	}
}

// imageWithIdentifier returns a buffer with the given 5-byte standard identifier
// at the start of the first volume descriptor sector (logical sector 16).
func imageWithIdentifier(id string) []byte {
	const descriptorOffset = 16 * 2048
	buf := make([]byte, descriptorOffset+8)
	copy(buf[descriptorOffset+1:], id)
	return buf
}
