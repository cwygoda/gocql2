package parser

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/cwygoda/cql2/api"
)

type tokenKind int

const (
	tokenEOF tokenKind = iota
	tokenIdentifier
	tokenQuotedIdentifier
	tokenKeyword
	tokenString
	tokenNumber
	tokenOperator
	tokenLParen
	tokenRParen
	tokenComma
)

type token struct {
	text string
	span api.Span
	kind tokenKind
}

type lexer struct {
	input        string
	byteOffset   int
	charOffset   int
	line         int
	column       int
	previousKind tokenKind
	hasPrevious  bool
}

var reservedTextKeywords = map[string]struct{}{
	"AND": {}, "OR": {}, "NOT": {}, "LIKE": {}, "BETWEEN": {}, "IN": {}, "IS": {}, "NULL": {}, "TRUE": {}, "FALSE": {},
	"CASEI": {}, "ACCENTI": {}, "DIV": {}, "DATE": {}, "TIMESTAMP": {}, "INTERVAL": {},
	"POINT": {}, "LINESTRING": {}, "POLYGON": {}, "MULTIPOINT": {}, "MULTILINESTRING": {}, "MULTIPOLYGON": {}, "GEOMETRYCOLLECTION": {}, "BBOX": {},
	"S_INTERSECTS": {}, "S_EQUALS": {}, "S_DISJOINT": {}, "S_TOUCHES": {}, "S_WITHIN": {}, "S_OVERLAPS": {}, "S_CROSSES": {}, "S_CONTAINS": {},
	"T_AFTER": {}, "T_BEFORE": {}, "T_CONTAINS": {}, "T_DISJOINT": {}, "T_DURING": {}, "T_EQUALS": {}, "T_FINISHEDBY": {}, "T_FINISHES": {}, "T_INTERSECTS": {}, "T_MEETS": {}, "T_METBY": {}, "T_OVERLAPPEDBY": {}, "T_OVERLAPS": {}, "T_STARTEDBY": {}, "T_STARTS": {},
	"A_EQUALS": {}, "A_CONTAINS": {}, "A_CONTAINEDBY": {}, "A_OVERLAPS": {},
}

var keywordFunctions = map[string]struct{}{
	"CASEI": {}, "ACCENTI": {}, "DATE": {}, "TIMESTAMP": {}, "INTERVAL": {},
	"POINT": {}, "LINESTRING": {}, "POLYGON": {}, "MULTIPOINT": {}, "MULTILINESTRING": {}, "MULTIPOLYGON": {}, "GEOMETRYCOLLECTION": {}, "BBOX": {},
	"S_INTERSECTS": {}, "S_EQUALS": {}, "S_DISJOINT": {}, "S_TOUCHES": {}, "S_WITHIN": {}, "S_OVERLAPS": {}, "S_CROSSES": {}, "S_CONTAINS": {},
	"T_AFTER": {}, "T_BEFORE": {}, "T_CONTAINS": {}, "T_DISJOINT": {}, "T_DURING": {}, "T_EQUALS": {}, "T_FINISHEDBY": {}, "T_FINISHES": {}, "T_INTERSECTS": {}, "T_MEETS": {}, "T_METBY": {}, "T_OVERLAPPEDBY": {}, "T_OVERLAPS": {}, "T_STARTEDBY": {}, "T_STARTS": {},
	"A_EQUALS": {}, "A_CONTAINS": {}, "A_CONTAINEDBY": {}, "A_OVERLAPS": {},
}

func lexText(input string) ([]token, error) {
	l := &lexer{input: input, line: 1, column: 1}
	var tokens []token
	for {
		tok, err := l.nextToken()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, tok)
		if tok.kind == tokenEOF {
			return tokens, nil
		}
		l.previousKind = tok.kind
		l.hasPrevious = true
	}
}

func (l *lexer) nextToken() (token, error) {
	l.skipWhitespace()
	start := l.location()
	if l.byteOffset >= len(l.input) {
		return token{kind: tokenEOF, span: api.Span{Start: start, End: start}}, nil
	}

	r, size := utf8.DecodeRuneInString(l.input[l.byteOffset:])
	switch {
	case r == '(':
		l.advance(r, size)
		return token{kind: tokenLParen, text: "(", span: api.Span{Start: start, End: l.location()}}, nil
	case r == ')':
		l.advance(r, size)
		return token{kind: tokenRParen, text: ")", span: api.Span{Start: start, End: l.location()}}, nil
	case r == ',':
		l.advance(r, size)
		return token{kind: tokenComma, text: ",", span: api.Span{Start: start, End: l.location()}}, nil
	case r == '\'':
		return l.stringToken(start)
	case r == '"':
		return l.quotedIdentifierToken(start)
	case isNumberStart(l.input[l.byteOffset:], l.canStartSignedNumber()):
		return l.numberToken(start)
	case isIdentifierStartRune(r, size):
		return l.identifierToken(start)
	case strings.ContainsRune("=<>+-*/%^", r):
		return l.operatorToken(start)
	default:
		return token{}, parseError(api.LanguageText, start, fmt.Sprintf("unexpected character %q", r))
	}
}

