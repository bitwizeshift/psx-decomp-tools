/*
Package vagtest builds synthetic VAG data for tests.

It assembles SPU ADPCM frames from their filter, shift, and sample nibbles, and
wraps a body in a standalone "VAGp" header, so tests can exercise the vag package
without real sample data.
*/
package vagtest
