package command_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/takai/htd/internal/config"
	"github.com/takai/htd/internal/model"
	"github.com/takai/htd/internal/store"
)

func writeReference(t *testing.T, dir, tool string, ref *model.Reference, body string) string {
	t.Helper()
	cfg := config.New(dir)
	p := store.PathForReferenceActive(cfg, tool, ref.ID)
	if err := store.WriteRef(p, ref, body); err != nil {
		t.Fatalf("WriteRef: %v", err)
	}
	return p
}

// ---------- reference add ----------

func TestReferenceAdd(t *testing.T) {
	dir := setupDir(t)
	out, _, err := runCmd(t, dir, "reference", "add", "--title", "Branch + PR workflow")
	if err != nil {
		t.Fatalf("reference add: %v", err)
	}
	refID := strings.TrimSpace(out)
	if refID == "" {
		t.Fatal("expected non-empty ID output")
	}
	cfg := config.New(dir)
	path := store.PathForReferenceActive(cfg, "claude", refID)
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file missing at %s: %v", path, err)
	}
	idx := filepath.Join(cfg.ReferenceToolDir("claude"), "INDEX.md")
	if _, err := os.Stat(idx); err != nil {
		t.Errorf("INDEX.md missing: %v", err)
	}
}

func TestReferenceAddRequiresTitle(t *testing.T) {
	dir := setupDir(t)
	_, _, err := runCmd(t, dir, "reference", "add")
	if err == nil {
		t.Fatal("expected error when --title is missing")
	}
}

func TestReferenceAddCustomToolLazyCreatesDir(t *testing.T) {
	dir := setupDir(t)
	out, _, err := runCmd(t, dir, "reference", "add", "--tool", "cursor", "--title", "Tool note")
	if err != nil {
		t.Fatalf("reference add: %v", err)
	}
	refID := strings.TrimSpace(out)
	cfg := config.New(dir)
	if _, err := os.Stat(store.PathForReferenceActive(cfg, "cursor", refID)); err != nil {
		t.Errorf("file missing in cursor tool dir: %v", err)
	}
}

func TestReferenceAddIDCollisionWithItem(t *testing.T) {
	dir := setupDir(t)
	// Capture an item, then add a reference with a colliding-by-title slug
	// on the same date — it should land with a _2 suffix.
	_, _, err := runCmd(t, dir, "capture", "add", "--title", "Shared title")
	if err != nil {
		t.Fatal(err)
	}
	out, _, err := runCmd(t, dir, "reference", "add", "--title", "Shared title")
	if err != nil {
		t.Fatalf("reference add: %v", err)
	}
	refID := strings.TrimSpace(out)
	if !strings.HasSuffix(refID, "_2") {
		t.Errorf("expected _2 suffix on cross-data-type collision, got %q", refID)
	}
}

func TestReferenceAddJSONIndexHasTypeSection(t *testing.T) {
	dir := setupDir(t)
	out, _, err := runCmd(t, dir, "reference", "add",
		"--title", "Some user fact",
		"--body", "Lead line for index.",
		"--tag", "type:user",
	)
	if err != nil {
		t.Fatal(err)
	}
	refID := strings.TrimSpace(out)
	cfg := config.New(dir)
	idx := filepath.Join(cfg.ReferenceToolDir("claude"), "INDEX.md")
	data, err := os.ReadFile(idx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "## user") {
		t.Errorf("expected ## user section in INDEX.md:\n%s", data)
	}
	want := "- [Some user fact](" + refID + ".md) — Lead line for index."
	if !strings.Contains(string(data), want) {
		t.Errorf("expected line %q in INDEX.md:\n%s", want, data)
	}
}

// ---------- reference get ----------

