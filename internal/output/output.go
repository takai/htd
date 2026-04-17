package output

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/takai/htd/internal/model"
)

const (
	ExitOK       = 0
	ExitError    = 1
	ExitNotFound = 2
)

type Printer struct {
	out  io.Writer
	err  io.Writer
	json bool
}

func New(out, err io.Writer, jsonMode bool) *Printer {
	return &Printer{out: out, err: err, json: jsonMode}
}

func (p *Printer) PrintID(id string) {
	fmt.Fprintln(p.out, id)
}

func (p *Printer) PrintPaths(paths []string) {
	if p.json {
		data, _ := json.Marshal(paths)
		fmt.Fprintln(p.out, string(data))
		return
	}
	for _, d := range paths {
		fmt.Fprintln(p.out, d)
	}
}

func (p *Printer) PrintItem(item *model.Item, body string) {
	if p.json {
		p.printItemJSON(item, body)
	} else {
		p.printItemText(item, body)
	}
}

func (p *Printer) PrintItems(items []*model.Item) {
	if p.json {
		p.printItemsJSON(items)
	} else {
		p.printItemsText(items)
	}
}

func (p *Printer) PrintError(msg string) {
	if p.json {
		data, _ := json.Marshal(map[string]string{"error": msg})
		fmt.Fprintln(p.err, string(data))
	} else {
		fmt.Fprintln(p.err, "error:", msg)
	}
}

func (p *Printer) printItemText(item *model.Item, body string) {
	fmt.Fprintf(p.out, "id:         %s\n", item.ID)
	fmt.Fprintf(p.out, "title:      %s\n", item.Title)
	fmt.Fprintf(p.out, "kind:       %s\n", item.Kind)
	fmt.Fprintf(p.out, "status:     %s\n", item.Status)
	if item.Project != "" {
		fmt.Fprintf(p.out, "project:    %s\n", item.Project)
	}
	fmt.Fprintf(p.out, "created_at: %s\n", item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	fmt.Fprintf(p.out, "updated_at: %s\n", item.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
	if item.DueAt != nil {
		fmt.Fprintf(p.out, "due_at:     %s\n", item.DueAt.Format("2006-01-02T15:04:05Z07:00"))
	}
	if item.DeferUntil != nil {
		fmt.Fprintf(p.out, "defer_until:%s\n", item.DeferUntil.Format("2006-01-02T15:04:05Z07:00"))
	}
	if item.ReviewAt != nil {
		fmt.Fprintf(p.out, "review_at:  %s\n", item.ReviewAt.Format("2006-01-02T15:04:05Z07:00"))
	}
	if item.Source != "" {
		fmt.Fprintf(p.out, "source:     %s\n", item.Source)
	}
	if len(item.Tags) > 0 {
		fmt.Fprintf(p.out, "tags:       %v\n", item.Tags)
	}
	if body != "" {
		fmt.Fprintf(p.out, "\n%s\n", body)
	}
}

func (p *Printer) printItemsText(items []*model.Item) {
	tw := tabwriter.NewWriter(p.out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tKIND\tSTATUS\tTITLE")
	for _, it := range items {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", it.ID, it.Kind, it.Status, truncateRunes(it.Title, 40))
	}
	_ = tw.Flush()
}

// truncateRunes returns s truncated to at most max runes, appending "..." if it
// was shortened. Slices by rune boundaries so multi-byte characters stay intact.
func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

// itemJSON is a flat JSON representation of an Item including body.
type itemJSON struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Kind       string    `json:"kind"`
	Status     string    `json:"status"`
	Project    string    `json:"project,omitempty"`
	CreatedAt  string    `json:"created_at"`
	UpdatedAt  string    `json:"updated_at"`
	DueAt      string    `json:"due_at,omitempty"`
	DeferUntil string    `json:"defer_until,omitempty"`
	ReviewAt   string    `json:"review_at,omitempty"`
	Source     string    `json:"source,omitempty"`
	Tags       []string  `json:"tags,omitempty"`
	Body       string    `json:"body,omitempty"`
}

func toItemJSON(item *model.Item, body string) itemJSON {
	j := itemJSON{
		ID:        item.ID,
		Title:     item.Title,
		Kind:      string(item.Kind),
		Status:    string(item.Status),
		Project:   item.Project,
		CreatedAt: item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: item.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Source:    item.Source,
		Tags:      item.Tags,
		Body:      body,
	}
	if item.DueAt != nil {
		j.DueAt = item.DueAt.Format("2006-01-02T15:04:05Z07:00")
	}
	if item.DeferUntil != nil {
		j.DeferUntil = item.DeferUntil.Format("2006-01-02T15:04:05Z07:00")
	}
	if item.ReviewAt != nil {
		j.ReviewAt = item.ReviewAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return j
}

func (p *Printer) printItemJSON(item *model.Item, body string) {
	data, _ := json.Marshal(toItemJSON(item, body))
	fmt.Fprintln(p.out, string(data))
}

func (p *Printer) printItemsJSON(items []*model.Item) {
	arr := make([]itemJSON, len(items))
	for i, it := range items {
		arr[i] = toItemJSON(it, "")
	}
	data, _ := json.Marshal(arr)
	fmt.Fprintln(p.out, string(data))
}

type waitingItemJSON struct {
	itemJSON
	AgeDays int `json:"age_days"`
}

// PrintWaitingItems prints items with an age_days field added in JSON output.
// Text output is identical to PrintItems. ageDays must be the same length as items.
func (p *Printer) PrintWaitingItems(items []*model.Item, ageDays []int) {
	if p.json {
		arr := make([]waitingItemJSON, len(items))
		for i, it := range items {
			arr[i] = waitingItemJSON{itemJSON: toItemJSON(it, ""), AgeDays: ageDays[i]}
		}
		data, _ := json.Marshal(arr)
		fmt.Fprintln(p.out, string(data))
		return
	}
	p.printItemsText(items)
}
