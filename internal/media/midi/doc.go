/*
Package midi decodes the MIDI-derived sequence data embedded in PlayStation SQV
music into structured sequences of events.

The data uses Standard MIDI File framing: an "MThd" header chunk followed by one
or more "MTrk" track chunks, each a stream of events prefixed by variable-length
delta times, with running status. It deviates from Standard MIDI in two ways that
this package accounts for: a note-off message and a pitch-wheel message each
carry a single data byte rather than two, and a track may omit the end-of-track
event, ending instead at the chunk's content with the chunk length's trailing
zero padding ignored.

Some tracks are genuinely truncated, their bytes stopping in the middle of a
final event. [Decode] reports this with [ErrTruncated] but still returns the
events it read, so a caller may use the recovered sequence or reject it.

Use [DecodeConfig] to read only the header, and [Decode] to read a whole file
into a [Sequence]. Events decode to concrete [Message] values; messages the
decoder does not model are preserved as [Unknown] so no track byte is lost.
*/
package midi
