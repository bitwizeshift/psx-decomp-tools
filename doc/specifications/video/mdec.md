# `MDEC` / `BS` bitstream format

![badge]

[badge]: <https://img.shields.io/badge/decoded-~90%25-green.svg> "90% implemented"

`BS` ("bitstream") is the JPEG-style compression the PlayStation MDEC hardware
decompresses into a picture. It is the codec inside each [`STR`][str] video frame
and of standalone `.BS` pictures. A frame is a header followed by a Huffman- and
run-length-coded set of DCT coefficients for a grid of 16×16 macroblocks.

This page describes the common **BS version 1 and 2** streams used by _Brave
Fencer Musashi_'s `LOGOA`/`LOGOB.STR`. Encrypted, chunk-based, version 3 (Huffman
DC), and `iki`/`ea`/`v0` variants exist but are out of scope here. The hardware
register-level reference is in the architecture notes; this page documents only
what a software decoder needs.

[str]: <./str.md>

## Units

All integers are **little-endian, unsigned**. The bitstream that follows the
header is a sequence of 16-bit little-endian halfwords whose bits are consumed
**most-significant first** (bit 15 down to bit 0).

## BS header (8 bytes)

| Offset | Size | Field     | Notes                                              |
| ------ | ---- | --------- | -------------------------------------------------- |
| `0x00` | 2    | `size`    | decompressed size in 4-byte units (decoder ignores)|
| `0x02` | 2    | `id`      | file ID, always `0x3800`                           |
| `0x04` | 2    | `quant`   | quantization scale applied to every block          |
| `0x06` | 2    | `version` | bitstream version (`1` or `2`)                     |

The frame has **no width or height**; those come from the enclosing
[STR sector header][str]. Dimensions are rounded up to a multiple of 16 to form
the macroblock grid, then the decoded picture is cropped back down.

## Macroblocks and blocks

Macroblocks are stored **column-major**: top to bottom down the leftmost 16-pixel
column, then the next column. Each macroblock holds six 8×8 blocks in order:

1. `Cr` — chroma, covers the whole 16×16 (4:2:0)
2. `Cb` — chroma, covers the whole 16×16
3. `Y1` `Y2` `Y3` `Y4` — luminance: top-left, top-right, bottom-left, bottom-right

Because the layout is one 8×8 chroma pair per four 8×8 luma blocks, a decoded
frame maps directly onto a 4:2:0 `YCbCr` image with no resampling.

## Block coding

Each block is a 10-bit DC value, then variable-length AC codes, then end of block:

- **DC (10 bits).** A signed 10-bit value. The reserved value `+0x1FF`
  (`0111111111`) is the **end-of-frame** marker, sent in place of the next
  block's DC. Otherwise the block's first coefficient is `DC × quantTable[0]`.
- **AC (Huffman).** Each code expands to an MDEC word holding a zero **run** in
  bits 15-10 and a signed **level** in bits 9-0, plus a trailing sign bit that
  negates the level. A 6-bit escape prefix `000001` is followed by a raw 16-bit
  MDEC word. The AC coefficient at scan position `k` is
  `(level × quantTable[k] × quant + 4) / 8`, saturated to signed 11 bits.
- **EOB (`10`).** Ends the block; remaining coefficients stay zero.

The full AC Huffman table, the quantization table, and the zig-zag order are in
the architecture notes' "BS Compression AC Values" and "MDEC Decompression"
sections.

## Reconstruction

Coefficients are de-zig-zagged into an 8×8 block, an inverse DCT is applied, and
the 8×8 spatial samples are written to the picture. Luminance and chrominance are
both centred: a decoded sample `s` becomes the byte `clamp(s + 128, 0, 255)`,
giving a full-range (`C420jpeg`) `YCbCr` picture that converts to RGB with the
standard JFIF coefficients.

## Remaining Work

- [ ] **Version 3 and variants.** v3 Huffman-codes the DC values (with distinct
      tables for Cr/Cb and Y), and the `iki`, `ea`, and `v0` streams use
      different headers or code tables. None are implemented; detection
      would key off the BS `version` word and the enclosing container.

- [ ] **IDCT precision.** The hardware uses a fixed-point IDCT with a scale
      table; the decoder here uses a floating-point IDCT, which matches visually
      but may differ by a unit in the last place from real hardware output.
