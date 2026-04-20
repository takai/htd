// Package query implements a minimal query DSL for filtering items.
//
// Grammar:
//
//	query   := orExpr
//	orExpr  := andExpr ( OR andExpr )*
//	andExpr := unary ( unary )+        // implicit AND by juxtaposition
//	unary   := ( NOT | '-' ) unary | primary
//	primary := '(' query ')' | term
//	term    := [WORD ':'] (WORD | QUOTED)
//
// Default match is case-insensitive substring. Field:value restricts the
// match to a single field. Array-valued fields (tags, refs) match if any
// element contains the needle.
package query

import (
	"fmt"
	"slices"

	"github.com/takai/htd/internal/model"
)

// ParseError reports a problem at a specific byte offset in the input.
type ParseError struct {
	Pos int
	Msg string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("query: %s at pos %d", e.Msg, e.Pos)
}

// Query is a parsed, ready-to-evaluate query expression. The zero value
// and queries parsed from the empty string match every item.
type Query struct {
	root Node
}

// Parse parses a query expression. Empty input produces a match-all Query.
func Parse(input string) (*Query, error) {
	toks, err := lex(input)
	if err != nil {
		return nil, err
	}
	p := &parser{toks: toks}
	root, err := p.parseQuery()
	if err != nil {
		return nil, err
	}
	return &Query{root: root}, nil
}

// Match reports whether the item (plus its Markdown body) satisfies the
// query. A match-all query returns true for every input.
func (q *Query) Match(item *model.Item, body string) bool {
	if q == nil || q.root == nil {
		return true
	}
	ctx := &evalCtx{item: item, body: body}
	return q.root.eval(ctx)
}

// ValidFields returns the whitelist of field names that can be used on the
// left side of a colon (e.g., `title:foo`). Returned slice is a copy.
func ValidFields() []string {
	out := make([]string, len(validFields))
	copy(out, validFields)
	return out
}

var validFields = []string{
	"id",
	"title",
	"body",
	"kind",
	"status",
	"project",
	"source",
	"tag",
	"ref",
}

func isValidField(name string) bool {
	return slices.Contains(validFields, name)
}
