package psexe_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/bitwizeshift/psx-decomp-tools/internal/exe/psexe"
	"github.com/bitwizeshift/psx-decomp-tools/internal/exe/psexe/psexetest"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

const region = "Sony Computer Entertainment Inc. for North America area"

// fullHeader returns the options and matching [psexe.Header] for an executable
// that sets every header field.
func fullHeader(text []byte) ([]psexetest.Option, psexe.Header) {
	opts := []psexetest.Option{
		psexetest.PC0(0x80010000),
		psexetest.GP0(0x12345678),
		psexetest.TextAddr(0x80010800),
		psexetest.Text(text),
		psexetest.Data(0x80020000, 0x10),
		psexetest.BSS(0x80030000, 0x20),
		psexetest.Stack(0x801ffff0, 0),
		psexetest.Region(region),
	}
	header := psexe.Header{
		PC0:       0x80010000,
		GP0:       0x12345678,
		TextAddr:  0x80010800,
		TextSize:  uint32(len(text)),
		DataAddr:  0x80020000,
		DataSize:  0x10,
		BSSAddr:   0x80030000,
		BSSSize:   0x20,
		StackAddr: 0x801ffff0,
		StackSize: 0,
		Region:    region,
	}
	return opts, header
}

func TestDecodeConfig(t *testing.T) {
	t.Parallel()

	fullOpts, fullWant := fullHeader([]byte{1, 2, 3, 4})

	testCases := []struct {
		name    string
		reader  io.Reader
		want    psexe.Header
		wantErr error
	}{
		{
			name:   "Full",
			reader: bytes.NewReader(psexetest.New(fullOpts...)),
			want:   fullWant,
		}, {
			name:   "EmptyText",
			reader: bytes.NewReader(psexetest.New()),
			want:   psexe.Header{},
		}, {
			name:   "RegionPaddingTrimmed",
			reader: bytes.NewReader(psexetest.New(psexetest.Region("hi"))),
			want:   psexe.Header{Region: "hi"},
		}, {
			name:    "BadMagic",
			reader:  bytes.NewReader(psexetest.New(psexetest.BadMagic())),
			wantErr: psexe.ErrInvalidHeader,
		}, {
			name:    "TruncatedHeader",
			reader:  bytes.NewReader(psexetest.New(psexetest.Truncate(100))),
			wantErr: psexe.ErrInvalidHeader,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			reader := tc.reader

			// Act
			header, err := psexe.DecodeConfig(reader)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("DecodeConfig() error = %v, want %v", got, want)
			}
			if got, want := header, tc.want; !cmp.Equal(got, want) {
				t.Errorf("DecodeConfig() header diff (-got +want):\n%s", cmp.Diff(got, want))
			}
		})
	}
}

func TestDecode(t *testing.T) {
	t.Parallel()

	payload := []byte{1, 2, 3, 4}
	fullOpts, fullWant := fullHeader(payload)

	testCases := []struct {
		name     string
		reader   io.Reader
		want     *psexe.File
		wantText []byte
		wantErr  error
	}{
		{
			name:     "Full",
			reader:   bytes.NewReader(psexetest.New(fullOpts...)),
			want:     &psexe.File{Header: fullWant},
			wantText: payload,
		}, {
			name: "BadMagic",
			reader: bytes.NewReader(
				psexetest.New(psexetest.BadMagic()),
			),
			wantErr: psexe.ErrInvalidHeader,
		}, {
			name: "TruncatedText",
			reader: bytes.NewReader(psexetest.New(
				psexetest.Text(payload),
				psexetest.Truncate(psexe.HeaderSize+2)),
			),
			wantErr: psexe.ErrTruncated,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			reader := tc.reader

			// Act
			file, err := psexe.Decode(reader)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Decode() error = %v, want %v", got, want)
			}
			opts := cmp.Options{
				cmpopts.IgnoreUnexported(psexe.File{}),
				cmpopts.EquateEmpty(),
			}
			if got, want := file, tc.want; !cmp.Equal(got, want, opts) {
				t.Errorf("Decode() file diff (-got +want):\n%s", cmp.Diff(got, want, opts))
			}
			if got, want := textOf(t, file), tc.wantText; !cmp.Equal(got, want, opts) {
				t.Errorf("Decode() text diff (-got +want):\n%s", cmp.Diff(got, want, opts))
			}
		})
	}
}

func TestFileSections(t *testing.T) {
	t.Parallel()

	opts, _ := fullHeader([]byte{1, 2, 3, 4})

	// Arrange
	file := mustDecode(t, psexetest.New(opts...))
	want := []psexe.Section{
		{Kind: psexe.Text, Addr: 0x80010800, Size: 4, Offset: 0x800},
		{Kind: psexe.Data, Addr: 0x80020000, Size: 0x10, Offset: -1},
		{Kind: psexe.BSS, Addr: 0x80030000, Size: 0x20, Offset: -1},
		{Kind: psexe.Stack, Addr: 0x801ffff0, Size: 0, Offset: -1},
	}

	// Act
	sections := file.Sections()

	// Assert
	if got, want := sections, want; !cmp.Equal(got, want) {
		t.Errorf("Sections() diff (-got +want):\n%s", cmp.Diff(got, want))
	}
}

func TestFileSection(t *testing.T) {
	t.Parallel()

	payload := []byte{1, 2, 3, 4}

	testCases := []struct {
		name     string
		kind     psexe.SectionKind
		wantText []byte
		wantOK   bool
	}{
		{
			name:     "Text",
			kind:     psexe.Text,
			wantText: payload,
			wantOK:   true,
		}, {
			name:   "Data",
			kind:   psexe.Data,
			wantOK: false,
		}, {
			name:   "BSS",
			kind:   psexe.BSS,
			wantOK: false,
		}, {
			name:   "Stack",
			kind:   psexe.Stack,
			wantOK: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			file := mustDecode(t, psexetest.New(psexetest.Text(payload)))

			// Act
			reader, ok := file.Section(tc.kind)

			// Assert
			if got, want := ok, tc.wantOK; got != want {
				t.Fatalf("Section() ok = %v, want %v", got, want)
			}
			if got, want := readerBytes(t, reader), tc.wantText; !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
				t.Errorf("Section() text diff (-got +want):\n%s", cmp.Diff(got, want, cmpopts.EquateEmpty()))
			}
		})
	}
}

// mustDecode decodes img into a [psexe.File], failing the test on error.
func mustDecode(t *testing.T, img []byte) *psexe.File {
	t.Helper()
	file, err := psexe.Decode(bytes.NewReader(img))
	if err != nil {
		t.Fatalf("Decode() unexpected error: %v", err)
	}
	return file
}

// textOf reads the text payload of file, returning nil for a nil file.
func textOf(t *testing.T, file *psexe.File) []byte {
	t.Helper()
	if file == nil {
		return nil
	}
	return readerBytes(t, file.Text())
}

// readerBytes reads all of r, returning nil for a nil reader.
func readerBytes(t *testing.T, r io.Reader) []byte {
	t.Helper()
	if r == nil {
		return nil
	}
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return data
}
