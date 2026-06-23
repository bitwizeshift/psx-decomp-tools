# `PS-X EXE` executable format

![badge]

[badge]: <https://img.shields.io/badge/decoded-~90%25-green.svg> "90% reverse engineered"

`PS-X EXE` is the **standard PlayStation executable** format. The BIOS loads it
from disc, copies its payload to a fixed address in RAM, and jumps to its entry
point. It is the main program of nearly every PS1 title, named with a region code
such as `SLUS_007.26`.

A `PS-X EXE` is **uncompressed**: a fixed 2048-byte header followed by the text
payload that the BIOS copies into memory unchanged.

## Units

All multi-byte integers are **little-endian, unsigned 32-bit**, matching the
PlayStation's MIPS R3000 core. Address and size fields are byte values in the
console's address space. The header is exactly `0x800` (2048) bytes, the size of
one CD sector.

## Layout

```text
+-------------------+  byte 0
| header (0x800)    |
+-------------------+  byte 0x800
| text payload      |  t_size bytes
| (loaded to t_addr)|
+-------------------+
```

The total file size is `0x800 + t_size`.

### Header (0x800 bytes)

| Offset | Size | Field      | Notes                                            |
| ------ | ---- | ---------- | ------------------------------------------------ |
| `0x00` | 8    | `magic`    | ASCII `"PS-X EXE"` (`50 53 2d 58 20 45 58 45`)   |
| `0x08` | 8    | —          | reserved, zero-filled                            |
| `0x10` | 4    | `pc0`      | initial program counter (entry point)            |
| `0x14` | 4    | `gp0`      | initial global pointer (register `r28`)          |
| `0x18` | 4    | `t_addr`   | address the text payload is loaded to            |
| `0x1c` | 4    | `t_size`   | text payload length in bytes                     |
| `0x20` | 4    | `d_addr`   | data region address (usually `0`)                |
| `0x24` | 4    | `d_size`   | data region size (usually `0`)                   |
| `0x28` | 4    | `b_addr`   | BSS region address                               |
| `0x2c` | 4    | `b_size`   | BSS region size                                  |
| `0x30` | 4    | `s_addr`   | initial stack and frame pointer base             |
| `0x34` | 4    | `s_size`   | offset added to `s_addr` for the initial stack   |
| `0x4c` | …    | `marker`   | ASCII region string, null-padded to `0x800`      |

The text payload begins at `0x800` and is `t_size` bytes long. `t_size` is a whole
number of sectors.

## Regions

The header names four regions, of which only the text region has bytes in the file:

- **text** — the code and read-only data, copied verbatim from offset `0x800` to
  `t_addr`.
- **data** — a separate initialized-data region. In practice `d_addr` and `d_size`
  are zero; a program's data is carried within the text payload.
- **BSS** — a zero-initialized region at `b_addr` of `b_size` bytes, cleared by the
  loader. It has no bytes in the file.
- **stack** — established at `s_addr + s_size`. It has no bytes in the file.

The BIOS does not validate the `marker` string; a program boots without it.

## Example

```text
00000000  50 53 2d 58 20 45 58 45   magic = "PS-X EXE"
00000008  00 00 00 00 00 00 00 00   reserved
00000010  00 00 01 80               pc0    = 0x80010000
00000014  00 00 00 00               gp0    = 0
00000018  00 08 01 80               t_addr = 0x80010800
0000001c  00 b0 06 00               t_size = 0x0006b000
00000020  00 00 00 00 00 00 00 00   d_addr = 0, d_size = 0
00000028  00 00 00 00 00 00 00 00   b_addr = 0, b_size = 0
00000030  f0 ff 1f 80 00 00 00 00   s_addr = 0x801ffff0, s_size = 0
...
0000004c  53 6f 6e 79 ...           marker = "Sony Computer Entertainment Inc. ..."
```

The text payload occupies sectors `1` through `0xd6` (`0x6b000 / 0x800 = 214`
sectors) and is loaded to `0x80010800`, leaving the entry point `pc0` at
`0x80010000`.
