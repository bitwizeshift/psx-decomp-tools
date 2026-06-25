/*
Package miditest builds synthetic Standard MIDI File data for tests.

It assembles raw message bytes and wraps timed events in MTrk track chunks and an
MThd header, so tests can exercise the midi package without real song files. The
message helpers return the bytes of a single message; [Track] and [File] frame
them into a complete file.
*/
package miditest
