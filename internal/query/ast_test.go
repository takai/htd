package query

import (
	"testing"

	"github.com/takai/htd/internal/model"
)

func fixtureItem() *model.Item {
	return &model.Item{
		ID:      "20260421-fix_parser_panic",
		Title:   "Fix parser panic",
		Kind:    model.KindNextAction,
		Status:  model.StatusActive,
		Project: "launch_cli",
		Source:  "email",
		Tags:    []string{"bug", "cli"},
		Refs:    []string{"https://github.com/foo/bar/pull/42"},
	}
}

const fixtureBody = "Reproduces on macOS during startup."

func TestMatch(t *testing.T) {
	item := fixtureItem()
	cases := []struct {
		name  string
		query string
		want  bool
	}{
		{"empty matches all", "", true},
		{"unfielded hits title", "parser", true},
		{"unfielded hits body", "macos", true},
		{"unfielded hits id", "20260421", true},
		{"unfielded hits tag element", "bug", true},
		{"unfielded hits ref element", "github.com", true},
		{"unfielded hits source", "email", true},
		{"unfielded hits project", "launch", true},
		{"unfielded misses unrelated", "nonsense_xyz", false},
		{"unfielded does not match kind enum", "next_action", false},
		{"unfielded does not match status enum", "active", false},
		{"case insensitive unfielded", "PARSER", true},
		{"fielded title hit", "title:parser", true},
		{"fielded title miss", "title:nonsense", false},
		{"fielded title quoted", `title:"fix parser"`, true},
		{"fielded body hit", "body:macOS", true},
		{"fielded body miss", "body:nonsense", false},
		{"fielded kind substring", "kind:next", true},
		{"fielded kind exact", "kind:next_action", true},
		{"fielded kind miss", "kind:project", false},
		{"fielded status hit", "status:active", true},
		{"fielded status miss", "status:done", false},
		{"fielded project hit", "project:launch_cli", true},
		{"fielded project miss", "project:other", false},
		{"fielded source hit", "source:email", true},
		{"fielded tag any element", "tag:bug", true},
		{"fielded tag substring of element", "tag:cl", true},
		{"fielded tag miss", "tag:other", false},
		{"fielded ref any element", "ref:github.com", true},
		{"fielded ref quoted URL", `ref:"https://github.com"`, true},
		{"fielded ref miss", "ref:notion.so", false},
		{"fielded id hit", "id:fix_parser", true},
		{"AND both true", "parser body:macos", true},
		{"AND one false", "parser body:nonsense", false},
		{"OR either true", "nonsense OR parser", true},
		{"OR both false", "nonsense OR xyzzy", false},
		{"NOT flips", "NOT parser", false},
		{"NOT of miss is true", "NOT nonsense", true},
		{"dash NOT flips", "-parser", false},
		{"parens grouping", "(nonsense OR parser) tag:bug", true},
		{"parens grouping false", "(nonsense OR xyzzy) tag:bug", false},
		{"issue-style compound",
			"(ref:github.com OR ref:notion.so) tag:bug", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			q, err := Parse(c.query)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", c.query, err)
			}
			got := q.Match(item, fixtureBody)
			if got != c.want {
				t.Errorf("Match(%q) = %v, want %v", c.query, got, c.want)
			}
		})
	}
}

func TestMatchEmptyFields(t *testing.T) {
	// Item with sparse fields: no project, no source, empty tags/refs.
	item := &model.Item{
		ID:     "20260421-x",
		Title:  "x",
		Kind:   model.KindInbox,
		Status: model.StatusActive,
	}
	cases := []struct {
		name  string
		query string
		want  bool
	}{
		{"unfielded does not falsely hit empty project", "nonsense", false},
		{"fielded project miss on empty", "project:foo", false},
		{"fielded source miss on empty", "source:foo", false},
		{"fielded tag miss on empty", "tag:foo", false},
		{"fielded ref miss on empty", "ref:foo", false},
		{"NOT of miss on empty is true", "NOT project:foo", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			q, err := Parse(c.query)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", c.query, err)
			}
			got := q.Match(item, "")
			if got != c.want {
				t.Errorf("Match(%q) = %v, want %v", c.query, got, c.want)
			}
		})
	}
}
