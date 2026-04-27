package store_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/takai/htd/internal/model"
	"github.com/takai/htd/internal/store"
)

func makeRef(id, title string, updated time.Time) *model.Reference {
	return &model.Reference{
		ID:        id,
		Title:     title,
		CreatedAt: updated,
		UpdatedAt: updated,
	}
}

func TestRefWriteRead(t *testing.T) {
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureReferenceToolDir(cfg, "claude"); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC)
	ref := &model.Reference{
		ID:        "r1",
		Title:     "How to commit",
		CreatedAt: now,
		UpdatedAt: now,
		Tags:      []string{"type:feedback", "git"},
	}
	body := "Always commit on a branch.\n\n## How to apply\n\nRun gh pr create."
	path := store.PathForReferenceActive(cfg, "claude", ref.ID)
	if err := store.WriteRef(path, ref, body); err != nil {
		t.Fatalf("WriteRef: %v", err)
	}

	gotRef, gotBody, err := store.ReadRef(path)
	if err != nil {
		t.Fatalf("ReadRef: %v", err)
	}
	if gotRef.ID != ref.ID {
		t.Errorf("ID: want %q, got %q", ref.ID, gotRef.ID)
	}
	if gotRef.Title != ref.Title {
		t.Errorf("Title: want %q, got %q", ref.Title, gotRef.Title)
	}
	if len(gotRef.Tags) != 2 || gotRef.Tags[0] != "type:feedback" {
		t.Errorf("Tags: want [type:feedback git], got %v", gotRef.Tags)
	}
	if gotBody != body {
		t.Errorf("Body mismatch\nwant: %q\n got: %q", body, gotBody)
	}
}

func TestRefWriteCreatesToolDirLazily(t *testing.T) {
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC)
	ref := makeRef("r1", "T", now)
	path := store.PathForReferenceActive(cfg, "newtool", ref.ID)
	// Note: do NOT call EnsureReferenceToolDir; WriteRef should create the dir.
	if err := store.WriteRef(path, ref, ""); err != nil {
		t.Fatalf("WriteRef: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		t.Errorf("tool dir missing after WriteRef: %v", err)
	}
}

func TestFindReferenceActive(t *testing.T) {
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC)
	ref := makeRef("r1", "T", now)
	path := store.PathForReferenceActive(cfg, "claude", ref.ID)
	if err := store.WriteRef(path, ref, ""); err != nil {
		t.Fatalf("WriteRef: %v", err)
	}

	got, err := store.FindReference(cfg, "r1")
	if err != nil {
		t.Fatalf("FindReference: %v", err)
	}
	if got.Path != path {
		t.Errorf("Path: want %q, got %q", path, got.Path)
	}
	if got.Tool != "claude" {
		t.Errorf("Tool: want claude, got %q", got.Tool)
	}
	if got.Archived {
		t.Error("Archived: want false, got true")
	}
}

func TestFindReferenceArchive(t *testing.T) {
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC)
	ref := makeRef("r2", "T", now)
	path := store.PathForReferenceArchive(cfg, "claude", ref.ID)
	if err := store.WriteRef(path, ref, ""); err != nil {
		t.Fatalf("WriteRef: %v", err)
	}

	got, err := store.FindReference(cfg, "r2")
	if err != nil {
		t.Fatalf("FindReference archive: %v", err)
	}
	if got.Path != path {
		t.Errorf("Path: want %q, got %q", path, got.Path)
	}
	if !got.Archived {
		t.Error("Archived: want true, got false")
	}
}

func TestFindReferencePrefersActiveOverArchive(t *testing.T) {
	// Pathological case (shouldn't happen in normal use); active wins.
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC)
	ref := makeRef("dup", "T", now)
	active := store.PathForReferenceActive(cfg, "claude", ref.ID)
	arch := store.PathForReferenceArchive(cfg, "claude", ref.ID)
	if err := store.WriteRef(active, ref, ""); err != nil {
		t.Fatal(err)
	}
	if err := store.WriteRef(arch, ref, ""); err != nil {
		t.Fatal(err)
	}

	got, err := store.FindReference(cfg, "dup")
	if err != nil {
		t.Fatalf("FindReference: %v", err)
	}
	if got.Path != active || got.Archived {
		t.Errorf("expected active hit, got %+v", got)
	}
}

func TestFindReferenceNotFound(t *testing.T) {
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatal(err)
	}
	_, err := store.FindReference(cfg, "ghost")
	if err == nil {
		t.Fatal("FindReference: expected error, got nil")
	}
	if !store.IsNotFound(err) {
		t.Errorf("FindReference: want NotFoundError, got %T %v", err, err)
	}
	nfe, ok := err.(*store.NotFoundError)
	if !ok {
		t.Fatalf("expected *store.NotFoundError, got %T", err)
	}
	if nfe.Kind != store.EntityReference {
		t.Errorf("Kind: want %q, got %q", store.EntityReference, nfe.Kind)
	}
}