func TestReferenceGet(t *testing.T) {
	dir := setupDir(t)
	now := time.Now()
	ref := &model.Reference{ID: "r1", Title: "Hello", CreatedAt: now, UpdatedAt: now}
	writeReference(t, dir, "claude", ref, "body line")

	out, _, err := runCmd(t, dir, "reference", "get", "r1")
	if err != nil {
		t.Fatalf("reference get: %v", err)
	}
	if !strings.Contains(out, "id:         r1") {
		t.Errorf("expected id line, got:\n%s", out)
	}
	if !strings.Contains(out, "tool:       claude") {
		t.Errorf("expected tool line, got:\n%s", out)
	}
	if strings.Contains(out, "(archived)") {
		t.Errorf("active ref should not show (archived):\n%s", out)
	}
}

func TestReferenceGetJSON(t *testing.T) {
	dir := setupDir(t)
	now := time.Now()
	ref := &model.Reference{ID: "r1", Title: "Hello", CreatedAt: now, UpdatedAt: now, Tags: []string{"type:user"}}
	writeReference(t, dir, "claude", ref, "body")

	out, _, err := runCmd(t, dir, "--json", "reference", "get", "r1")
	if err != nil {
		t.Fatalf("reference get --json: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("json unmarshal: %v\n%s", err, out)
	}
	if got["id"] != "r1" {
		t.Errorf("id: %v", got["id"])
	}
	if got["tool"] != "claude" {
		t.Errorf("tool: %v", got["tool"])
	}
	if _, exists := got["archived"]; exists {
		t.Errorf("archived key should be omitted for active hits, got: %v", got["archived"])
	}
}

func TestReferenceGetNotFound(t *testing.T) {
	dir := setupDir(t)
	_, _, err := runCmd(t, dir, "reference", "get", "missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if !store.IsNotFound(err) {
		t.Errorf("expected NotFoundError, got %T %v", err, err)
	}
}

// ---------- reference list ----------

func TestReferenceListBasic(t *testing.T) {
	dir := setupDir(t)
	t0 := time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC)
	for _, id := range []string{"a", "b", "c"} {
		ref := &model.Reference{ID: id, Title: id, CreatedAt: t0, UpdatedAt: t0}
		writeReference(t, dir, "claude", ref, "")
	}

	out, _, err := runCmd(t, dir, "reference", "list")
	if err != nil {
		t.Fatalf("reference list: %v", err)
	}
	for _, id := range []string{"a", "b", "c"} {
		if !strings.Contains(out, id) {
			t.Errorf("expected %q in output:\n%s", id, out)
		}
	}
}

func TestReferenceListIsToolScoped(t *testing.T) {
	dir := setupDir(t)
	now := time.Now()
	writeReference(t, dir, "claude", &model.Reference{ID: "c1", Title: "C", CreatedAt: now, UpdatedAt: now}, "")
	writeReference(t, dir, "cursor", &model.Reference{ID: "u1", Title: "U", CreatedAt: now, UpdatedAt: now}, "")

	out, _, err := runCmd(t, dir, "reference", "list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "c1") {
		t.Errorf("missing claude entry:\n%s", out)
	}
	if strings.Contains(out, "u1") {
		t.Errorf("cursor entry leaked into claude listing:\n%s", out)
	}
}

func TestReferenceListByTag(t *testing.T) {
	dir := setupDir(t)
	now := time.Now()
	writeReference(t, dir, "claude", &model.Reference{ID: "tagged", Title: "T", CreatedAt: now, UpdatedAt: now, Tags: []string{"type:user"}}, "")
	writeReference(t, dir, "claude", &model.Reference{ID: "untagged", Title: "U", CreatedAt: now, UpdatedAt: now}, "")

	out, _, err := runCmd(t, dir, "reference", "list", "--tag", "type:user")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "tagged") {
		t.Errorf("expected tagged in output:\n%s", out)
	}
	if strings.Contains(out, "untagged") {
		t.Errorf("untagged should be excluded:\n%s", out)
	}
}

// ---------- reference update ----------

