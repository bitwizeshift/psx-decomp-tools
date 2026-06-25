/*
Package pac reads the "PAC" archives nested inside the ".CD" containers of the
PSX game Brave Fencer Musashi.

A PAC node begins with a 16-byte header: the ASCII tag "PAC\x00", a kind word, a
header-size word, and a total-size word. A node is either a leaf or a directory.

A leaf has a zero header-size word and a kind word naming the type of the payload
that follows the header. A directory has a non-zero header-size word giving the
number of 2048-byte sectors, less one, that its header occupies; its children are
laid out contiguously after that header, each padded to a 2048-byte boundary, and
fill the node to its end. Directories nest, with a nested directory always
appearing as the final child of its parent. A directory's header region may itself
hold inline data, such as model geometry, ahead of its children.

The meaning of the total-size word varies by kind and is not relied upon. A node's
payload is instead taken to be the whole region the node was allotted by its
parent, after the header, so that every byte of the archive is recoverable.

The package flattens this tree into the payloads it contains, exposed through a
reader modelled on [archive/zip]: open a node with [NewReader], range over
[Reader.File], and stream a payload with [File.Open]. Each [File] carries a
slash-separated index [File.Name] describing its position in the tree; a
directory's own inline data is named "<dir>/data".

Some directories embed children that are not themselves PAC nodes, such as scene
scripts or overlay code. The package cannot size such a child, so it reports the
remainder of its directory as a single raw [File] (see [File.Raw]) rather than
failing, ensuring every byte is recoverable.
*/
package pac
