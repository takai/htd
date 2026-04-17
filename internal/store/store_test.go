package store_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/takai/htd/internal/config"
	"github.com/takai/htd/internal/model"
	"github.com/takai/htd/internal/store"
)

func newTestCfg(t *testing.T) *config.Config {
	t.Helper()
	return config.New(t.TempDir())
}

// ---------- EnsureDirs ----------

func TestEnsureDirs(t *testing.T) {
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatalf("EnsureDirs: %v", err)
	}
	for _, d := range cfg.AllDirs() {
		if _, err := os.Stat(d); err != nil {
			t.Errorf("dir %q missing after EnsureDirs: %v", d, err)
		}
	}
}

func TestEnsureDirsIdempotent(t *testing.T) {
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatalf("first EnsureDirs: %v", err)
	}
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatalf("second EnsureDirs: %v", err)
	}
}

// ---------- PathForItem ----------

func TestPathForItemActive(t *testing.T) {
	cfg := newTestCfg(t)
	item := &model.Item{ID: "20260417-test", Kind: model.KindInbox, Status: model.StatusActive}
	want := filepath.Join(cfg.Root, "items", "inbox", "20260417-test.md")
	if got := store.PathForItem(cfg, item); got != want {
		t.Errorf("PathForItem active inbox: want %q, got %q", want, got)
	}
}

func TestPathForItemTerminal(t *testing.T) {
	cfg := newTestCfg(t)
	for _, status := range []model.Status{model.StatusDone, model.StatusCanceled, model.StatusDiscarded, model.StatusArchived} {
		item := &model.Item{ID: "20260417-test", Kind: model.KindNextAction, Status: status}
		want := filepath.Join(cfg.Root, "archive", "items", "20260417-test.md")
		if got := store.PathForItem(cfg, item); got != want {
			t.Errorf("PathForItem %q: want %q, got %q", status, want, got)
		}
	}
}

// ---------- FindItem ----------

func TestFindItem(t *testing.T) {
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatal(err)
	}

	item := makeItem("20260417-find_me", model.KindProject, model.StatusActive)
	path := store.PathForItem(cfg, item)
	if err := store.Write(path, item, "body text"); err != nil {
		t.Fatalf("Write: %v", err)
	}

	found, err := store.FindItem(cfg, "20260417-find_me")
	if err != nil {
		t.Fatalf("FindItem: %v", err)
	}
	if found != path {
		t.Errorf("FindItem: want %q, got %q", path, found)
	}
}

func TestFindItemArchived(t *testing.T) {
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatal(err)
	}

	item := makeItem("20260417-archived_one", model.KindNextAction, model.StatusDone)
	path := store.PathForItem(cfg, item)
	if err := store.Write(path, item, ""); err != nil {
		t.Fatalf("Write: %v", err)
	}

	found, err := store.FindItem(cfg, "20260417-archived_one")
	if err != nil {
		t.Fatalf("FindItem archived: %v", err)
	}
	if found != path {
		t.Errorf("FindItem archived: want %q, got %q", path, found)
	}
}

func TestFindItemNotFound(t *testing.T) {
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatal(err)
	}
	_, err := store.FindItem(cfg, "20260417-ghost")
	if err == nil {
		t.Fatal("FindItem: expected error for missing item, got nil")
	}
	var nfe *store.NotFoundError
	if !store.IsNotFound(err) {
		t.Errorf("FindItem: expected NotFoundError, got %T: %v", err, err)
	}
	_ = nfe
}

// ---------- Read / Write ----------

func TestWriteRead(t *testing.T) {
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatal(err)
	}

	due := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)
	item := &model.Item{
		ID:        "20260417-rw_test",
		Title:     "Read Write Test",
		Kind:      model.KindNextAction,
		Status:    model.StatusActive,
		CreatedAt: time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC),
		DueAt:     &due,
		Tags:      []string{"a", "b"},
	}
	body := "# Detail\n\nSome content with --- inside."

	path := store.PathForItem(cfg, item)
	if err := store.Write(path, item, body); err != nil {
		t.Fatalf("Write: %v", err)
	}

	gotItem, gotBody, err := store.Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if gotItem.ID != item.ID {
		t.Errorf("ID: want %q, got %q", item.ID, gotItem.ID)
	}
	if gotItem.Title != item.Title {
		t.Errorf("Title: want %q, got %q", item.Title, gotItem.Title)
	}
	if gotItem.DueAt == nil {
		t.Error("DueAt: want non-nil, got nil")
	}
	if len(gotItem.Tags) != 2 {
		t.Errorf("Tags: want 2, got %d", len(gotItem.Tags))
	}
	if gotBody != body {
		t.Errorf("Body: want %q, got %q", body, gotBody)
	}
}

