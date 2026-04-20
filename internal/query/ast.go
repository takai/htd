package query

import (
	"strings"

	"github.com/takai/htd/internal/model"
)

// Node is a sealed AST node interface.
type Node interface {
	eval(ctx *evalCtx) bool
	isNode()
}

// AndNode matches when both children match.
type AndNode struct{ L, R Node }

// OrNode matches when either child matches.
type OrNode struct{ L, R Node }

// NotNode matches when its child does not.
type NotNode struct{ X Node }

// TermNode matches a single needle against one or more fields. Field == ""
// means unfielded: the needle is searched across the unfielded-field set.
// Needle is always lowercased at parse time for case-insensitive matching.
type TermNode struct {
	Field  string
	Needle string
}

func (*AndNode) isNode()  {}
func (*OrNode) isNode()   {}
func (*NotNode) isNode()  {}
func (*TermNode) isNode() {}

// evalCtx carries per-item state for a single Match call. Lowercase views
// of the item's string fields are computed lazily on first access and
// cached so multi-term queries don't re-lowercase.
type evalCtx struct {
	item *model.Item
	body string

	lcTitle, lcBody, lcProject, lcSource, lcID, lcKind, lcStatus string
	lcTags, lcRefs                                               []string

	titleReady, bodyReady, projectReady, sourceReady, idReady, kindReady, statusReady bool
	tagsReady, refsReady                                                              bool
}

func (c *evalCtx) title() string {
	if !c.titleReady {
		c.lcTitle = strings.ToLower(c.item.Title)
		c.titleReady = true
	}
	return c.lcTitle
}

func (c *evalCtx) bodyLC() string {
	if !c.bodyReady {
		c.lcBody = strings.ToLower(c.body)
		c.bodyReady = true
	}
	return c.lcBody
}

func (c *evalCtx) project() string {
	if !c.projectReady {
		c.lcProject = strings.ToLower(c.item.Project)
		c.projectReady = true
	}
	return c.lcProject
}

func (c *evalCtx) source() string {
	if !c.sourceReady {
		c.lcSource = strings.ToLower(c.item.Source)
		c.sourceReady = true
	}
	return c.lcSource
}

func (c *evalCtx) id() string {
	if !c.idReady {
		c.lcID = strings.ToLower(c.item.ID)
		c.idReady = true
	}
	return c.lcID
}

func (c *evalCtx) kind() string {
	if !c.kindReady {
		c.lcKind = strings.ToLower(string(c.item.Kind))
		c.kindReady = true
	}
	return c.lcKind
}

func (c *evalCtx) status() string {
	if !c.statusReady {
		c.lcStatus = strings.ToLower(string(c.item.Status))
		c.statusReady = true
	}
	return c.lcStatus
}

func (c *evalCtx) tags() []string {
	if !c.tagsReady {
		c.lcTags = lowerSlice(c.item.Tags)
		c.tagsReady = true
	}
	return c.lcTags
}

func (c *evalCtx) refs() []string {
	if !c.refsReady {
		c.lcRefs = lowerSlice(c.item.Refs)
		c.refsReady = true
	}
	return c.lcRefs
}

func lowerSlice(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = strings.ToLower(s)
	}
	return out
}

func (n *AndNode) eval(ctx *evalCtx) bool {
	return n.L.eval(ctx) && n.R.eval(ctx)
}

func (n *OrNode) eval(ctx *evalCtx) bool {
	return n.L.eval(ctx) || n.R.eval(ctx)
}

func (n *NotNode) eval(ctx *evalCtx) bool {
	return !n.X.eval(ctx)
}

func (n *TermNode) eval(ctx *evalCtx) bool {
	switch n.Field {
	case "":
		return ctx.matchUnfielded(n.Needle)
	case "title":
		return strings.Contains(ctx.title(), n.Needle)
	case "body":
		return strings.Contains(ctx.bodyLC(), n.Needle)
	case "project":
		return strings.Contains(ctx.project(), n.Needle)
	case "source":
		return strings.Contains(ctx.source(), n.Needle)
	case "id":
		return strings.Contains(ctx.id(), n.Needle)
	case "kind":
		return strings.Contains(ctx.kind(), n.Needle)
	case "status":
		return strings.Contains(ctx.status(), n.Needle)
	case "tag":
		return anyContains(ctx.tags(), n.Needle)
	case "ref":
		return anyContains(ctx.refs(), n.Needle)
	}
	return false
}

func (ctx *evalCtx) matchUnfielded(needle string) bool {
	if strings.Contains(ctx.title(), needle) {
		return true
	}
	if strings.Contains(ctx.bodyLC(), needle) {
		return true
	}
	if strings.Contains(ctx.id(), needle) {
		return true
	}
	if ctx.item.Project != "" && strings.Contains(ctx.project(), needle) {
		return true
	}
	if ctx.item.Source != "" && strings.Contains(ctx.source(), needle) {
		return true
	}
	if anyContains(ctx.tags(), needle) {
		return true
	}
	if anyContains(ctx.refs(), needle) {
		return true
	}
	return false
}

func anyContains(haystacks []string, needle string) bool {
	for _, h := range haystacks {
		if strings.Contains(h, needle) {
			return true
		}
	}
	return false
}