func TestListReferencesSortAndScope(t *testing.T) {
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatal(err)
	}
	t0 := time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC)
	refs := []*model.Reference{
		makeRef("a", "A", t0),
		makeRef("b", "B", t0.Add(2*time.Hour)),
		makeRef("c", "C", t0.Add(time.Hour)),
	}
	for _, r := range refs {
		if err := store.WriteRef(store.PathForReferenceActive(cfg, "claude", r.ID), r, ""); err != nil {
			t.Fatal(err)
		}
	}
	// Different tool — must not appear.
	other := makeRef("z", "Z", t0.Add(3*time.Hour))
	if err := store.WriteRef(store.PathForReferenceActive(cfg, "other", other.ID), other, ""); err != nil {
		t.Fatal(err)
	}
	// Archived — must not appear unless includeArchive.
	arch := makeRef("d", "D", t0.Add(4*time.Hour))
	if err := store.WriteRef(store.PathForReferenceArchive(cfg, "claude", arch.ID), arch, ""); err != nil {
		t.Fatal(err)
	}

	got, err := store.ListReferences(cfg, "claude", false)
	if err != nil {
		t.Fatalf("ListReferences: %v", err)
	}
	wantIDs := []string{"b", "c", "a"} // sorted by updated desc
	if len(got) != len(wantIDs) {
		t.Fatalf("len: want %d, got %d", len(wantIDs), len(got))
	}
	for i, id := range wantIDs {
		if got[i].Reference.ID != id {
			t.Errorf("idx %d: want %q, got %q", i, id, got[i].Reference.ID)
		}
		if got[i].Tool != "claude" {
			t.Errorf("idx %d: tool want claude, got %q", i, got[i].Tool)
		}
		if got[i].Archived {
			t.Errorf("idx %d: archived should be false", i)
		}
	}

	// With archive included.
	got, err = store.ListReferences(cfg, "claude", true)
	if err != nil {
		t.Fatalf("ListReferences includeArchive: %v", err)
	}
	if len(got) != 4 {
		t.Errorf("includeArchive len: want 4, got %d", len(got))
	}
}

func TestListReferencesTiebreakByID(t *testing.T) {
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatal(err)
	}
	tt := time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC)
	for _, id := range []string{"zzz", "aaa", "mmm"} {
		if err := store.WriteRef(store.PathForReferenceActive(cfg, "claude", id), makeRef(id, id, tt), ""); err != nil {
			t.Fatal(err)
		}
	}
	got, err := store.ListReferences(cfg, "claude", false)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"aaa", "mmm", "zzz"}
	for i, id := range want {
		if got[i].Reference.ID != id {
			t.Errorf("tiebreak idx %d: want %q, got %q", i, id, got[i].Reference.ID)
		}
	}
}

func TestListReferencesIgnoresIndexFile(t *testing.T) {
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureReferenceToolDir(cfg, "claude"); err != nil {
		t.Fatal(err)
	}
	// Create a stray INDEX.md (wrong format) and verify ListReferences
	// does not try to parse it as a reference.
	idx := filepath.Join(cfg.ReferenceToolDir("claude"), "INDEX.md")
	if err := os.WriteFile(idx, []byte("# Reference index\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := store.ListReferences(cfg, "claude", false)
	if err != nil {
		t.Fatalf("ListReferences: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("len: want 0, got %d", len(got))
	}
}

func TestListReferenceToolsDiscoversSubdirs(t *testing.T) {
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatal(err)
	}
	for _, tool := range []string{"claude", "cursor", "aider"} {
		if err := store.EnsureReferenceToolDir(cfg, tool); err != nil {
			t.Fatal(err)
		}
	}
	got, err := store.ListReferenceTools(cfg)
	if err != nil {
		t.Fatalf("ListReferenceTools: %v", err)
	}
	want := []string{"aider", "claude", "cursor"}
	if len(got) != len(want) {
		t.Fatalf("len: want %d, got %d (%v)", len(want), len(got), got)
	}
	for i, tool := range want {
		if got[i] != tool {
			t.Errorf("idx %d: want %q, got %q", i, tool, got[i])
		}
	}
}

func TestMoveRef(t *testing.T) {
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC)
	ref := makeRef("m1", "T", now)
	src := store.PathForReferenceActive(cfg, "claude", ref.ID)
	if err := store.WriteRef(src, ref, "body"); err != nil {
		t.Fatal(err)
	}
	dst := store.PathForReferenceArchive(cfg, "claude", ref.ID)
	if err := store.MoveRef(src, dst, ref, "body"); err != nil {
		t.Fatalf("MoveRef: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("src still exists after MoveRef")
	}
	if _, err := os.Stat(dst); err != nil {
		t.Errorf("dst missing after MoveRef: %v", err)
	}
}
