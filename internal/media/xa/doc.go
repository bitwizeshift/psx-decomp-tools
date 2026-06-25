/*
Package xa decodes CD-ROM XA ADPCM audio into signed 16-bit PCM.

XA audio is carried in the Form 2 sectors of a Mode 2 track: 2304 bytes of ADPCM
per sector, arranged as eighteen 128-byte sound groups, each holding eight sound
units of 28 four-bit samples. A [Coding] byte, taken from the sector subheader,
describes the channel count and sample rate.

A [Decoder] decodes the sectors of a single logical channel in order, carrying the
ADPCM predictor state across sectors. Construct it with [NewDecoder] and feed each
sector's user data to [Decoder.DecodeSector], which returns interleaved PCM. Only
the four-bit coding used by the game's streams is supported; other codings report
[ErrUnsupportedCoding].
*/
package xa
