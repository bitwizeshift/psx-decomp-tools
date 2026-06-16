package disc_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/disc"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/cue"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/disctest"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track"
	"github.com/bitwizeshift/psx-decomp-tools/internal/disc/track/tracktest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// sectorMSF is the address written into every built sector; its value does not
// affect the decoded payload.
var sectorMSF = cue.MSF{Minute: 0, Second: 0, Frame: 0}

// mode2MixedRaw returns the raw bytes of an XA Mode 2 track holding one Form 1
// and one Form 2 sector, alongside the logical payload they decode to.
func mode2MixedRaw() (raw, payload []byte) {
	sub := track.Subheader{File: 1, Channel: 1}
	form1 := dataBytes(2048)
	form2 := bytes.Repeat([]byte{0x5A}, 2324)
	first := tracktest.BuildMode2Form1Sector(sub, sectorMSF, form1)
	second := tracktest.BuildMode2Form2Sector(sub, sectorMSF, form2)
	raw = append(append([]byte{}, first...), second...)
	payload = append(append([]byte{}, form1...), form2...)
	return raw, payload
}

// corruptMode1Raw returns a full Mode 1 sector whose EDC no longer matches its
// data, so decoding it reports [track.ErrChecksum].
func corruptMode1Raw() []byte {
	sector := tracktest.BuildMode1Sector(sectorMSF, dataBytes(2048))
	sector[20] ^= 0xFF
	return sector
}

// errWriter is an [io.Writer] that always fails with err.
type errWriter struct{ err error }

func (w errWriter) Write([]byte) (int, error) { return 0, w.err }

func TestTrackMetadata(t *testing.T) {
	t.Parallel()

	mixed, _ := mode2MixedRaw()

	testCases := []struct {
		name        string
		mode        cue.Mode
		raw         []byte
		wantMode    cue.Mode
		wantSize    int
		wantSectors int
	}{
		{
			name:        "BareMode1",
			mode:        cue.ModeMode1_2048,
			raw:         dataBytes(2 * 2048),
			wantMode:    cue.ModeMode1_2048,
			wantSize:    2048,
			wantSectors: 2,
		},
		{
			name:        "XAMode2",
			mode:        cue.ModeMode2_2352,
			raw:         mixed,
			wantMode:    cue.ModeMode2_2352,
			wantSize:    2352,
			wantSectors: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := disctest.MustOpenTrack(tc.mode, tc.raw)

			// Act
			number := sut.Number()
			mode := sut.Mode()
			size := sut.SectorSize()
			sectors := sut.SectorCount()

			// Assert
			if got, want := number, 1; !cmp.Equal(got, want) {
				t.Errorf("Track.Number() = %d, want %d", got, want)
			}
			if got, want := mode, tc.wantMode; !cmp.Equal(got, want) {
				t.Errorf("Track.Mode() = %v, want %v", got, want)
			}
			if got, want := size, tc.wantSize; !cmp.Equal(got, want) {
				t.Errorf("Track.SectorSize() = %d, want %d", got, want)
			}
			if got, want := sectors, tc.wantSectors; !cmp.Equal(got, want) {
				t.Errorf("Track.SectorCount() = %d, want %d", got, want)
			}
		})
	}
}

func TestTrackCueTrack(t *testing.T) {
	t.Parallel()

	// Arrange
	sut := disctest.MustOpenTrack(cue.ModeMode1_2048, dataBytes(2048))
	want := cue.Track{
		Number:  1,
		Mode:    cue.ModeMode1_2048,
		Type:    cue.TypeBinary,
		File:    disctest.TrackFileName,
		Indices: []cue.Index{{Number: 1, Offset: cue.MSF{}}},
	}

	// Act
	cueTrack := sut.CueTrack()

	// Assert
	if got, want := cueTrack, want; !cmp.Equal(got, want) {
		t.Errorf("Track.CueTrack() = mismatch (-want +got):\n%s", cmp.Diff(want, got))
	}
}

func TestTrackReadSector(t *testing.T) {
	t.Parallel()

	raw := dataBytes(2 * 2048)

	testCases := []struct {
		name    string
		index   int
		want    track.Sector
		wantErr error
	}{
		{
			name:    "FirstSector",
			index:   0,
			want:    track.Sector{Number: 0, UserData: raw[0:2048], Raw: raw[0:2048]},
			wantErr: nil,
		},
		{
			name:    "SecondSector",
			index:   1,
			want:    track.Sector{Number: 1, UserData: raw[2048:4096], Raw: raw[2048:4096]},
			wantErr: nil,
		},
		{
			name:    "PastEnd",
			index:   2,
			want:    track.Sector{},
			wantErr: io.EOF,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := disctest.MustOpenTrack(cue.ModeMode1_2048, raw)

			// Act
			sector, err := sut.ReadSector(tc.index)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Track.ReadSector(%d) = error %v, want %v", tc.index, got, want)
			}
			if got, want := sector, tc.want; !cmp.Equal(got, want) {
				t.Errorf("Track.ReadSector(%d) = mismatch (-want +got):\n%s", tc.index, cmp.Diff(want, got))
			}
		})
	}
}