func TestReferenceUpdateTitle(t *testing.T) {
	dir := setupDir(t)
	now := time.Now()
	writeReference(t, dir, "claude", &model.Reference{ID: "r1", Title: "Old", CreatedAt: now, UpdatedAt: now}, "")
	if _, _, err := runCmd(t, dir, "reference", "update", "r1", "title=New"); err != nil {
		t.Fatalf("update: %v", err)
	}
	cfg := config.New(dir)
	res, err := store.FindReference(cfg, "r1")
	if err != nil {
		t.Fatal(err)
	}
	ref, _, err := store.ReadRef(res.Path)
	if err != nil {
		t.Fatal(err)
	}
	if ref.Title != "New" {
		t.Errorf("title: want New, got %q", ref.Title)
	}
}

func TestReferenceUpdateRegroupsIndex(t *testing.T) {
	dir := setupDir(t)
	now := time.Now()
	writeReference(t, dir, "claude",
		&model.Reference{ID: "r1", Title: "Hello", CreatedAt: now, UpdatedAt: now, Tags: []string{"type:user"}},
		"lead line",
	)
	cfg := config.New(dir)
	if err := store.WriteIndex(cfg, "claude"); err != nil {
		t.Fatal(err)
	}

	if _, _, err := runCmd(t, dir, "reference", "update", "r1", "tags=[type:project]"); err != nil {
		t.Fatalf("update: %v", err)
	}
	idx := filepath.Join(cfg.ReferenceToolDir("claude"), "INDEX.md")
	data, err := os.ReadFile(idx)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "## user") {
		t.Errorf("user section should be empty after retag:\n%s", data)
	}
	if !strings.Contains(string(data), "## project") {
		t.Errorf("project section should appear:\n%s", data)
	}
}

func TestReferenceUpdateRejectsProtectedField(t *testing.T) {
	dir := setupDir(t)
	now := time.Now()
	writeReference(t, dir, "claude", &model.Reference{ID: "r1", Title: "T", CreatedAt: now, UpdatedAt: now}, "")
	if _, _, err := runCmd(t, dir, "reference", "update", "r1", "id=other"); err == nil {
		t.Fatal("expected error on protected field id")
	}
	if _, _, err := runCmd(t, dir, "reference", "update", "r1", "tool=other"); err == nil {
		t.Fatal("expected error on protected field tool")
	}
}

func TestReferenceUpdateVerbose(t *testing.T) {
	dir := setupDir(t)
	now := time.Now()
	writeReference(t, dir, "claude", &model.Reference{ID: "r1", Title: "Old", CreatedAt: now, UpdatedAt: now}, "")
	out, _, err := runCmd(t, dir, "--verbose", "reference", "update", "r1", "title=New")
	if err != nil {
		t.Fatalf("update --verbose: %v", err)
	}
	if !strings.Contains(out, "updated r1: title=New") {
		t.Errorf("expected verbose line, got:\n%s", out)
	}
}

// ---------- reference reindex ----------

func TestReferenceReindex(t *testing.T) {
	dir := setupDir(t)
	now := time.Now()
	writeReference(t, dir, "claude", &model.Reference{ID: "r1", Title: "T", CreatedAt: now, UpdatedAt: now, Tags: []string{"type:user"}}, "fact")

	cfg := config.New(dir)
	idx := filepath.Join(cfg.ReferenceToolDir("claude"), "INDEX.md")
	if err := os.WriteFile(idx, []byte("# garbage manual edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runCmd(t, dir, "reference", "reindex"); err != nil {
		t.Fatalf("reindex: %v", err)
	}
	data, err := os.ReadFile(idx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "## user") {
		t.Errorf("reindex should restore canonical index:\n%s", data)
	}
}

func TestReferenceReindexIdempotent(t *testing.T) {
	dir := setupDir(t)
	now := time.Now()
	writeReference(t, dir, "claude", &model.Reference{ID: "r1", Title: "T", CreatedAt: now, UpdatedAt: now, Tags: []string{"type:user"}}, "fact")

	if _, _, err := runCmd(t, dir, "reference", "reindex"); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(dir)
	idx := filepath.Join(cfg.ReferenceToolDir("claude"), "INDEX.md")
	first, err := os.ReadFile(idx)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := runCmd(t, dir, "reference", "reindex"); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(idx)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Errorf("non-idempotent reindex\nfirst:  %q\nsecond: %q", first, second)
	}
}
