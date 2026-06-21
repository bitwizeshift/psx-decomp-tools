# `TIM` image format

![badge]

[badge]: <https://img.shields.io/badge/decoded-~90%25-orange.svg> "90% reverse engineered"

`TIM` is the standard PlayStation texture image. It pairs an optional color
lookup table (CLUT) with a block of packed pixels laid out for upload into the
console's frame buffer.

## Units

All integers are **little-endian, unsigned**. The id word and flag word are
32-bit; block fields are a 32-bit byte count followed by 16-bit fields. Pixel and
CLUT data is a run of 16-bit words.

## File header (8 bytes)

| Offset | Size | Field  | Notes                                          |
| ------ | ---- | ------ | ---------------------------------------------- |
| `0x00` | 4    | `id`   | always `0x00000010`                            |
| `0x04` | 4    | `flag` | bits `0-2` pixel mode, bit `3` CLUT present    |

Bits `4-31` of `flag` are reserved and zero. When bit `3` is set, a CLUT block
follows the header; otherwise the pixel block follows immediately.

## CLUT block

Present only when the CLUT flag is set.

| Offset | Size | Field   | Notes                                       |
| ------ | ---- | ------- | ------------------------------------------- |
| `0x00` | 4    | `bnum`  | block length in bytes, including this header|
| `0x04` | 2    | `x`     | frame-buffer X of the CLUT                  |
| `0x06` | 2    | `y`     | frame-buffer Y of the CLUT                  |
| `0x08` | 2    | `colors`| color entries per palette                   |
| `0x0a` | 2    | `count` | number of palettes                          |
| `0x0c` | …    | entries | `colors * count` 16-bit colors              |

Invariant: `bnum == 12 + colors*count*2`. A `TIM` may carry more than one
palette; a CLUT image is rendered against one chosen palette (palette 0 by
default).

## Pixel block

| Offset | Size | Field  | Notes                                        |
| ------ | ---- | ------ | -------------------------------------------- |
| `0x00` | 4    | `bnum` | block length in bytes, including this header |
| `0x04` | 2    | `x`    | frame-buffer X of the image                  |
| `0x06` | 2    | `y`    | frame-buffer Y of the image                  |
| `0x08` | 2    | `w`    | block width in 16-bit words                  |
| `0x0a` | 2    | `h`    | image height in pixels                       |
| `0x0c` | …    | pixels | `w * h` 16-bit words of packed pixel data    |

Invariant: `bnum == 12 + w*h*2`. The block width is in frame-buffer words, not
pixels; the pixel width depends on the mode.

Because every block carries its own length, a TIM's total size is fully determined
by its header, so TIMs may be stored back-to-back with no separator between them.

## Pixel modes

| Mode | `flag & 7` | Storage             | Pixel width | CLUT |
| ---- | ---------- | ------------------- | ----------- | ---- |
| 4bpp | `0`        | 4-bit CLUT indices  | `w * 4`     | yes  |
| 8bpp | `1`        | 8-bit CLUT indices  | `w * 2`     | yes  |
| 16bpp| `2`        | direct ABGR1555     | `w`         | no   |
| 24bpp| `3`        | direct RGB888       | `w * 2 / 3` | no   |
| mixed| `4`        | reserved            | --           | --    |

For 4bpp, the low nibble of a byte is the left pixel. For 8bpp, one byte is one
index. The 4bpp and 8bpp modes decode to an `image.Paletted`; the 16bpp and 24bpp
modes decode to an `image.NRGBA`.

## Color (ABGR1555)

A 16-bit color packs three 5-bit channels and a semi-transparency bit:

| Bits   | Channel |
| ------ | ------- |
| `0-4`  | red     |
| `5-9`  | green   |
| `10-14`| blue    |
| `15`   | `STP`   |

Each 5-bit channel is expanded to 8 bits by replicating its high bits into the
low ones, so `0x1f` maps to `0xff`: `c8 = c5<<3 | c5>>2`.

A 16-bit value of `0x0000` is treated as **fully transparent**; every other value
is opaque. The `STP` bit, which the console uses to drive semi-transparency
blending, is not otherwise interpreted. 24bpp colors are always opaque.

## Unknowns

- [ ] The semantics of the `STP` bit and the console's semi-transparency blend
      modes, which determine how a non-zero `STP` color should composite.

- [ ] The layout of the mixed mode (`flag & 7 == 4`), which is recognized as a
      valid header but not decoded.
