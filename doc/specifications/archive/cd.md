# `.CD` archive format

![badge]

[badge]: <https://img.shields.io/badge/decoded-~80%25-orange.svg> "80% reverse engineered"

`CD` is a **nested container archive** format first discovered in
_Brave Fencer Musashi_ ([SLUS-00726]).

[SLUS-00726]: https://psxdatacenter.com/games/U/B/SLUS-00726.html

`.CD` files are an **uncompressed** container format. It is a flat archive of
sector-aligned members described by a table of contents at the start of the file.

## Units

All multi-byte integers are **little-endian, unsigned 32-bit**. Positions are
expressed in **sectors of 2048 bytes**; a member that begins at sector `S` begins
at byte offset `S * 2048`.

## Layout

```text
+------------------+  byte 0
| header (8 bytes) |
+------------------+  byte 8
| TOC entries      |  count * 8 bytes
| (8 bytes each)   |
+------------------+
| zero padding     |  to the end of sector 0
+------------------+  byte 2048  (sector 1)
| member 0 data    |
| member 1 data    |
| ...              |
+------------------+
```

### Header (8 bytes)

| Offset | Size | Field      | Notes                              |
| ------ | ---- | ---------- | ---------------------------------- |
| `0x00` | 4    | `count`    | number of members in the archive   |
| `0x04` | 4    | `reserved` | always `0` in observed data        |

### TOC entry (8 bytes, repeated `count` times)

| Offset | Size | Field         | Notes                                   |
| ------ | ---- | ------------- | --------------------------------------- |
| `+0x00`| 4    | `startSector` | first 2048-byte sector of the member    |
| `+0x04`| 4    | `byteSize`    | exact member length in bytes (unpadded) |

The table of contents itself occupies sector 0, so the first member begins at
sector 1. Members are laid out contiguously and each is padded up to a sector
boundary, giving the invariant:

```go
startSector[i+1] == startSector[i] + ceil(byteSize[i] / 2048)
```

This chaining holds for every member of a `.CD` archive.

## Example

```text
00000000  31 00 00 00  00 00 00 00   count = 0x31 (49), reserved = 0
00000008  01 00 00 00  00 58 00 00   member 0: sector 1,    size 0x5800
00000010  0c 00 00 00  00 18 01 00   member 1: sector 0x0c, size 0x11800
00000018  40 00 00 00  00 d0 07 00   member 2: sector 0x40, size 0x7d000
...
```

Member 0 starts at sector 1 (byte `0x800`) and is `0x5800` bytes, spanning
`0x5800 / 0x800 = 11` sectors, so member 1 starts at sector `1 + 11 = 12`
(`0x0c`), and so on.

## Member contents

A member's bytes are opaque to the `cd` layer. In practice members are one of:

- a nested [`PAC`][pac] archive (the common case)
- a raw streamed-audio blob beginning with the ASCII tag `.sqv`;
- a length-prefixed build artifact (e.g. a `C:\TIMPACK\…` overlay)

[pac]: <./pac.md>
