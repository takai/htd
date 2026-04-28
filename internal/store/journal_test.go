package store_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/takai/htd/internal/model"
	"github.com/takai/htd/internal/store"
)

func TestJournalWriteRead(t *testing.T) {
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 4, 28, 9, 0, 0, 0, time.UTC)
	j := &model.Journal{
		CreatedAt: now,
		UpdatedAt: now,
		Tags:      []string{"daily"},
	}
	body := "# 2026-04-28\n\n## What I did"
	path := store.PathForJournal(cfg, "2026-04-28")
	if err := store.WriteJournal(path, j, body); err != nil {
		t.Fatalf("WriteJournal: %v", err)
	}

	gotJ, gotBody, err := store.ReadJournal(path)
	if err != nil {
		t.Fatalf("ReadJournal: %v", err)
	}
	if !gotJ.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt: want %v, got %v", now, gotJ.CreatedAt)
	}
	if len(gotJ.Tags) != 1 || gotJ.Tags[0] != "daily" {
		t.Errorf("Tags: want [daily], got %v", gotJ.Tags)
	}
	if gotBody != body {
		t.Errorf("Body mismatch\nwant: %q\n got: %q", body, gotBody)
	}
}

func TestJournalReadNoFrontmatter(t *testing.T) {
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatal(err)
	}
	path := store.PathForJournal(cfg, "plain")
	body := "# Just a hand-edited note\n\nNo YAML."
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	j, gotBody, err := store.ReadJournal(path)
	if err != nil {
		t.Fatalf("ReadJournal plain: %v", err)
	}
	if !j.CreatedAt.IsZero() || !j.UpdatedAt.IsZero() || len(j.Tags) > 0 {
		t.Errorf("expected zero-valued metadata, got %+v", j)
	}
	if gotBody != body {
		t.Errorf("Body mismatch\nwant: %q\n got: %q", body, gotBody)
	}
}

func TestJournalWriteNilSkipsFrontmatter(t *testing.T) {
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatal(err)
	}
	path := store.PathForJournal(cfg, "plain")
	if err := store.WriteJournal(path, nil, "# Hi\n"); err != nil {
		t.Fatalf("WriteJournal nil: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(string(data), "---") {
		t.Errorf("expected plain Markdown when journal is nil, got:\n%s", data)
	}
}

func TestJournalFindMissing(t *testing.T) {
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatal(err)
	}
	_, err := store.FindJournal(cfg, "ghost")
	if err == nil {
		t.Fatal("expected error for missing journal")
	}
	if !store.IsNotFound(err) {
		t.Errorf("expected NotFoundError, got %T %v", err, err)
	}
	nfe, ok := err.(*store.NotFoundError)
	if !ok {
		t.Fatalf("expected *store.NotFoundError, got %T", err)
	}
	if nfe.Kind != store.EntityJournal {
		t.Errorf("Kind: want %q, got %q", store.EntityJournal, nfe.Kind)
	}
}

func TestJournalListSortedDesc(t *testing.T) {
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"2026-04-26", "2026-04-28", "2026-04-27"} {
		if err := store.WriteJournal(store.PathForJournal(cfg, name), nil, "# "+name+"\n"); err != nil {
			t.Fatal(err)
		}
	}
	got, err := store.ListJournals(cfg, time.Time{})
	if err != nil {
		t.Fatalf("ListJournals: %v", err)
	}
	want := []string{"2026-04-28", "2026-04-27", "2026-04-26"}
	if len(got) != len(want) {
		t.Fatalf("len: want %d, got %d", len(want), len(got))
	}
	for i, name := range want {
		if got[i].Name != name {
			t.Errorf("idx %d: want %q, got %q", i, name, got[i].Name)
		}
	}
}

func TestJournalListSinceFiltersByName(t *testing.T) {
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"2026-04-25", "2026-04-27", "2026-04-29"} {
		if err := store.WriteJournal(store.PathForJournal(cfg, name), nil, "# "+name+"\n"); err != nil {
			t.Fatal(err)
		}
	}
	since := time.Date(2026, 4, 27, 0, 0, 0, 0, time.Local)
	got, err := store.ListJournals(cfg, since)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"2026-04-29", "2026-04-27"}
	if len(got) != len(want) {
		t.Fatalf("len: want %d, got %d (%v)", len(want), len(got), got)
	}
	for i, name := range want {
		if got[i].Name != name {
			t.Errorf("idx %d: want %q, got %q", i, name, got[i].Name)
		}
	}
}

func TestJournalListIgnoresSubdirsAndNonMarkdown(t *testing.T) {
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatal(err)
	}
	if err := store.WriteJournal(store.PathForJournal(cfg, "2026-04-28"), nil, "# x\n"); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(cfg.JournalDir(), "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.JournalDir(), "notes.txt"), []byte("ignore"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := store.ListJournals(cfg, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "2026-04-28" {
		t.Errorf("unexpected list: %+v", got)
	}
}

func TestJournalEnsureDirsIncludesJournal(t *testing.T) {
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cfg.JournalDir()); err != nil {
		t.Errorf("journal dir missing after EnsureDirs: %v", err)
	}
}
