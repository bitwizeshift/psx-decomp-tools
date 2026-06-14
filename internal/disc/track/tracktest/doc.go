/*
Package tracktest provides test doubles and fixtures for exercising the track
package.

It offers [track.SectorReader] and [track.SectorVerifier] doubles with fixed
behavior, along with builders that assemble valid raw CD-ROM sectors in memory
so that consumers can test decoding and extraction without real disc images.
*/
package tracktest
