package output_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/takai/htd/internal/model"
	"github.com/takai/htd/internal/output"
)

func makeItem(id string, kind model.Kind) *model.Item {
	now := time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC)
	return &model.Item{
		ID:        id,
		Title:     "Test Title",
		Kind:      kind,
		Status:    model.StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
		Tags:      []string{"a", "b"},
	}
}

func TestPrintItemText(t *testing.T) {
	var out bytes.Buffer
	p := output.New(&out, &bytes.Buffer{}, false, false)
	item := makeItem("20260417-test", model.KindInbox)
	p.PrintItem(item, "some body text")

	s := out.String()
	if !strings.Contains(s, "20260417-test") {
		t.Errorf("text output missing ID: %q", s)
	}
	if !strings.Contains(s, "Test Title") {
		t.Errorf("text output missing title: %q", s)
	}
}

func TestPrintItemJSON(t *testing.T) {
	var out bytes.Buffer
	p := output.New(&out, &bytes.Buffer{}, true, false)
	item := makeItem("20260417-json", model.KindNextAction)
	p.PrintItem(item, "body content")

	var obj map[string]any
	if err := json.Unmarshal(out.Bytes(), &obj); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out.String())
	}
	if obj["id"] != "20260417-json" {
		t.Errorf("JSON id: want %q, got %v", "20260417-json", obj["id"])
	}
	if obj["body"] != "body content" {
		t.Errorf("JSON body: want %q, got %v", "body content", obj["body"])
	}
}

func TestPrintItemsText(t *testing.T) {
	var out bytes.Buffer
	p := output.New(&out, &bytes.Buffer{}, false, false)
	items := []*model.Item{
		makeItem("20260417-a", model.KindInbox),
		makeItem("20260417-b", model.KindProject),
	}
	p.PrintItems(items)

	s := out.String()
	if !strings.Contains(s, "20260417-a") || !strings.Contains(s, "20260417-b") {
		t.Errorf("text list missing items: %q", s)
	}
}

func TestPrintItemsTextTruncatesTitleByRunes(t *testing.T) {
	var out bytes.Buffer
	p := output.New(&out, &bytes.Buffer{}, false, false)
	long := strings.Repeat("チ", 50)
	item := makeItem("20260417-a", model.KindInbox)
	item.Title = long
	p.PrintItems([]*model.Item{item})

	s := out.String()
	if !utf8.ValidString(s) {
		t.Errorf("output is not valid UTF-8: %q", s)
	}
	if !strings.Contains(s, "...") {
		t.Errorf("long title should be truncated with ellipsis: %q", s)
	}
}

func TestPrintItemsTextAlignsLongIDs(t *testing.T) {
	var out bytes.Buffer
	p := output.New(&out, &bytes.Buffer{}, false, false)
	longID := "20260417-" + strings.Repeat("x", 60)
	items := []*model.Item{
		makeItem(longID, model.KindInbox),
		makeItem("20260417-short", model.KindProject),
	}
	p.PrintItems(items)

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("want 3 lines (header + 2 rows), got %d: %q", len(lines), out.String())
	}
	if !strings.Contains(lines[1], longID) {
		t.Errorf("long ID row missing ID: %q", lines[1])
	}
	// KIND column must start at the same offset across header and every row.
	headerKind := strings.Index(lines[0], "KIND")
	row1Kind := strings.Index(lines[1], "inbox")
	row2Kind := strings.Index(lines[2], "project")
	if headerKind != row1Kind || headerKind != row2Kind {
		t.Errorf("KIND column offsets must match: header=%d row1=%d row2=%d\n%s",
			headerKind, row1Kind, row2Kind, out.String())
	}
}

func TestPrintItemsJSON(t *testing.T) {
	var out bytes.Buffer
	p := output.New(&out, &bytes.Buffer{}, true, false)
	items := []*model.Item{
		makeItem("20260417-x", model.KindInbox),
	}
	p.PrintItems(items)

	var arr []map[string]any
	if err := json.Unmarshal(out.Bytes(), &arr); err != nil {
		t.Fatalf("invalid JSON array: %v\noutput: %s", err, out.String())
	}
	if len(arr) != 1 {
		t.Errorf("JSON array length: want 1, got %d", len(arr))
	}
}

func TestPrintError(t *testing.T) {
	var errOut bytes.Buffer
	p := output.New(&bytes.Buffer{}, &errOut, false, false)
	p.PrintError("something went wrong")

	if !strings.Contains(errOut.String(), "something went wrong") {
		t.Errorf("PrintError: expected error in stderr, got %q", errOut.String())
	}
}

