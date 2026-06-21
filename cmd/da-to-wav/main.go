// Program da-to-wav is a scrappy, one-off utility for converting raw PCM audio
// extracted from a PSX disc image into WAV files.
//
// This is not the permanent tool, but just a working placeholder for a
// purpose-driven conversion.
package main

import (
	"fmt"
	"os"

	"github.com/bitwizeshift/psx-decomp-tools/internal/audio/wav"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: da-to-wav <file.da>")
		os.Exit(2)
	}
	if err := run(os.Args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "da-to-wav: %v\n", err)
		os.Exit(1)
	}
}

// run converts the ".DA" PCM file at path into a sibling ".wav" file. CD-DA audio
// is 16-bit stereo at 44100 Hz.
func run(path string) error {
	pcm, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	out, err := os.Create(path + ".wav")
	if err != nil {
		return err
	}
	format := wav.Format{SampleRate: 44100, Channels: 2, BitsPerSample: 16}
	if err := wav.Encode(out, format, pcm); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
