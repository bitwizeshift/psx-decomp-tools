/*
Package vag decodes PlayStation VAG samples into signed 16-bit PCM.

A VAG sample is a stream of SPU ADPCM: 16-byte frames, each carrying a
shift/filter byte, a loop-flag byte, and 28 four-bit samples decoded through a
two-tap predictor whose state carries across frames. The stream occurs in two
shapes: as a standalone file with a big-endian "VAGp" header that records the
sample rate and body length, and headerless inside a VAB bank, where the bank's
size table delimits each sample.

Use [Decode] for a headerless ADPCM body, [DecodeConfig] to read only a "VAGp"
header, and [DecodeSample] to decode a standalone "VAGp" file; these decode the
whole stream in order without interpreting the loop flags. Use [DecodeWithLoop]
when the sustain loop matters, such as for synthesis: it reads the frame flags,
stops at the block marked as the end, and reports the [Loop] region to repeat.
*/
package vag
