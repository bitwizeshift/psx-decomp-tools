/*
Package edc computes the CD-ROM error-detection code.

The EDC is the 32-bit CRC appended to data sectors (Mode 1 and XA Form 1/2) that
protects a sector's header and data regions. This package exposes only the raw
computation; callers are responsible for selecting the byte range a given sector
layout protects.
*/
package edc
