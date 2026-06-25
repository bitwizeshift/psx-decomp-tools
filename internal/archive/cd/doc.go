/*
Package cd reads the ".CD" archives used by the PSX game Brave Fencer Musashi.

A ".CD" archive is a flat container: a one-sector table of contents followed by a
run of 2048-byte sector-aligned member files. The table of contents records, for
each member, its starting sector and exact byte size; members are laid out
contiguously so that each begins where the previous one's sectors end.

The package exposes a random-access reader modelled on [archive/zip]: open an
archive with [NewReader] or [OpenReader], range over [Reader.File], and stream a
member's bytes with [File.Open]. It only reads archives; it does not write them,
and it does not interpret member contents.
*/
package cd
