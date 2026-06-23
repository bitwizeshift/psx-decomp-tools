/*
Package mdec decodes PlayStation BS bitstreams into pictures.

BS ("bitstream") is the Huffman-and-run-length compression the PSX MDEC hardware
inflates: a frame is a header followed by Huffman-coded DC and AC coefficients
for a grid of 16x16 macroblocks, each holding a low-resolution Cr and Cb block
and four luminance blocks. Dequantizing, inverse-transforming, and assembling
those blocks yields a 4:2:0 picture, which is why the decoded frame maps directly
onto an [image.YCbCr].

Use [Decode] with a frame's bitstream and its pixel dimensions, which come from
the enclosing STR sector header (see the str package) since the bitstream records
none. This package decodes the common BS version 1 and 2 streams; encrypted,
chunked, and version 3 variants are not handled.
*/
package mdec
