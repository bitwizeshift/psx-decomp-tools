# `STR` streaming movie format

![badge]

[badge]: <https://img.shields.io/badge/decoded-~95%25-green.svg> "95% reverse engineered"

`STR` is the standard PlayStation streaming movie container. It interleaves
[MDEC][mdec]-compressed video frames with [XA ADPCM][xa] audio so a movie can be
played straight off the disc, one CD sector at a time, without seeking.

In _Brave Fencer Musashi_ ([SLUS-00726]) the dumped `.STR` files are not raw CD
sectors: they hold the concatenated **user data** of each sector, and a sibling
`.xa` file records the per-sector [CD-XA subheaders][subheader]. The subheaders
are needed to walk the stream, because audio and video sectors carry different
amounts of user data.

[mdec]: <./mdec.md>
[xa]: <../audio/vag.md>
[subheader]: <../archive/cd.md>
[SLUS-00726]: <https://psxdatacenter.com/games/U/B/SLUS-00726.html>

## Units

All integers are **little-endian, unsigned**.

## Stream layout

A movie is a run of CD sectors. Each sector's `.xa` subheader classifies it:

| Submode | Form | Size   | Kind                                            |
| ------- | ---- | ------ | ----------------------------------------------- |
| `0x64`  | 2    | `2324` | XA ADPCM audio (`0xe4` marks the final sector)  |
| `0x48`  | 1    | `2048` | MDEC video                                      |
| `0x00`  | 1    | `2048` | filler/padding (see below)                      |

The submode is a bit field: `0x04` flags audio, `0x08` flags video, `0x20` flags
a Form 2 sector, `0x40` flags a real-time stream sector, and `0x80` flags the
final sector of the file (so `0xe4` is the last audio sector).

Audio sectors are demuxed by channel and decoded with the [XA][xa] decoder;
that path already exists as the `str-demux` tool. Not every stream carries video:
`ST01`–`ST06`, `OPEN`, `OUT`, `END`, and `SHOPS01` are **audio-only** streams of
8–16 interleaved channels, padded with filler sectors; their visuals are rendered
by the game engine, not stored. Only `LOGOA`/`LOGOB` are MDEC movies. The video
sectors are the subject of the rest of this document.

## STR sector header (32 bytes)

Every video sector begins with a fixed header, followed by up to `2016` bytes of
the frame's [BS bitstream][mdec].

| Offset | Size | Field          | Notes                                                   |
| ------ | ---- | -------------- | ------------------------------------------------------- |
| `0x00` | 2    | `status`       | `0x0160` for a standard stream                          |
| `0x02` | 2    | `type`         | `0x8001` for MDEC video                                 |
| `0x04` | 2    | `sectorIndex`  | zero-based position of this sector within its frame     |
| `0x06` | 2    | `sectorCount`  | number of sectors the frame spans                       |
| `0x08` | 4    | `frameNumber`  | one-based frame ordinal                                 |
| `0x0c` | 4    | `dataSize`     | length of the frame's bitstream, excluding header/pad   |
| `0x10` | 2    | `width`        | frame width in pixels (e.g. `320`)                      |
| `0x12` | 2    | `height`       | frame height in pixels (e.g. `220`)                     |
| `0x14` | 12   | —              | reserved; copies of the first sector's bitstream header |

`status` and `type` together form the four-byte signature `60 01 01 80` that
identifies a standard STR movie.

## Frame reassembly

A frame's bitstream is spread across `sectorCount` consecutive video sectors,
which may be separated by interleaved audio sectors. The decoder begins a frame
at the sector whose `sectorIndex` is `0`, concatenates the post-header payload of
each sector in `sectorIndex` order, and trims the result to `dataSize` to drop
the trailing zero padding of the final sector. The reassembled bytes are the BS
bitstream the [MDEC decoder][mdec] turns into a picture.

## Filler sectors

The `0x00` sectors are **not media**. Each begins with a zeroed 32-byte STR
header slot followed by a payload that is uninitialized host RAM left over from
the disc-mastering tool: a constant per-file main-RAM pointer at offset `0x20`
(`0x806c52d0` in `ST01`, `0x806f7570` in `OUT`), interspersed with the tool's
data structures and the Windows source paths of the audio it muxed
(`sasi\out_wav\xa\POUT_02.XA`). They carry no `0x0160`/`0x8001` signature, no
`0x3800` bitstream marker, and no game-readable data — they are filler the
mastering inserts to hold the multi-channel XA interleave timing.

The decoder therefore treats them as a third sector kind alongside audio and
video. `str-demux` extracts every audio channel to `<name>.chNN.wav` and writes
the concatenated filler verbatim to `<name>.data.bin` for archival, while the
video path ignores it.

## Output

Decoded frames have no native container. This toolchain encodes them as
[YUV4MPEG2][y4m] (`.y4m`): a single uncompressed file that players and converters
read directly. STR carries no frame rate, so it is supplied externally; PSX FMV
is conventionally 15 frames per second.

[y4m]: <https://wiki.multimedia.cx/index.php/YUV4MPEG2>

## Open questions

- **The reserved 12 bytes** of the sector header, which mirror the leading bytes
  of the first sector's bitstream header.
