/*
Package tim decodes PlayStation TIM texture images into Go's [image] model.

A TIM file pairs an optional color lookup table (CLUT) with a block of packed
pixels in one of four storage modes. Decode a stream with [Parse] to inspect the
full structure — every palette, the framebuffer coordinates, and the pixel
mode — or use [Decode] and [DecodeConfig], which satisfy [image.RegisterFormat]
and are registered under the name "tim" so [image.Decode] recognizes the format.

CLUT modes resolve to an [image.Paletted]; the direct color modes resolve to an
[image.NRGBA]. A 16-bit pixel of zero is treated as fully transparent.
*/
package tim
