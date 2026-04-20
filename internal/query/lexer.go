package query

import (
	"strings"
)

type tokenKind int

const (
	tokEOF tokenKind = iota
	tokLParen
	tokRParen
	tokColon
	tokAnd
	tokOr
	tokNot
	tokWord
	tokQuoted
)

type token struct {
	kind  tokenKind
	value string
	pos   int
}

// lex tokenizes the input query string. Whitespace is a separator but
// produces no token; juxtaposition between tokens is implicit AND. The
// reserved words AND/OR/NOT (case-insensitive) and a leading "-" become
// dedicated tokens.
func lex(input string) ([]token, error) {
	var out []token
	i := 0
	for i < len(input) {
		c := input[i]
		if isSpace(c) {
			i++
			continue
		}
		switch c {
		case '(':
			out = append(out, token{kind: tokLParen, value: "(", pos: i})
			i++
		case ')':
			out = append(out, token{kind: tokRParen, value: ")", pos: i})
			i++
		case ':':
			out = append(out, token{kind: tokColon, value: ":", pos: i})
			i++
		case '"':
			t, next, err := scanQuoted(input, i)
			if err != nil {
				return nil, err
			}
			out = append(out, t)
			i = next
		case '-':
			out = append(out, token{kind: tokNot, value: "-", pos: i})
			i++
		default:
			t, next := scanWord(input, i)
			out = append(out, t)
			i = next
		}
	}
	out = append(out, token{kind: tokEOF, value: "", pos: i})
	return out, nil
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func isWordBoundary(c byte) bool {
	return isSpace(c) || c == '(' || c == ')' || c == ':' || c == '"'
}

// scanQuoted consumes `"..."` starting at input[start] (which must be `"`).
// Supports \" and \\ escapes. Returns the token (kind=tokQuoted, value=
// unescaped contents, pos=start) and the next index (one past the closing
// quote). An unterminated quote yields a *ParseError at the opening quote.
func scanQuoted(input string, start int) (token, int, error) {
	var sb strings.Builder
	i := start + 1
	for i < len(input) {
		c := input[i]
		if c == '\\' && i+1 < len(input) {
			n := input[i+1]
			if n == '"' || n == '\\' {
				sb.WriteByte(n)
				i += 2
				continue
			}
		}
		if c == '"' {
			return token{kind: tokQuoted, value: sb.String(), pos: start}, i + 1, nil
		}
		sb.WriteByte(c)
		i++
	}
	return token{}, 0, &ParseError{Pos: start, Msg: "unterminated quoted string"}
}

// scanWord consumes a bareword starting at input[start]. A bareword runs
// until whitespace, a paren, a colon, or a quote. If the bareword matches
// a reserved keyword (AND/OR/NOT, case-insensitive), the corresponding
// reserved token is emitted with the original case preserved in value.
func scanWord(input string, start int) (token, int) {
	i := start
	for i < len(input) && !isWordBoundary(input[i]) {
		i++
	}
	lit := input[start:i]
	switch strings.ToUpper(lit) {
	case "AND":
		return token{kind: tokAnd, value: lit, pos: start}, i
	case "OR":
		return token{kind: tokOr, value: lit, pos: start}, i
	case "NOT":
		return token{kind: tokNot, value: lit, pos: start}, i
	}
	return token{kind: tokWord, value: lit, pos: start}, i
}