func (l *lexer) skipWhitespace() {
	for l.byteOffset < len(l.input) {
		r, size := utf8.DecodeRuneInString(l.input[l.byteOffset:])
		if !unicode.IsSpace(r) {
			return
		}
		l.advance(r, size)
	}
}

func (l *lexer) stringToken(start api.Location) (token, error) {
	l.advance('\'', 1)
	var b strings.Builder
	for l.byteOffset < len(l.input) {
		r, size := utf8.DecodeRuneInString(l.input[l.byteOffset:])
		if r == '\'' {
			if strings.HasPrefix(l.input[l.byteOffset:], "''") {
				b.WriteRune('\'')
				l.advance('\'', 1)
				l.advance('\'', 1)
				continue
			}
			l.advance(r, size)
			return token{kind: tokenString, text: b.String(), span: api.Span{Start: start, End: l.location()}}, nil
		}
		if r == '\\' && strings.HasPrefix(l.input[l.byteOffset:], "\\'") {
			b.WriteRune('\'')
			l.advance('\\', 1)
			l.advance('\'', 1)
			continue
		}
		b.WriteRune(r)
		l.advance(r, size)
	}
	return token{}, parseError(api.LanguageText, start, "unterminated string literal")
}

func (l *lexer) quotedIdentifierToken(start api.Location) (token, error) {
	l.advance('"', 1)
	var b strings.Builder
	first := true
	for l.byteOffset < len(l.input) {
		r, size := utf8.DecodeRuneInString(l.input[l.byteOffset:])
		if r == '"' {
			l.advance(r, size)
			if first {
				return token{}, parseError(api.LanguageText, start, "quoted identifier must not be empty")
			}
			return token{kind: tokenQuotedIdentifier, text: b.String(), span: api.Span{Start: start, End: l.location()}}, nil
		}
		if first {
			if !isIdentifierStartRune(r, size) {
				return token{}, parseError(api.LanguageText, l.location(), fmt.Sprintf("invalid quoted identifier start character %q", r))
			}
			first = false
		} else if !isIdentifierPartRune(r, size) {
			return token{}, parseError(api.LanguageText, l.location(), fmt.Sprintf("invalid quoted identifier character %q", r))
		}
		b.WriteRune(r)
		l.advance(r, size)
	}
	return token{}, parseError(api.LanguageText, start, "unterminated quoted identifier")
}

func (l *lexer) numberToken(start api.Location) (token, error) {
	begin := l.byteOffset
	if l.peekByte('+') || l.peekByte('-') {
		r, size := utf8.DecodeRuneInString(l.input[l.byteOffset:])
		l.advance(r, size)
	}
	seenDigit := false
	for l.byteOffset < len(l.input) {
		r, size := utf8.DecodeRuneInString(l.input[l.byteOffset:])
		if r < '0' || r > '9' {
			break
		}
		seenDigit = true
		l.advance(r, size)
	}
	if l.peekByte('.') {
		l.advance('.', 1)
		for l.byteOffset < len(l.input) {
			r, size := utf8.DecodeRuneInString(l.input[l.byteOffset:])
			if r < '0' || r > '9' {
				break
			}
			seenDigit = true
			l.advance(r, size)
		}
	}
	if !seenDigit {
		return token{}, parseError(api.LanguageText, start, "invalid numeric literal")
	}
	if l.peekByte('e') || l.peekByte('E') {
		r, size := utf8.DecodeRuneInString(l.input[l.byteOffset:])
		l.advance(r, size)
		if l.peekByte('+') || l.peekByte('-') {
			r, size = utf8.DecodeRuneInString(l.input[l.byteOffset:])
			l.advance(r, size)
		}
		expDigits := false
		for l.byteOffset < len(l.input) {
			r, size = utf8.DecodeRuneInString(l.input[l.byteOffset:])
			if r < '0' || r > '9' {
				break
			}
			expDigits = true
			l.advance(r, size)
		}
		if !expDigits {
			return token{}, parseError(api.LanguageText, start, "invalid numeric exponent")
		}
	}
	return token{kind: tokenNumber, text: l.input[begin:l.byteOffset], span: api.Span{Start: start, End: l.location()}}, nil
}

