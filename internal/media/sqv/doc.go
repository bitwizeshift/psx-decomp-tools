/*
Package sqv decodes PlayStation SQV sequenced-music containers.

An SQV file begins with the tag ".sqv" and bundles one or more MIDI-derived
sequences with a single VAB instrument bank, the bank's samples being the
instruments the sequences play. The components are not at fixed offsets: the
sequences are located by their "MThd" tags and the bank by its "pBAV" tag.

Use [Decode] to read a whole container into a [File], exposing the sequences
(see the midi package) and the bank (see the vab package). [Render] synthesizes
one sequence against the bank into PCM that the wav package can encode.
*/
package sqv
