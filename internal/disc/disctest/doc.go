/*
Package disctest provides in-memory test doubles and factories for package
disc.

It lets consumers build [disc.Disc] and [disc.Track] values backed by data held
in memory rather than files on disk: [OpenFunc] and [StatFunc] serve named
buffers, the Err and Sequential variants drive disc's error paths, and [Disc]
and [OpenTrack] assemble ready-to-read discs and tracks from CUE text and raw
sector data.
*/
package disctest
