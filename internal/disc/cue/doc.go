/*
Package cue parses the CUE sheets that describe the track layout of a CD-ROM
image.

Parsing is the package's only responsibility: it turns the text of a sheet into
a [File] of [Track] values and has no knowledge of the binary data files the
sheet references. Use [FromFile] to parse a sheet on disk or [FromReader] to
parse one from an arbitrary source.

A track flattens the governing FILE declaration together with its own TRACK
declaration and sub-commands, so each [Track] carries its source file, layout,
indices, and timing. Positions are modelled by [MSF] timecodes, and FILE types,
TRACK modes, and FLAGS values are modelled by the [Type], [Mode], and [Flag]
enumerations. Unrecognized values for those enumerations are reported as errors;
commands that carry no information for downstream consumers are accepted but not
retained.
*/
package cue