func TestPrintID(t *testing.T) {
	var out bytes.Buffer
	p := output.New(&out, &bytes.Buffer{}, false, false)
	p.PrintID("20260417-new_item")

	if strings.TrimSpace(out.String()) != "20260417-new_item" {
		t.Errorf("PrintID: want %q, got %q", "20260417-new_item", out.String())
	}
}

func TestPrintPathsText(t *testing.T) {
	var out bytes.Buffer
	p := output.New(&out, &bytes.Buffer{}, false, false)
	p.PrintPaths([]string{"items/inbox", "items/project", "reference"})

	want := "items/inbox\nitems/project\nreference\n"
	if got := out.String(); got != want {
		t.Errorf("PrintPaths text: want %q, got %q", want, got)
	}
}

func TestPrintPathsJSON(t *testing.T) {
	var out bytes.Buffer
	p := output.New(&out, &bytes.Buffer{}, true, false)
	p.PrintPaths([]string{"items/inbox", "reference"})

	var arr []string
	if err := json.Unmarshal(out.Bytes(), &arr); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out.String())
	}
	if len(arr) != 2 || arr[0] != "items/inbox" || arr[1] != "reference" {
		t.Errorf("PrintPaths JSON: got %v", arr)
	}
}

func TestPrintUpdatesSilentByDefault(t *testing.T) {
	var out bytes.Buffer
	p := output.New(&out, &bytes.Buffer{}, false, false)
	item := makeItem("20260417-mute", model.KindInbox)
	p.PrintUpdates([]output.Update{{
		Item:    item,
		Changes: []output.Change{{Key: "status", Value: "done"}},
	}})
	if out.Len() != 0 {
		t.Errorf("PrintUpdates without --verbose should be silent, got %q", out.String())
	}
}

func TestPrintUpdatesVerboseText(t *testing.T) {
	var out bytes.Buffer
	p := output.New(&out, &bytes.Buffer{}, false, true)
	item := makeItem("20260417-v", model.KindInbox)
	p.PrintUpdates([]output.Update{{
		Item: item,
		Changes: []output.Change{
			{Key: "kind", Value: "next_action"},
			{Key: "due_at", Value: "2026-05-01T00:00:00Z"},
		},
	}})
	want := "updated 20260417-v: kind=next_action due_at=2026-05-01T00:00:00Z\n"
	if got := out.String(); got != want {
		t.Errorf("verbose text: want %q, got %q", want, got)
	}
}

func TestPrintUpdatesVerboseTextBulk(t *testing.T) {
	var out bytes.Buffer
	p := output.New(&out, &bytes.Buffer{}, false, true)
	a := makeItem("20260417-a", model.KindInbox)
	b := makeItem("20260417-b", model.KindInbox)
	p.PrintUpdates([]output.Update{
		{Item: a, Changes: []output.Change{{Key: "status", Value: "done"}}},
		{Item: b, Changes: []output.Change{{Key: "status", Value: "done"}}},
	})
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 lines, got %d: %q", len(lines), out.String())
	}
	if !strings.HasPrefix(lines[0], "updated 20260417-a:") {
		t.Errorf("line 0: %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "updated 20260417-b:") {
		t.Errorf("line 1: %q", lines[1])
	}
}

func TestPrintUpdatesVerboseJSON(t *testing.T) {
	var out bytes.Buffer
	p := output.New(&out, &bytes.Buffer{}, true, true)
	a := makeItem("20260417-a", model.KindInbox)
	b := makeItem("20260417-b", model.KindProject)
	p.PrintUpdates([]output.Update{
		{Item: a, Changes: []output.Change{{Key: "status", Value: "done"}}},
		{Item: b, Changes: []output.Change{{Key: "status", Value: "done"}}},
	})
	var arr []map[string]any
	if err := json.Unmarshal(out.Bytes(), &arr); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out.String())
	}
	if len(arr) != 2 {
		t.Fatalf("want 2 items in array, got %d", len(arr))
	}
	if arr[0]["id"] != "20260417-a" || arr[1]["id"] != "20260417-b" {
		t.Errorf("JSON IDs: got %v", arr)
	}
}

func TestFormatTimePtr(t *testing.T) {
	if got := output.FormatTimePtr(nil); got != "" {
		t.Errorf("nil: want %q, got %q", "", got)
	}
	ts := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	if got := output.FormatTimePtr(&ts); got != "2026-05-01T00:00:00Z" {
		t.Errorf("ts: got %q", got)
	}
}

func TestFormatList(t *testing.T) {
	if got := output.FormatList(nil); got != "[]" {
		t.Errorf("nil: want %q, got %q", "[]", got)
	}
	if got := output.FormatList([]string{"a", "b", "c"}); got != "[a,b,c]" {
		t.Errorf("a,b,c: got %q", got)
	}
}