func TestTrackReadAt(t *testing.T) {
	t.Parallel()

	payload := dataBytes(3 * 2048)

	testCases := []struct {
		name    string
		length  int
		off     int64
		wantN   int
		want    []byte
		wantErr error
	}{
		{
			name:    "WithinFirstSector",
			length:  100,
			off:     0,
			wantN:   100,
			want:    payload[0:100],
			wantErr: nil,
		},
		{
			name:    "AtSectorBoundary",
			length:  100,
			off:     2048,
			wantN:   100,
			want:    payload[2048:2148],
			wantErr: nil,
		},
		{
			name:    "SpanningSectors",
			length:  200,
			off:     2000,
			wantN:   200,
			want:    payload[2000:2200],
			wantErr: nil,
		},
		{
			name:    "PartialAtEnd",
			length:  10,
			off:     int64(len(payload)) - 4,
			wantN:   4,
			want:    payload[len(payload)-4:],
			wantErr: io.EOF,
		},
		{
			name:    "AtEnd",
			length:  10,
			off:     int64(len(payload)),
			wantN:   0,
			want:    nil,
			wantErr: io.EOF,
		},
		{
			name:    "NegativeOffset",
			length:  10,
			off:     -1,
			wantN:   0,
			want:    nil,
			wantErr: disc.ErrNegativeOffset,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := disctest.MustOpenTrack(cue.ModeMode1_2048, payload)
			buf := make([]byte, tc.length)

			// Act
			read, err := sut.ReadAt(buf, tc.off)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Track.ReadAt(p, %d) = error %v, want %v", tc.off, got, want)
			}
			if got, want := read, tc.wantN; !cmp.Equal(got, want) {
				t.Errorf("Track.ReadAt(p, %d) = %d bytes, want %d", tc.off, got, want)
			}
			if got, want := buf[:read], tc.want; !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
				t.Errorf("Track.ReadAt(p, %d) = mismatch (-want +got):\n%s", tc.off, cmp.Diff(want, got))
			}
		})
	}
}

func TestTrackRead(t *testing.T) {
	t.Parallel()

	bare := dataBytes(2 * 2048)
	mixedRaw, mixedPayload := mode2MixedRaw()

	testCases := []struct {
		name    string
		mode    cue.Mode
		raw     []byte
		want    []byte
		wantErr error
	}{
		{
			name:    "BareMode1",
			mode:    cue.ModeMode1_2048,
			raw:     bare,
			want:    bare,
			wantErr: nil,
		},
		{
			name:    "XAMixedForms",
			mode:    cue.ModeMode2_2352,
			raw:     mixedRaw,
			want:    mixedPayload,
			wantErr: nil,
		},
		{
			name:    "ChecksumFailure",
			mode:    cue.ModeMode1_2352,
			raw:     corruptMode1Raw(),
			want:    nil,
			wantErr: track.ErrChecksum,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := disctest.MustOpenTrack(tc.mode, tc.raw)

			// Act
			payload, err := io.ReadAll(sut)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("io.ReadAll(track) = error %v, want %v", got, want)
			}
			if got, want := payload, tc.want; !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
				t.Errorf("io.ReadAll(track) = mismatch (-want +got):\n%s", cmp.Diff(want, got))
			}
		})
	}
}

func TestTrackSeek(t *testing.T) {
	t.Parallel()

	payload := dataBytes(3 * 2048)
	size := int64(len(payload))

	testCases := []struct {
		name    string
		mode    cue.Mode
		raw     []byte
		offset  int64
		whence  int
		wantPos int64
		wantErr error
	}{
		{
			name:    "FromStart",
			mode:    cue.ModeMode1_2048,
			raw:     payload,
			offset:  100,
			whence:  io.SeekStart,
			wantPos: 100,
			wantErr: nil,
		},
		{
			name:    "FromCurrent",
			mode:    cue.ModeMode1_2048,
			raw:     payload,
			offset:  50,
			whence:  io.SeekCurrent,
			wantPos: 50,
			wantErr: nil,
		},
		{
			name:    "FromEnd",
			mode:    cue.ModeMode1_2048,
			raw:     payload,
			offset:  -100,
			whence:  io.SeekEnd,
			wantPos: size - 100,
			wantErr: nil,
		},
		{
			name:    "NegativeResult",
			mode:    cue.ModeMode1_2048,
			raw:     payload,
			offset:  -1,
			whence:  io.SeekStart,
			wantPos: 0,
			wantErr: disc.ErrNegativeOffset,
		},
		{
			name:    "InvalidWhence",
			mode:    cue.ModeMode1_2048,
			raw:     payload,
			offset:  0,
			whence:  99,
			wantPos: 0,
			wantErr: disc.ErrWhence,
		},
		{
			name:    "EndOfCorruptTrack",
			mode:    cue.ModeMode1_2352,
			raw:     corruptMode1Raw(),
			offset:  0,
			whence:  io.SeekEnd,
			wantPos: 0,
			wantErr: track.ErrChecksum,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := disctest.MustOpenTrack(tc.mode, tc.raw)

			// Act
			pos, err := sut.Seek(tc.offset, tc.whence)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Track.Seek(%d, %d) = error %v, want %v", tc.offset, tc.whence, got, want)
			}
			if got, want := pos, tc.wantPos; !cmp.Equal(got, want) {
				t.Errorf("Track.Seek(%d, %d) = %d, want %d", tc.offset, tc.whence, got, want)
			}
		})
	}
}

