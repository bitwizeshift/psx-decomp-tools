/*
Package mdectest builds synthetic BS bitstreams for tests.

It writes the 8-byte BS header and packs bits into the little-endian,
most-significant-first halfwords the bitstream uses, so tests can assemble frames
from individual blocks or raw Huffman codes and exercise the mdec decoder without
a real movie. [Builder.Block] appends a DC-only block, and [Builder.EndOfFrame]
appends the frame terminator.
*/
package mdectest
