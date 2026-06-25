/*
Package sqvtest builds synthetic SQV containers for tests.

It frames MIDI sequences and a VAB bank behind the ".sqv" header, so tests can
exercise the sqv package without a real container. Build the sequences with the
miditest package and the bank with the vabtest package.
*/
package sqvtest
