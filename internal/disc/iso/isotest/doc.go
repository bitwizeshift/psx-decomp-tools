/*
Package isotest assembles ISO 9660 filesystem images in memory for tests.

A [Builder] lays out a directory tree, system area, path tables, and a primary
volume descriptor into a single byte slice that [github.com/bitwizeshift/psx-decomp-tools/internal/disc/iso.New]
can read through a [bytes.Reader]. Helpers are also provided for constructing the
deliberately malformed images used to exercise error handling.
*/
package isotest
