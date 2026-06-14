package cue

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
)

// parser assembles a [File] from the lines of a CUE sheet. It tracks the FILE
// context that governs subsequent tracks and the track currently being built.
type parser struct {
	scanner  *bufio.Scanner
	file     *File
	fileName string
	fileType Type
	haveFile bool
	track    *Track
}

// newParser returns a parser that reads the CUE sheet from r.
func newParser(r io.Reader) *parser {
	return &parser{
		scanner: bufio.NewScanner(r),
		file:    &File{},
	}
}

// parse consumes every line of the sheet and returns the assembled [File]. It
// returns one of the package sentinels when a line is malformed, or the reader's
// error if reading fails.
func (p *parser) parse() (*File, error) {
	for p.scanner.Scan() {
		line := strings.TrimSpace(p.scanner.Text())
		if line == "" {
			continue
		}
		if err := p.parseLine(line); err != nil {
			return nil, err
		}
	}
	if err := p.scanner.Err(); err != nil {
		return nil, fmt.Errorf("cue: read: %w", err)
	}
	p.finishTrack()
	return p.file, nil
}

// parseLine dispatches a single non-empty, trimmed line to the handler for its
// command.
func (p *parser) parseLine(line string) error {
	fields, err := p.tokenize(line)
	if err != nil {
		return err
	}
	command, args := strings.ToUpper(fields[0]), fields[1:]
	switch command {
	case "REM":
		p.addComment(strings.TrimSpace(line[len(fields[0]):]))
		return nil
	case "TITLE", "PERFORMER", "SONGWRITER":
		return nil
	case "CATALOG":
		return p.parseCatalog(args)
	case "CDTEXTFILE":
		return p.parseCDTextFile(args)
	case "FILE":
		return p.parseFile(args)
	case "TRACK":
		return p.parseTrack(args)
	case "FLAGS":
		return p.parseFlags(args)
	case "ISRC":
		return p.parseISRC(args)
	case "PREGAP":
		return p.parsePregap(args)
	case "POSTGAP":
		return p.parsePostgap(args)
	case "INDEX":
		return p.parseIndex(args)
	default:
		return fmt.Errorf("%w: unknown command %q", ErrSyntax, command)
	}
}

// addComment records a REM comment against the current track, or the disc when
// no track is open.
func (p *parser) addComment(comment string) {
	if p.track != nil {
		p.track.Comments = append(p.track.Comments, comment)
		return
	}
	p.file.Comments = append(p.file.Comments, comment)
}

// parseCatalog handles a CATALOG command.
func (p *parser) parseCatalog(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("%w: CATALOG expects one argument", ErrSyntax)
	}
	p.file.Catalog = args[0]
	return nil
}

// parseCDTextFile handles a CDTEXTFILE command.
func (p *parser) parseCDTextFile(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("%w: CDTEXTFILE expects one argument", ErrSyntax)
	}
	p.file.CDTextFile = args[0]
	return nil
}

// parseFile handles a FILE command, updating the context applied to subsequent
// tracks.
func (p *parser) parseFile(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("%w: FILE expects a name and type", ErrSyntax)
	}
	var fileType Type
	if err := fileType.UnmarshalText([]byte(args[1])); err != nil {
		return err
	}
	p.fileName, p.fileType, p.haveFile = args[0], fileType, true
	return nil
}

// parseTrack handles a TRACK command, finalizing any open track before starting
// a new one bound to the current FILE context.
func (p *parser) parseTrack(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("%w: TRACK expects a number and mode", ErrSyntax)
	}
	if !p.haveFile {
		return fmt.Errorf("%w: TRACK before FILE", ErrSyntax)
	}
	number, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("%w: invalid track number %q", ErrSyntax, args[0])
	}
	var mode Mode
	if err := mode.UnmarshalText([]byte(args[1])); err != nil {
		return err
	}
	p.finishTrack()
	p.track = &Track{
		Number: number,
		Mode:   mode,
		Type:   p.fileType,
		File:   p.fileName,
	}
	return nil
}

