/*
Package vab decodes PlayStation VAB instrument banks.

A VAB bank ("pBAV") pairs a set of instrument programs with a pool of VAG
samples. Its header is followed by a fixed program-attribute table, a
tone-attribute table that maps note ranges to samples, a size table that
delimits each sample, and the concatenated SPU ADPCM waveform data.

Use [Decode] to parse a whole bank, or [DecodeConfig] to read only the header.
The decoded [Bank] exposes its programs and tones, and [Bank.Waveform] returns a
sample's raw SPU ADPCM bytes, which the vag package decodes to PCM.
*/
package vab
