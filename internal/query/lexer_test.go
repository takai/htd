package query

import (
	"errors"
	"testing"
)

type tok struct {
	kind  tokenKind
	value string
}

func lexToks(t *testing.T, input string) []tok {
	t.Helper()
	toks, err := lex(input)
	if err != nil {
		t.Fatalf("lex(%q) unexpected error: %v", input, err)
	}
	out := make([]tok, 0, len(toks))
	for _, x := range toks {
		out = append(out, tok{x.kind, x.value})
	}
	return out
}

func TestLex(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  []tok
	}{
		{"empty", "", []tok{{tokEOF, ""}}},
		{"whitespace only", "   \t  ", []tok{{tokEOF, ""}}},
		{"single word", "foo", []tok{{tokWord, "foo"}, {tokEOF, ""}}},
		{"surrounding whitespace trimmed", "  foo  ", []tok{{tokWord, "foo"}, {tokEOF, ""}}},
		{"two words", "foo bar", []tok{{tokWord, "foo"}, {tokWord, "bar"}, {tokEOF, ""}}},
		{"parens", "(a b)", []tok{
			{tokLParen, "("}, {tokWord, "a"}, {tokWord, "b"}, {tokRParen, ")"}, {tokEOF, ""},
		}},
		{"colon fielded", "title:foo", []tok{
			{tokWord, "title"}, {tokColon, ":"}, {tokWord, "foo"}, {tokEOF, ""},
		}},
		{"quoted simple", `"fix panic"`, []tok{{tokQuoted, "fix panic"}, {tokEOF, ""}}},
		{"quoted with escape quote", `"a\"b"`, []tok{{tokQuoted, `a"b`}, {tokEOF, ""}}},
		{"quoted with escape backslash", `"a\\b"`, []tok{{tokQuoted, `a\b`}, {tokEOF, ""}}},
		{"field quoted", `title:"fix panic"`, []tok{
			{tokWord, "title"}, {tokColon, ":"}, {tokQuoted, "fix panic"}, {tokEOF, ""},
		}},
		{"OR reserved uppercase", "a OR b", []tok{
			{tokWord, "a"}, {tokOr, "OR"}, {tokWord, "b"}, {tokEOF, ""},
		}},
		{"or reserved lowercase", "a or b", []tok{
			{tokWord, "a"}, {tokOr, "or"}, {tokWord, "b"}, {tokEOF, ""},
		}},
		{"NOT reserved", "NOT a", []tok{{tokNot, "NOT"}, {tokWord, "a"}, {tokEOF, ""}}},
		{"AND reserved", "a AND b", []tok{
			{tokWord, "a"}, {tokAnd, "AND"}, {tokWord, "b"}, {tokEOF, ""},
		}},
		{"leading dash is NOT", "-foo", []tok{{tokNot, "-"}, {tokWord, "foo"}, {tokEOF, ""}}},
		{"intraword dash preserved", "20260417-foo", []tok{
			{tokWord, "20260417-foo"}, {tokEOF, ""},
		}},
		{"dash before field", "-tag:bug", []tok{
			{tokNot, "-"}, {tokWord, "tag"}, {tokColon, ":"}, {tokWord, "bug"}, {tokEOF, ""},
		}},
		{"nested NOT with group", "NOT (a OR b)", []tok{
			{tokNot, "NOT"}, {tokLParen, "("}, {tokWord, "a"}, {tokOr, "OR"}, {tokWord, "b"}, {tokRParen, ")"}, {tokEOF, ""},
		}},
		{"ref value is a URL bareword", "ref:github.com", []tok{
			{tokWord, "ref"}, {tokColon, ":"}, {tokWord, "github.com"}, {tokEOF, ""},
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := lexToks(t, c.input)
			if len(got) != len(c.want) {
				t.Fatalf("lex(%q) got %d tokens, want %d: got=%v want=%v",
					c.input, len(got), len(c.want), got, c.want)
			}
			for i, w := range c.want {
				if got[i].kind != w.kind || got[i].value != w.value {
					t.Errorf("token[%d] = {kind=%d value=%q}, want {kind=%d value=%q}",
						i, got[i].kind, got[i].value, w.kind, w.value)
				}
			}
		})
	}
}

func TestLexError(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantPos int
	}{
		{"unterminated quote", `"foo`, 0},
		{"unterminated quote after escape", `"a\"b`, 0},
		{"unterminated quote mid input", `foo "bar`, 4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := lex(c.input)
			if err == nil {
				t.Fatalf("lex(%q) expected error, got nil", c.input)
			}
			var pe *ParseError
			if !errors.As(err, &pe) {
				t.Fatalf("expected *ParseError, got %T: %v", err, err)
			}
			if pe.Pos != c.wantPos {
				t.Errorf("err.Pos = %d, want %d (err=%v)", pe.Pos, c.wantPos, pe)
			}
		})
	}
}
