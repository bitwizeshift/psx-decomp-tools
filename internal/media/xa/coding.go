package xa

// Coding is the audio parameter byte from a Form 2 sector subheader. Its low bits
// select the channel count, sample rate, and sample width of the ADPCM stream.
type Coding byte

// Stereo reports whether the stream carries two interleaved channels rather than
// one.
func (c Coding) Stereo() bool {
	return c&0x03 == 1
}

// Channels returns the number of channels in the stream: two when stereo, one
// otherwise.
func (c Coding) Channels() int {
	if c.Stereo() {
		return 2
	}
	return 1
}

// SampleRate returns the stream's sample rate in Hz.
func (c Coding) SampleRate() int {
	if c&0x0c == 0 {
		return 37800
	}
	return 18900
}

// fourBit reports whether the stream uses four-bit samples, the only width this
// package decodes.
func (c Coding) fourBit() bool {
	return c&0x30 == 0
}
