/*
Package track decomposes the binary data of a CUE-described track into sectors.

It reads fixed-size sectors from a track's data, decodes each sector's headers
into structured form, optionally verifies its checksum, and extracts the logical
payload. The package operates on [io.ReaderAt] data and knows nothing of the CUE
sheet beyond the [cue.Mode] that governs a track's sector layout.

A typical use reads every sector, selects a decoder for the track's mode, then
extracts each sector's payload:

	reader := track.BinSectorReader{Reader: bin, SectorSize: size}
	sectors, err := track.ReadAllSectors(reader)
	decoder := factory.DecoderFor(t)

Decoders return a best-effort [Sector] alongside any error so that
[LenientSectorDecoder] can tolerate selected failures.
*/
package track
