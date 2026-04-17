package output_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

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
	p := output.New(&out, &bytes.Buffer{}, false)
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
	p := output.New(&out, &bytes.Buffer{}, true)
	item := makeItem("20260417-json", model.KindNextAction)
	p.PrintItem(item, "body content")

	var obj map[string]interface{}
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
	p := output.New(&out, &bytes.Buffer{}, false)
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

func TestPrintItemsJSON(t *testing.T) {
	var out bytes.Buffer
	p := output.New(&out, &bytes.Buffer{}, true)
	items := []*model.Item{
		makeItem("20260417-x", model.KindInbox),
	}
	p.PrintItems(items)

	var arr []map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &arr); err != nil {
		t.Fatalf("invalid JSON array: %v\noutput: %s", err, out.String())
	}
	if len(arr) != 1 {
		t.Errorf("JSON array length: want 1, got %d", len(arr))
	}
}

func TestPrintError(t *testing.T) {
	var errOut bytes.Buffer
	p := output.New(&bytes.Buffer{}, &errOut, false)
	p.PrintError("something went wrong")

	if !strings.Contains(errOut.String(), "something went wrong") {
		t.Errorf("PrintError: expected error in stderr, got %q", errOut.String())
	}
}

func TestPrintID(t *testing.T) {
	var out bytes.Buffer
	p := output.New(&out, &bytes.Buffer{}, false)
	p.PrintID("20260417-new_item")

	if strings.TrimSpace(out.String()) != "20260417-new_item" {
		t.Errorf("PrintID: want %q, got %q", "20260417-new_item", out.String())
	}
}
