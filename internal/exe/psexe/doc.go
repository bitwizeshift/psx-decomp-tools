/*
Package psexe reads PlayStation PS-X EXE executables.

A PS-X EXE is a fixed 0x800-byte header followed by a text payload that the BIOS
loads to a fixed address. The header records the entry point, the global and stack
pointers, and the address ranges of the text, data, BSS, and stack regions. This
package reads that header and describes those regions; it does not disassemble or
relocate the code.
*/
package psexe