func (l *lexer) identifierToken(start api.Location) (token, error) {
	begin := l.byteOffset
	for l.byteOffset < len(l.input) {
		r, size := utf8.DecodeRuneInString(l.input[l.byteOffset:])
		if !isIdentifierPartRune(r, size) {
			break
		}
		l.advance(r, size)
	}
	text := l.input[begin:l.byteOffset]
	upper := strings.ToUpper(text)
	if _, ok := reservedTextKeywords[upper]; ok {
		return token{kind: tokenKeyword, text: upper, span: api.Span{Start: start, End: l.location()}}, nil
	}
	return token{kind: tokenIdentifier, text: text, span: api.Span{Start: start, End: l.location()}}, nil
}

func (l *lexer) operatorToken(start api.Location) (token, error) {
	if l.byteOffset+1 <= len(l.input) {
		for _, op := range []string{"<=", ">=", "<>"} {
			if strings.HasPrefix(l.input[l.byteOffset:], op) {
				for _, r := range op {
					l.advance(r, 1)
				}
				return token{kind: tokenOperator, text: op, span: api.Span{Start: start, End: l.location()}}, nil
			}
		}
	}
	r, size := utf8.DecodeRuneInString(l.input[l.byteOffset:])
	l.advance(r, size)
	return token{kind: tokenOperator, text: string(r), span: api.Span{Start: start, End: l.location()}}, nil
}

func (l *lexer) location() api.Location {
	return api.Location{ByteOffset: l.byteOffset, CharOffset: l.charOffset, Line: l.line, Column: l.column}
}

func (l *lexer) advance(r rune, size int) {
	l.byteOffset += size
	l.charOffset++
	if r == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}
}

func (l *lexer) peekByte(b byte) bool {
	return l.byteOffset < len(l.input) && l.input[l.byteOffset] == b
}

func (l *lexer) canStartSignedNumber() bool {
	if !l.hasPrevious {
		return true
	}
	switch l.previousKind {
	case tokenOperator, tokenLParen, tokenComma, tokenKeyword:
		return true
	default:
		return false
	}
}

func isNumberStart(s string, allowSign bool) bool {
	if s == "" {
		return false
	}
	if s[0] >= '0' && s[0] <= '9' {
		return true
	}
	if s[0] == '.' {
		return len(s) > 1 && s[1] >= '0' && s[1] <= '9'
	}
	if allowSign && (s[0] == '+' || s[0] == '-') {
		if len(s) < 2 {
			return false
		}
		return s[1] >= '0' && s[1] <= '9' || s[1] == '.' && len(s) > 2 && s[2] >= '0' && s[2] <= '9'
	}
	return false
}

func isIdentifierStartRune(r rune, size int) bool {
	return isValidUTF8Rune(r, size) && isIdentifierStart(r)
}

func isIdentifierPartRune(r rune, size int) bool {
	return isValidUTF8Rune(r, size) && isIdentifierPart(r)
}

func isValidUTF8Rune(r rune, size int) bool {
	return r != utf8.RuneError || size != 1
}

func isIdentifierStart(r rune) bool {
	switch {
	case r == ':' || r == '_':
		return true
	case 'A' <= r && r <= 'Z':
		return true
	case 'a' <= r && r <= 'z':
		return true
	case '\u00c0' <= r && r <= '\u00d6':
		return true
	case '\u00d8' <= r && r <= '\u00f6':
		return true
	case '\u00f8' <= r && r <= '\u02ff':
		return true
	case '\u0370' <= r && r <= '\u037d':
		return true
	case '\u037f' <= r && r <= '\u1ffe':
		return true
	case '\u200c' <= r && r <= '\u200d':
		return true
	case '\u2070' <= r && r <= '\u218f':
		return true
	case '\u2c00' <= r && r <= '\u2fef':
		return true
	case '\u3001' <= r && r <= '\ud7ff':
		return true
	case '\uf900' <= r && r <= '\ufdcf':
		return true
	case '\ufdf0' <= r && r <= '\ufffd':
		return true
	case 0x10000 <= r && r <= 0xeffff:
		return true
	default:
		return false
	}
}

func isIdentifierPart(r rune) bool {
	return isIdentifierStart(r) || isASCIIDigit(r) || r == '.' || '\u0300' <= r && r <= '\u036f' || '\u203f' <= r && r <= '\u2040'
}

func isASCIIDigit(r rune) bool {
	return '0' <= r && r <= '9'
}
