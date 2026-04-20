package query

import (
	"fmt"
	"strings"
)

type parser struct {
	toks []token
	i    int
}

func (p *parser) peek() token {
	return p.toks[p.i]
}

func (p *parser) peekN(n int) token {
	if p.i+n >= len(p.toks) {
		return p.toks[len(p.toks)-1]
	}
	return p.toks[p.i+n]
}

func (p *parser) advance() token {
	t := p.toks[p.i]
	if t.kind != tokEOF {
		p.i++
	}
	return t
}

func (p *parser) parseQuery() (Node, error) {
	if p.peek().kind == tokEOF {
		return nil, nil
	}
	n, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tokEOF {
		t := p.peek()
		return nil, &ParseError{Pos: t.pos, Msg: fmt.Sprintf("unexpected %s", tokDesc(t))}
	}
	return n, nil
}

func (p *parser) parseOr() (Node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.peek().kind == tokOr {
		p.advance()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &OrNode{L: left, R: right}
	}
	return left, nil
}

func (p *parser) parseAnd() (Node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for {
		t := p.peek()
		if t.kind == tokAnd {
			p.advance()
			right, err := p.parseUnary()
			if err != nil {
				return nil, err
			}
			left = &AndNode{L: left, R: right}
			continue
		}
		if !startsUnary(t.kind) {
			break
		}
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &AndNode{L: left, R: right}
	}
	return left, nil
}

func (p *parser) parseUnary() (Node, error) {
	if p.peek().kind == tokNot {
		p.advance()
		inner, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &NotNode{X: inner}, nil
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() (Node, error) {
	t := p.peek()
	if t.kind == tokLParen {
		p.advance()
		if p.peek().kind == tokRParen {
			return nil, &ParseError{Pos: p.peek().pos, Msg: "empty parentheses"}
		}
		inner, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.peek().kind != tokRParen {
			return nil, &ParseError{Pos: p.peek().pos, Msg: "expected ')'"}
		}
		p.advance()
		return inner, nil
	}
	return p.parseTerm()
}

func (p *parser) parseTerm() (Node, error) {
	t := p.peek()
	if t.kind == tokWord && p.peekN(1).kind == tokColon {
		field := strings.ToLower(t.value)
		if !isValidField(field) {
			return nil, &ParseError{
				Pos: t.pos,
				Msg: fmt.Sprintf("unknown field %q (valid: %s)", t.value, strings.Join(validFields, ", ")),
			}
		}
		p.advance() // word
		p.advance() // colon
		v := p.peek()
		switch v.kind {
		case tokWord, tokQuoted, tokOr, tokAnd, tokNot:
			p.advance()
			return &TermNode{Field: field, Needle: strings.ToLower(v.value)}, nil
		default:
			return nil, &ParseError{Pos: v.pos, Msg: "expected value after ':'"}
		}
	}
	switch t.kind {
	case tokWord, tokQuoted:
		p.advance()
		return &TermNode{Needle: strings.ToLower(t.value)}, nil
	case tokEOF:
		return nil, &ParseError{Pos: t.pos, Msg: "unexpected end of input"}
	default:
		return nil, &ParseError{Pos: t.pos, Msg: fmt.Sprintf("unexpected %s", tokDesc(t))}
	}
}

func startsUnary(k tokenKind) bool {
	switch k {
	case tokWord, tokQuoted, tokLParen, tokNot:
		return true
	}
	return false
}

func tokDesc(t token) string {
	switch t.kind {
	case tokEOF:
		return "end of input"
	case tokLParen:
		return "'('"
	case tokRParen:
		return "')'"
	case tokColon:
		return "':'"
	case tokAnd:
		return "AND"
	case tokOr:
		return "OR"
	case tokNot:
		return "NOT"
	case tokWord:
		return fmt.Sprintf("%q", t.value)
	case tokQuoted:
		return fmt.Sprintf("quoted %q", t.value)
	}
	return "token"
}
