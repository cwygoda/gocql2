package gocql2

import (
	"fmt"
	"strings"
)

// Location identifies either a text position, a JSON semantic path, or both.
type Location struct {
	JSONPath   JSONPath
	ByteOffset int
	CharOffset int
	Line       int
	Column     int
}

// NoLocation returns an unavailable location marker.
func NoLocation() Location {
	return Location{ByteOffset: -1, CharOffset: -1}
}

// JSONPathRoot returns the root JSON path: $.
func JSONPathRoot() JSONPath { return nil }

// JSONPath is a path to a JSON value.
type JSONPath []PathElement

// PathElement is one object key or array index step in a JSONPath.
type PathElement struct {
	Index *int
	Key   string
}

// Key appends an object-key path element.
func (p JSONPath) Key(key string) JSONPath {
	out := append(JSONPath{}, p...)
	out = append(out, PathElement{Key: key})
	return out
}

// Index appends an array-index path element.
func (p JSONPath) Index(index int) JSONPath {
	out := append(JSONPath{}, p...)
	out = append(out, PathElement{Index: &index})
	return out
}

func (p JSONPath) String() string {
	var b strings.Builder
	b.WriteByte('$')
	for _, elem := range p {
		if elem.Index != nil {
			fmt.Fprintf(&b, "[%d]", *elem.Index)
			continue
		}
		if isSimpleJSONPathKey(elem.Key) {
			b.WriteByte('.')
			b.WriteString(elem.Key)
			continue
		}
		fmt.Fprintf(&b, "[%q]", elem.Key)
	}
	return b.String()
}

func isSimpleJSONPathKey(key string) bool {
	if key == "" {
		return false
	}
	for i, r := range key {
		if r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || i > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}

// ParseError is returned for CQL2 syntax and semantic parse failures.
//
//nolint:govet // Public field order is grouped for API readability rather than fieldalignment.
type ParseError struct {
	Cause    error
	Message  string
	Source   Language
	Location Location
	Expected []string
}

func (e *ParseError) Error() string {
	if e == nil {
		return "<nil>"
	}

	var b strings.Builder
	if e.Source != "" {
		b.WriteString(string(e.Source))
		b.WriteByte(' ')
	}
	b.WriteString("parse error")

	if e.Location.JSONPath != nil || e.Source == LanguageJSON && e.Location.ByteOffset < 0 && e.Location.Line == 0 {
		b.WriteString(" at ")
		b.WriteString(e.Location.JSONPath.String())
	} else if e.Location.Line > 0 && e.Location.Column > 0 {
		fmt.Fprintf(&b, " at line %d, column %d", e.Location.Line, e.Location.Column)
	} else if e.Location.ByteOffset >= 0 {
		fmt.Fprintf(&b, " at byte %d", e.Location.ByteOffset)
	}

	if e.Message != "" {
		b.WriteString(": ")
		b.WriteString(e.Message)
	}
	if len(e.Expected) > 0 {
		b.WriteString("; expected ")
		b.WriteString(strings.Join(e.Expected, ", "))
	}
	return b.String()
}

func (e *ParseError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func parseError(source Language, loc Location, message string, expected ...string) *ParseError {
	return &ParseError{Source: source, Location: loc, Message: message, Expected: expected}
}