func TestTrack_SeekThenRead_ReadsFromOffset(t *testing.T) {
	t.Parallel()

	// Arrange
	payload := dataBytes(2 * 2048)
	sut := disctest.MustOpenTrack(cue.ModeMode1_2048, payload)

	// Act
	pos, seekErr := sut.Seek(2048, io.SeekStart)
	rest, readErr := io.ReadAll(sut)

	// Assert
	if got, want := seekErr, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("Track.Seek(2048, SeekStart) = error %v, want %v", got, want)
	}
	if got, want := readErr, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("io.ReadAll(track) = error %v, want %v", got, want)
	}
	if got, want := pos, int64(2048); !cmp.Equal(got, want) {
		t.Errorf("Track.Seek(2048, SeekStart) = %d, want %d", got, want)
	}
	if got, want := rest, payload[2048:]; !cmp.Equal(got, want) {
		t.Errorf("io.ReadAll(track) = mismatch (-want +got):\n%s", cmp.Diff(want, got))
	}
}

func TestTrackWriteTo(t *testing.T) {
	t.Parallel()

	bare := dataBytes(2 * 2048)
	mixedRaw, mixedPayload := mode2MixedRaw()

	testCases := []struct {
		name string
		mode cue.Mode
		raw  []byte
		want []byte
	}{
		{
			name: "BareMode1",
			mode: cue.ModeMode1_2048,
			raw:  bare,
			want: bare,
		},
		{
			name: "XAMixedForms",
			mode: cue.ModeMode2_2352,
			raw:  mixedRaw,
			want: mixedPayload,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := disctest.MustOpenTrack(tc.mode, tc.raw)
			var buf bytes.Buffer

			// Act
			written, err := sut.WriteTo(&buf)

			// Assert
			if got, want := err, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Track.WriteTo(...) = error %v, want %v", got, want)
			}
			if got, want := written, int64(len(tc.want)); !cmp.Equal(got, want) {
				t.Errorf("Track.WriteTo(...) = %d bytes, want %d", got, want)
			}
			if got, want := buf.Bytes(), tc.want; !cmp.Equal(got, want) {
				t.Errorf("Track.WriteTo(...) = mismatch (-want +got):\n%s", cmp.Diff(want, got))
			}
		})
	}
}

func TestTrack_WriteToFailingWriter_ReturnsError(t *testing.T) {
	t.Parallel()

	// Arrange
	writeErr := io.ErrClosedPipe
	sut := disctest.MustOpenTrack(cue.ModeMode1_2048, dataBytes(2048))

	// Act
	written, err := sut.WriteTo(errWriter{err: writeErr})

	// Assert
	if got, want := err, writeErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("Track.WriteTo(errWriter) = error %v, want %v", got, want)
	}
	if got, want := written, int64(0); !cmp.Equal(got, want) {
		t.Errorf("Track.WriteTo(errWriter) = %d bytes, want %d", got, want)
	}
}

func TestTrack_ShortFinalSector_ReturnsShortSectorError(t *testing.T) {
	t.Parallel()

	// Arrange
	data := map[string][]byte{"track.bin": make([]byte, 2048+100)}
	size := map[string][]byte{"track.bin": make([]byte, 2*2048)}
	d := disc.NewDisc(
		mustCue(t, bareTrack),
		disc.WithOpenFunc(disctest.OpenFunc(data)),
		disc.WithStatFunc(disctest.StatFunc(size)),
	)
	sut := openTrack(t, d, 1)

	// Act
	payload, err := io.ReadAll(sut)

	// Assert
	if got, want := err, track.ErrShortSector; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("io.ReadAll(track) = error %v, want %v", got, want)
	}
	if got, want := payload, make([]byte, 2048); !cmp.Equal(got, want) {
		t.Errorf("io.ReadAll(track) = mismatch (-want +got):\n%s", cmp.Diff(want, got))
	}
}

func TestTrack_WriteToCorruptTrack_ReturnsChecksumError(t *testing.T) {
	t.Parallel()

	// Arrange
	sut := disctest.MustOpenTrack(cue.ModeMode1_2352, corruptMode1Raw())
	var buf bytes.Buffer

	// Act
	written, err := sut.WriteTo(&buf)

	// Assert
	if got, want := err, track.ErrChecksum; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("Track.WriteTo(...) = error %v, want %v", got, want)
	}
	if got, want := written, int64(0); !cmp.Equal(got, want) {
		t.Errorf("Track.WriteTo(...) = %d bytes, want %d", got, want)
	}
}
