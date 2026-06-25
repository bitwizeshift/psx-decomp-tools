/*
Package avi muxes compressed video frames and PCM audio into an AVI file.

AVI is a RIFF container: a header list describing one video and one audio stream,
a movi list of interleaved frame and audio chunks, and a trailing index. The
per-frame video codec is pluggable through a [FrameEncoder], so frames can be
stored losslessly as PNG or smaller as MJPEG without changing the muxer; the
audio is uncompressed little-endian PCM.

Use [NewWriter] over an [io.WriteSeeker] with the stream formats and an encoder,
then [Writer.WriteVideo] and [Writer.WriteAudio] in playback order, and finally
[Writer.Close], which writes the index and patches the header totals that are
known only once every chunk has been written.
*/
package avi
