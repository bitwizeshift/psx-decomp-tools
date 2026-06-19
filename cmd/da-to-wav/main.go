// Program da-to-wav is a scrappy, one-off utility for converting extract
// raw PCM audio from a PSX disc image into WAV files.
//
// This is not the permanent tool, but just working placeholder for a
// purpose-driven
package main

import (
	"encoding/binary"

	"os"
)

func main() {
	file := os.Args[1]
	pcm, err := os.ReadFile(file)
	if err != nil {
		panic(err)
	}

	wav, err := os.Create(file + ".wav")
	if err != nil {
		panic(err)
	}

	defer wav.Close()

	const (
		sampleRate    = 32000
		bitsPerSample = 16
		channels      = 2
	)

	dataSize := uint32(len(pcm))
	byteRate := uint32(sampleRate * channels * bitsPerSample / 8)
	blockAlign := uint16(channels * bitsPerSample / 8)

	wav.Write([]byte("RIFF"))
	binary.Write(wav, binary.LittleEndian, uint32(36)+dataSize)

	wav.Write([]byte("WAVE"))
	wav.Write([]byte("fmt "))

	binary.Write(wav, binary.LittleEndian, uint32(16))
	binary.Write(wav, binary.LittleEndian, uint16(1)) // PCM
	binary.Write(wav, binary.LittleEndian, uint16(channels))
	binary.Write(wav, binary.LittleEndian, uint32(sampleRate))
	binary.Write(wav, binary.LittleEndian, byteRate)
	binary.Write(wav, binary.LittleEndian, blockAlign)
	binary.Write(wav, binary.LittleEndian, uint16(bitsPerSample))
	wav.Write([]byte("data"))
	binary.Write(wav, binary.LittleEndian, dataSize)
	wav.Write(pcm)
}
