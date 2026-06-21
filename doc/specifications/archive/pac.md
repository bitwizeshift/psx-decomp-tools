# `PAC` archive format

![badge]

[badge]: <https://img.shields.io/badge/decoded-~65%25-orange.svg> "~65% reverse engineered"

> [!NOTE]
>
> It is distinctly possible that `PAC` is not actually its _own_ format, but
> rather a substructure of [`CD`][cd] itself. **More analysis is needed**.
>
> This spec outlines it independently, because it can act as a self-contained
> format.

`PAC` is a **nested container archive** format first discovered in
_Brave Fencer Musashi_ ([SLUS-00726]).

This page details the format of this file as it has been uncovered.

[SLUS-00726]: <https://psxdatacenter.com/games/U/B/SLUS-00726.html>

A `PAC` archive is a **tree**: directory nodes contain other nodes, and leaf
nodes carry resource payloads (textures, models, sound banks, scene data,
overlay code, etc...).

[cd]: <./cd.md>

## Units

All integers are **little-endian, unsigned 32-bit**. Nodes are aligned and padded
to **2048-byte sectors**.

## Node header (16 bytes)

Every node -- leaf or directory -- begins with:

| Offset | Size | Field    | Notes                                               |
| ------ | ---- | -------- | --------------------------------------------------- |
| `0x00` | 4    | `magic`  | ASCII `"PAC\0"` (`50 41 43 00`)                     |
| `0x04` | 4    | `kind`   | resource type for leaves; `0` for directories       |
| `0x08` | 4    | `header` | `0` for a leaf; sectors−1 of header for a directory |
| `0x0c` | 4    | `total`  | size word; interpretation depends on `kind`         |

The discriminator is the `header` word (equivalently, `kind == 0` ⟺ directory in
all observed data):

- **Leaf** -- `header == 0`. A payload follows the 16-byte header.
- **Directory** -- `header > 0`. Its header occupies `(header + 1)` sectors.

## Directory nodes

```go
dataStart = nodeStart + (header + 1) * 2048
```

The bytes between the 16-byte header and `dataStart` are the directory's **own
inline data region** (often model geometry); this region is sometimes zero
padding and sometimes real content.

Children are laid out contiguously starting at `dataStart`, each padded to a
sector boundary, and **fill the node to its end**. The number of children is *not*
read from a field -- children are walked until the node end is reached. A nested
directory child is always the **last child** of its parent and is given the
remainder of the node.

A child is sized while walking:

- a **leaf** child occupies `ceil(total / 2048)` sectors (see the caveat below);
- a **directory** child (the tail) occupies the rest of the parent.

### Raw children

Some scene archives embed children that are *not* `PAC` nodes; scene scripts and
MIPS overlay code. Their size cannot be derived from a `PAC` header. In such case,
the remainder of the directory is given to the child as a **raw blob**.

## Leaf nodes

> [!NOTE]
>
> This section is not yet fully understood.

A leaf carries its payload immediately after the 16-byte header. The `total` word
is **not** a reliable payload length:

- for kinds `1, 2, 3, 8, 0x103` it equals the node's total size (header + payload),
  so `ceil(total / 2048)` correctly advances to the next sibling;
- for kind `4` (the most common) `total` is **smaller** than the node's real
  extent -- e.g. a `0x10b000`-byte member declares `total = 0x6ee3d`.

Because of this, a leaf's payload is interpreted as the whole region the node
was allotted by its parent (or, for a top-level member, the whole `.CD` member),
after the 16-byte header -- never `total`.

### Resource sequences

A leaf payload is not always a single resource. Some payloads begin with a region
of zero padding and then carry a sequence of **self-delimiting** resources stored
back-to-back, with no count or offset table; the run is recovered by reading each
resource's own header to find where the next begins. The size and purpose of the
leading region, and which leaves use this layout, are not yet established.

## Observed leaf kinds

Counts across `MAIN.CD` + `SC01–SC07.CD`:

| `kind`  | count | notes                                              |
| ------- | ----- | -------------------------------------------------- |
| `0x001` | 86    | `total` = node size                                |
| `0x002` | 42    | `total` = node size                                |
| `0x003` | 4     | `total` = node size                                |
| `0x004` | 138   | most common; `total` < node size (extent is larger)|
| `0x008` | 14    | `total` = node size; frequently `0x5800` blocks    |
| `0x100` | --     | seen as a directory `kind` in scene archives       |
| `0x101` | 52    | large blobs                                        |
| `0x103` | 32    | `total` = node size                                |
| `0x107` | 4     | --                                                  |

The semantic meaning of each `kind` (which is a TIM, a model, a CLUT, etc.) is
not yet established.

## Extraction

Readers should flattens the tree into a list of payloads, each named by its
slash-separated index path within the tree, plus a directory's inline data named
`<dir>/data`. For example a `MAIN.CD` member might yield:

```
000/0.bin          leaf at child index 0
013/2/0.bin        leaf at 13 → 2 → 0
005/data.bin       inline data region of directory 005 (model geometry)
SC03 …/N.bin       raw remainder of a scene directory (File.Raw)
```

## Unknowns

- [ ] The meaning of the `kind` word, and a magic-based sniff for each leaf
      type.

- [ ] The meaning of the `total` word for kind `4` (decompressed size? element
      count?) and whether kind-`4` leaves are themselves sub-containers worth
      splitting.

- [ ] The size and purpose of the leading zero region that precedes a resource
      sequence in some leaves, and what marks a leaf as carrying a sequence
      rather than a single resource.

- [ ] The structure of the scene scripts carried as raw children -- this is also
      where the game's (custom-encoded) dialog text is expected to live.
