/*
Package wav writes uncompressed PCM audio as canonical RIFF/WAVE files.

It is a small writer shared by the project's audio tools. Describe the audio with
a [Format] and write a buffer of interleaved little-endian PCM samples with
[Encode]; the package emits the 44-byte canonical header followed by the samples.
It does not read WAVE files or encode any compressed format.
*/
package wav
