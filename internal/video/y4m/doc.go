/*
Package y4m encodes video frames into the YUV4MPEG2 stream format.

A YUV4MPEG2 stream is a single uncompressed file: an ASCII stream header naming
the frame size and rate, followed by one "FRAME" record per picture holding raw
planar YCbCr 4:2:0 samples. It carries no compression of its own, so it is a
convenient interchange target that players and converters such as ffmpeg read
directly.

Use [NewEncoder] to begin a stream for a [Format], then [Encoder.WriteFrame] for
each picture. A frame given as an [image.YCbCr] with 4:2:0 subsampling is written
without conversion, matching the output of the mdec package; any other image is
converted to YCbCr first.
*/
package y4m
