/*
Package strtest builds synthetic STR streams for tests.

It assembles sectors into a stream and the matching CD-XA subheaders, so tests
can drive the str package without a real movie. [Builder.Video] lays a frame's
bitstream across as many sectors as it needs, [Builder.Audio] inserts an
interleaved audio sector, and [Builder.Sector] writes one sector with an explicit
header for malformed cases.
*/
package strtest
