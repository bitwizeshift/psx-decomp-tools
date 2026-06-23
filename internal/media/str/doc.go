/*
Package str demuxes PlayStation STR streaming movies into their video frames.

An STR movie is stored as the user data of consecutive CD sectors, each
classified by the CD-XA subheader kept in a sidecar (see the subheader package).
The stream interleaves XA ADPCM audio sectors with MDEC video sectors; a video
sector begins with a 32-byte STR header, and a frame's BS bitstream spans one or
more such sectors. Audio is decoded separately with the xa package.

Use [NewDemuxer] with the stream and its subheaders, then call [Demuxer.Next]
for the interleaved video frames and audio packets in stream order, or
[Demuxer.NextFrame] for the video frames alone. Decode a returned [Frame] into a
picture with the mdec package, and an [AudioPacket] into PCM with the xa package.
*/
package str