// parseFlags handles a FLAGS command on the current track.
func (p *parser) parseFlags(args []string) error {
	if p.track == nil {
		return fmt.Errorf("%w: FLAGS before TRACK", ErrSyntax)
	}
	if len(args) == 0 {
		return fmt.Errorf("%w: FLAGS expects at least one flag", ErrSyntax)
	}
	flags := make([]Flag, 0, len(args))
	for _, arg := range args {
		var flag Flag
		if err := flag.UnmarshalText([]byte(arg)); err != nil {
			return err
		}
		flags = append(flags, flag)
	}
	p.track.Flags = flags
	return nil
}

// parseISRC handles an ISRC command on the current track.
func (p *parser) parseISRC(args []string) error {
	if p.track == nil {
		return fmt.Errorf("%w: ISRC before TRACK", ErrSyntax)
	}
	if len(args) != 1 {
		return fmt.Errorf("%w: ISRC expects one argument", ErrSyntax)
	}
	p.track.ISRC = args[0]
	return nil
}

// parsePregap handles a PREGAP command on the current track.
func (p *parser) parsePregap(args []string) error {
	if p.track == nil {
		return fmt.Errorf("%w: PREGAP before TRACK", ErrSyntax)
	}
	offset, err := p.timecode(args)
	if err != nil {
		return err
	}
	p.track.Pregap = offset
	return nil
}

// parsePostgap handles a POSTGAP command on the current track.
func (p *parser) parsePostgap(args []string) error {
	if p.track == nil {
		return fmt.Errorf("%w: POSTGAP before TRACK", ErrSyntax)
	}
	offset, err := p.timecode(args)
	if err != nil {
		return err
	}
	p.track.Postgap = offset
	return nil
}

// parseIndex handles an INDEX command on the current track.
func (p *parser) parseIndex(args []string) error {
	if p.track == nil {
		return fmt.Errorf("%w: INDEX before TRACK", ErrSyntax)
	}
	if len(args) != 2 {
		return fmt.Errorf("%w: INDEX expects a number and timecode", ErrSyntax)
	}
	number, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("%w: invalid index number %q", ErrSyntax, args[0])
	}
	var offset MSF
	if err := offset.UnmarshalText([]byte(args[1])); err != nil {
		return err
	}
	p.track.Indices = append(p.track.Indices, Index{Number: number, Offset: offset})
	return nil
}

// timecode parses the single MSF argument shared by PREGAP and POSTGAP.
func (p *parser) timecode(args []string) (*MSF, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%w: expects one timecode", ErrSyntax)
	}
	var offset MSF
	if err := offset.UnmarshalText([]byte(args[0])); err != nil {
		return nil, err
	}
	return &offset, nil
}

// finishTrack appends the in-progress track, if any, to the file and clears it.
func (p *parser) finishTrack() {
	if p.track != nil {
		p.file.Tracks = append(p.file.Tracks, *p.track)
		p.track = nil
	}
}

// tokenize splits a line into whitespace-separated tokens, treating any
// double-quoted span as a single token. It returns [ErrSyntax] if a quote is
// left unterminated.
func (p *parser) tokenize(line string) ([]string, error) {
	var tokens []string
	var token strings.Builder
	inQuote := false
	for _, r := range line {
		switch {
		case r == '"':
			inQuote = !inQuote
		case unicode.IsSpace(r) && !inQuote:
			if token.Len() > 0 {
				tokens = append(tokens, token.String())
				token.Reset()
			}
		default:
			token.WriteRune(r)
		}
	}
	if inQuote {
		return nil, fmt.Errorf("%w: unterminated quote", ErrSyntax)
	}
	if token.Len() > 0 {
		tokens = append(tokens, token.String())
	}
	return tokens, nil
}
