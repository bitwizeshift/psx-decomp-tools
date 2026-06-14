package cue

import (
	"fmt"
	"strings"
)

// enum constrains the integer-backed enumeration types defined in this package.
type enum interface {
	~int
}

// namesToValues inverts a value-to-name table into a name-to-value lookup, used
// to resolve the canonical CUE text of an enumeration back to its value.
func namesToValues[T enum](names map[T]string) map[string]T {
	values := make(map[string]T, len(names))
	for value, name := range names {
		values[name] = value
	}
	return values
}

// enumString renders value as its canonical CUE text from names, falling back
// to a "kind(n)" form for values outside the table.
func enumString[T enum](names map[T]string, value T, kind string) string {
	if name, ok := names[value]; ok {
		return name
	}
	return fmt.Sprintf("%s(%d)", kind, int(value))
}

// enumValue resolves text to its enumeration value via a case-insensitive
// lookup in values. It returns sentinel wrapped with the offending text when no
// value matches.
func enumValue[T enum](values map[string]T, text []byte, sentinel error) (T, error) {
	if value, ok := values[strings.ToUpper(string(text))]; ok {
		return value, nil
	}
	var zero T
	return zero, fmt.Errorf("%w: %q", sentinel, text)
}
