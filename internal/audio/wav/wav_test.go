package wav_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/audio/wav"
	"github.com/bitwizeshift/psx-decomp-tools/internal/testing/iotest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestEncode(t *testing.T) {
	t.Parallel()

	testErr := errors.New("write failed")
	testCases := []struct {
		name    string
		writer  io.Writer
		format  wav.Format
		pcm     []byte
		want    []byte
		wantErr error
	}{
		{
			name:    "HeaderWriteFails",
			writer:  iotest.TruncateErrWriter(0, testErr),
			format:  wav.Format{SampleRate: 44100, Channels: 2, BitsPerSample: 16},
			pcm:     []byte{0x01, 0x02, 0x03, 0x04},
			wantErr: testErr,
		}, {
			name:    "SampleWriteFails",
			writer:  iotest.TruncateErrWriter(44, testErr),
			format:  wav.Format{SampleRate: 44100, Channels: 2, BitsPerSample: 16},
			pcm:     []byte{0x01, 0x02, 0x03, 0x04},
			wantErr: testErr,
		},
		{
			name:   "StereoSample",
			writer: &bytes.Buffer{},
			format: wav.Format{SampleRate: 44100, Channels: 2, BitsPerSample: 16},
			pcm:    []byte{0x01, 0x02, 0x03, 0x04},
			want: []byte{
				'R', 'I', 'F', 'F', 0x28, 0x00, 0x00, 0x00, 'W', 'A', 'V', 'E',
				'f', 'm', 't', ' ', 0x10, 0x00, 0x00, 0x00, 0x01, 0x00, 0x02, 0x00,
				0x44, 0xac, 0x00, 0x00, 0x10, 0xb1, 0x02, 0x00, 0x04, 0x00, 0x10, 0x00,
				'd', 'a', 't', 'a', 0x04, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x04,
			},
			wantErr: nil,
		}, {
			name:    "InvalidFormat",
			writer:  &bytes.Buffer{},
			format:  wav.Format{SampleRate: 0, Channels: 2, BitsPerSample: 16},
			pcm:     []byte{0x01, 0x02},
			want:    nil,
			wantErr: wav.ErrInvalidFormat,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			writer := tc.writer

			// Act
			err := wav.Encode(writer, tc.format, tc.pcm)
			bytes := iotest.ReadAllFromWriter(t, writer)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Encode() error = %v, want %v", got, want)
			}
			if got, want := bytes, tc.want; !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
				t.Errorf("Encode() bytes diff (-got +want):\n%s", cmp.Diff(got, want, cmpopts.EquateEmpty()))
			}
		})
	}
}
