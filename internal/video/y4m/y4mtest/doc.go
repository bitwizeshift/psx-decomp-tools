/*
Package y4mtest builds synthetic YUV4MPEG2 data for tests.

It constructs solid-colour [image.YCbCr] frames and assembles the stream header
and FRAME records that the y4m package is expected to write, so tests can compare
encoder output against known bytes without a real movie.
*/
package y4mtest