// ---------- Move ----------

func TestMove(t *testing.T) {
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatal(err)
	}

	item := makeItem("20260417-move_me", model.KindInbox, model.StatusActive)
	src := store.PathForItem(cfg, item)
	if err := store.Write(src, item, "move body"); err != nil {
		t.Fatalf("Write: %v", err)
	}

	item.Kind = model.KindNextAction
	dst := store.PathForItem(cfg, item)
	if err := store.Move(src, dst, item, "move body"); err != nil {
		t.Fatalf("Move: %v", err)
	}

	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("src still exists after Move")
	}
	if _, err := os.Stat(dst); err != nil {
		t.Errorf("dst missing after Move: %v", err)
	}

	gotItem, gotBody, err := store.Read(dst)
	if err != nil {
		t.Fatalf("Read dst: %v", err)
	}
	if gotItem.Kind != model.KindNextAction {
		t.Errorf("Kind: want next_action, got %q", gotItem.Kind)
	}
	if gotBody != "move body" {
		t.Errorf("Body: want %q, got %q", "move body", gotBody)
	}
}

// ---------- List ----------

func TestList(t *testing.T) {
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatal(err)
	}

	items := []*model.Item{
		makeItem("20260417-item1", model.KindInbox, model.StatusActive),
		makeItem("20260417-item2", model.KindNextAction, model.StatusActive),
		makeItem("20260417-item3", model.KindProject, model.StatusActive),
		makeItem("20260417-item4", model.KindNextAction, model.StatusDone),
	}
	for _, it := range items {
		p := store.PathForItem(cfg, it)
		if err := store.Write(p, it, ""); err != nil {
			t.Fatalf("Write %q: %v", it.ID, err)
		}
	}

	// All active
	activeKind := model.KindNextAction
	got, err := store.List(cfg, store.Filter{Kind: &activeKind, Status: statusPtr(model.StatusActive)})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 || got[0].ID != "20260417-item2" {
		t.Errorf("List next_action active: got %v", itemIDs(got))
	}

	// All items regardless of status
	got, err = store.List(cfg, store.Filter{})
	if err != nil {
		t.Fatalf("List all: %v", err)
	}
	if len(got) != 4 {
		t.Errorf("List all: want 4, got %d: %v", len(got), itemIDs(got))
	}
}

func TestListByTag(t *testing.T) {
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatal(err)
	}

	it1 := makeItem("20260417-tagged", model.KindInbox, model.StatusActive)
	it1.Tags = []string{"cli", "docs"}
	it2 := makeItem("20260417-untagged", model.KindInbox, model.StatusActive)

	for _, it := range []*model.Item{it1, it2} {
		p := store.PathForItem(cfg, it)
		if err := store.Write(p, it, ""); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}

	got, err := store.List(cfg, store.Filter{Tag: "cli"})
	if err != nil {
		t.Fatalf("List by tag: %v", err)
	}
	if len(got) != 1 || got[0].ID != "20260417-tagged" {
		t.Errorf("List by tag: got %v", itemIDs(got))
	}
}

func TestListByProject(t *testing.T) {
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatal(err)
	}

	it1 := makeItem("20260417-linked", model.KindNextAction, model.StatusActive)
	it1.Project = "20260417-proj"
	it2 := makeItem("20260417-unlinked", model.KindNextAction, model.StatusActive)

	for _, it := range []*model.Item{it1, it2} {
		p := store.PathForItem(cfg, it)
		if err := store.Write(p, it, ""); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}

	got, err := store.List(cfg, store.Filter{ProjectID: "20260417-proj"})
	if err != nil {
		t.Fatalf("List by project: %v", err)
	}
	if len(got) != 1 || got[0].ID != "20260417-linked" {
		t.Errorf("List by project: got %v", itemIDs(got))
	}
}

// ---------- helpers ----------

func makeItem(id string, kind model.Kind, status model.Status) *model.Item {
	now := time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC)
	return &model.Item{
		ID:        id,
		Title:     id,
		Kind:      kind,
		Status:    status,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func statusPtr(s model.Status) *model.Status { return &s }

func itemIDs(items []*model.Item) []string {
	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = it.ID
	}
	return ids
}
