package model_test

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/takai/htd/internal/model"
)

func TestKindConstants(t *testing.T) {
	kinds := model.ValidKinds()
	expected := []model.Kind{
		model.KindInbox, model.KindNextAction, model.KindProject,
		model.KindWaitingFor, model.KindSomeday, model.KindTickler,
	}
	if len(kinds) != len(expected) {
		t.Fatalf("want %d kinds, got %d", len(expected), len(kinds))
	}
	for i, k := range expected {
		if kinds[i] != k {
			t.Errorf("kinds[%d]: want %q, got %q", i, k, kinds[i])
		}
	}
}

func TestStatusConstants(t *testing.T) {
	statuses := model.ValidStatuses()
	expected := []model.Status{
		model.StatusActive, model.StatusDone, model.StatusCanceled,
		model.StatusDiscarded, model.StatusArchived,
	}
	if len(statuses) != len(expected) {
		t.Fatalf("want %d statuses, got %d", len(expected), len(statuses))
	}
	for i, s := range expected {
		if statuses[i] != s {
			t.Errorf("statuses[%d]: want %q, got %q", i, s, statuses[i])
		}
	}
}

func TestIsTerminal(t *testing.T) {
	cases := []struct {
		status   model.Status
		terminal bool
	}{
		{model.StatusActive, false},
		{model.StatusDone, true},
		{model.StatusCanceled, true},
		{model.StatusDiscarded, true},
		{model.StatusArchived, true},
	}
	for _, c := range cases {
		if got := model.IsTerminal(c.status); got != c.terminal {
			t.Errorf("IsTerminal(%q) = %v, want %v", c.status, got, c.terminal)
		}
	}
}

func TestIsActive(t *testing.T) {
	if !model.IsActive(model.StatusActive) {
		t.Error("IsActive(StatusActive) = false, want true")
	}
	for _, s := range []model.Status{model.StatusDone, model.StatusCanceled, model.StatusDiscarded, model.StatusArchived} {
		if model.IsActive(s) {
			t.Errorf("IsActive(%q) = true, want false", s)
		}
	}
}

func TestItemYAMLRoundTrip(t *testing.T) {
	due := time.Date(2026, 4, 17, 15, 0, 0, 0, time.FixedZone("JST", 9*3600))
	defer_ := time.Date(2026, 4, 20, 0, 0, 0, 0, time.FixedZone("JST", 9*3600))
	review := time.Date(2026, 4, 25, 0, 0, 0, 0, time.FixedZone("JST", 9*3600))
	item := &model.Item{
		ID:          "20260417-test_item",
		Title:       "Test Item",
		Kind:        model.KindNextAction,
		Status:      model.StatusActive,
		Project:     "20260417-my_project",
		CreatedAt:   time.Date(2026, 4, 17, 9, 0, 0, 0, time.FixedZone("JST", 9*3600)),
		UpdatedAt:   time.Date(2026, 4, 17, 9, 30, 0, 0, time.FixedZone("JST", 9*3600)),
		DueAt:       &due,
		DeferUntil:  &defer_,
		ReviewAt:    &review,
		Source:      "manual",
		Tags:        []string{"cli", "docs"},
		Refs:        []string{"https://example.com/pr/1", "https://notion.so/x"},
	}

	data, err := yaml.Marshal(item)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got model.Item
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.ID != item.ID {
		t.Errorf("ID: want %q, got %q", item.ID, got.ID)
	}
	if got.Title != item.Title {
		t.Errorf("Title: want %q, got %q", item.Title, got.Title)
	}
	if got.Kind != item.Kind {
		t.Errorf("Kind: want %q, got %q", item.Kind, got.Kind)
	}
	if got.Status != item.Status {
		t.Errorf("Status: want %q, got %q", item.Status, got.Status)
	}
	if got.Project != item.Project {
		t.Errorf("Project: want %q, got %q", item.Project, got.Project)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "cli" || got.Tags[1] != "docs" {
		t.Errorf("Tags: want [cli docs], got %v", got.Tags)
	}
	if len(got.Refs) != 2 || got.Refs[0] != "https://example.com/pr/1" || got.Refs[1] != "https://notion.so/x" {
		t.Errorf("Refs: want [https://example.com/pr/1 https://notion.so/x], got %v", got.Refs)
	}
	if got.DueAt == nil {
		t.Error("DueAt: want non-nil, got nil")
	}
	if got.DeferUntil == nil {
		t.Error("DeferUntil: want non-nil, got nil")
	}
	if got.ReviewAt == nil {
		t.Error("ReviewAt: want non-nil, got nil")
	}
}

func TestItemYAMLOmitsNilOptionalFields(t *testing.T) {
	item := &model.Item{
		ID:        "20260417-minimal",
		Title:     "Minimal",
		Kind:      model.KindInbox,
		Status:    model.StatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	data, err := yaml.Marshal(item)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	s := string(data)
	for _, field := range []string{"due_at", "defer_until", "review_at", "project", "source", "tags", "refs"} {
		if contains(s, field+":") {
			t.Errorf("YAML contains %q field but it should be omitted", field)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
