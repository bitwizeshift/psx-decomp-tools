/*
Package subheader reads the per-sector CD-XA subheaders that dump-image records in
the ".xa" sidecar beside an extracted track or stream.

Sidecars store the 4-byte subheader of every logical sector, in order: the file
and channel numbers, the submode flags, and the coding byte. The submode flags
identify a sector's form and whether it carries audio, which together with the form
give the number of user-data bytes the sector occupies in the extracted file.
*/
package subheader
