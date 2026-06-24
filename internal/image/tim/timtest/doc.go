/*
Package timtest assembles TIM image byte streams in memory for tests.

[New] builds a stream from a set of [Option] values: choose the pixel mode, add a
CLUT, and set the pixel block. Options also exist to corrupt a stream — a bad id
word, a wrong byte count, or a truncated tail — so consumers can exercise the
error paths of [tim.Parse] and [tim.Decode].
*/
package timtest
