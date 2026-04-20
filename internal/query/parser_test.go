package query

import (
	"errors"
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  Node
	}{
		{"empty", "", nil},
		{"whitespace only", "   ", nil},
		{"single unfielded", "panic", &TermNode{Needle: "panic"}},
		{"unfielded lowercased", "Panic", &TermNode{Needle: "panic"}},
		{"single fielded", "title:foo", &TermNode{Field: "title", Needle: "foo"}},
		{"fielded quoted", `title:"fix panic"`, &TermNode{Field: "title", Needle: "fix panic"}},
		{"reserved word in value position",
			"title:OR", &TermNode{Field: "title", Needle: "or"}},
		{"implicit AND", "a b",
			&AndNode{L: &TermNode{Needle: "a"}, R: &TermNode{Needle: "b"}}},
		{"explicit AND", "a AND b",
			&AndNode{L: &TermNode{Needle: "a"}, R: &TermNode{Needle: "b"}}},
		{"three-way AND left-associates",
			"a b c",
			&AndNode{
				L: &AndNode{L: &TermNode{Needle: "a"}, R: &TermNode{Needle: "b"}},
				R: &TermNode{Needle: "c"},
			}},
		{"OR", "a OR b",
			&OrNode{L: &TermNode{Needle: "a"}, R: &TermNode{Needle: "b"}}},
		{"NOT", "NOT a", &NotNode{X: &TermNode{Needle: "a"}}},
		{"leading dash is NOT", "-a", &NotNode{X: &TermNode{Needle: "a"}}},
		{"NOT binds tighter than AND",
			"NOT a b",
			&AndNode{L: &NotNode{X: &TermNode{Needle: "a"}}, R: &TermNode{Needle: "b"}}},
		{"AND tighter than OR",
			"a b OR c",
			&OrNode{
				L: &AndNode{L: &TermNode{Needle: "a"}, R: &TermNode{Needle: "b"}},
				R: &TermNode{Needle: "c"},
			}},
		{"parens override precedence",
			"a (b OR c)",
			&AndNode{
				L: &TermNode{Needle: "a"},
				R: &OrNode{L: &TermNode{Needle: "b"}, R: &TermNode{Needle: "c"}},
			}},
		{"dash before fielded",
			"-tag:bug",
			&NotNode{X: &TermNode{Field: "tag", Needle: "bug"}}},
		{"ref fielded",
			"ref:github.com",
			&TermNode{Field: "ref", Needle: "github.com"}},
		{"complex: issue-style",
			"(ref:github.com OR ref:notion.so) tag:bug",
			&AndNode{
				L: &OrNode{
					L: &TermNode{Field: "ref", Needle: "github.com"},
					R: &TermNode{Field: "ref", Needle: "notion.so"},
				},
				R: &TermNode{Field: "tag", Needle: "bug"},
			}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			q, err := Parse(c.input)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", c.input, err)
			}
			if !reflect.DeepEqual(q.root, c.want) {
				t.Errorf("Parse(%q).root =\n  %#v\nwant\n  %#v", c.input, q.root, c.want)
			}
		})
	}
}

func TestParseError(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"unknown field", "unknownfield:foo"},
		{"trailing OR", "foo OR"},
		{"leading OR", "OR foo"},
		{"trailing AND", "a AND"},
		{"leading AND", "AND a"},
		{"empty parens", "()"},
		{"unmatched LParen", "(a"},
		{"unmatched RParen", "a)"},
		{"missing value after colon", "title:"},
		{"lone colon", ":"},
		{"lone NOT", "NOT"},
		{"lone dash", "-"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := Parse(c.input)
			if err == nil {
				t.Fatalf("Parse(%q) expected error, got nil", c.input)
			}
			var pe *ParseError
			if !errors.As(err, &pe) {
				t.Fatalf("Parse(%q): expected *ParseError, got %T: %v", c.input, err, err)
			}
			if pe.Pos < 0 {
				t.Errorf("Parse(%q): ParseError.Pos = %d, want >= 0", c.input, pe.Pos)
			}
		})
	}
}

func TestParseErrorUnknownFieldMessage(t *testing.T) {
	_, err := Parse("tilte:foo")
	if err == nil {
		t.Fatal("expected error")
	}
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *ParseError, got %T", err)
	}
	if !containsAll(pe.Msg, "unknown field", "tilte") {
		t.Errorf("ParseError.Msg = %q; want to mention unknown field and \"tilte\"", pe.Msg)
	}
	if pe.Pos != 0 {
		t.Errorf("ParseError.Pos = %d, want 0 (field position)", pe.Pos)
	}
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !contains(s, sub) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
