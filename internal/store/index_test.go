package store_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/takai/htd/internal/model"
	"github.com/takai/htd/internal/store"
)

func refWithBody(id, title, body string, updated time.Time, tags ...string) store.ReferenceWithBody {
	return store.ReferenceWithBody{
		Reference: &model.Reference{
			ID:        id,
			Title:     title,
			CreatedAt: updated,
			UpdatedAt: updated,
			Tags:      tags,
		},
		Body: body,
		Tool: "claude",
	}
}

func TestRenderIndexEmpty(t *testing.T) {
	got := store.RenderIndex(nil)
	want := "# Reference index\n\n_No entries._\n"
	if string(got) != want {
		t.Errorf("empty render mismatch\nwant: %q\n got: %q", want, string(got))
	}
}

func TestRenderIndexDeterministic(t *testing.T) {
	t0 := time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC)
	refs := []store.ReferenceWithBody{
		refWithBody("a", "A", "alpha line", t0, "type:user"),
		refWithBody("b", "B", "beta line", t0.Add(time.Hour), "type:feedback"),
		refWithBody("c", "C", "gamma line", t0.Add(2*time.Hour), "type:user"),
	}
	first := store.RenderIndex(refs)
	second := store.RenderIndex(refs)
	if !bytes.Equal(first, second) {
		t.Errorf("non-deterministic output\nfirst:  %q\nsecond: %q", first, second)
	}
}

func TestRenderIndexGroupingAndSort(t *testing.T) {
	t0 := time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC)
	refs := []store.ReferenceWithBody{
		refWithBody("u_old", "User old", "user old fact", t0, "type:user"),
		refWithBody("u_new", "User new", "user new fact", t0.Add(2*time.Hour), "type:user"),
		refWithBody("f1", "Feedback one", "feedback fact", t0.Add(time.Hour), "type:feedback"),
		refWithBody("p1", "Project one", "project fact", t0, "type:project"),
		refWithBody("r1", "Reference one", "reference fact", t0, "type:reference"),
		refWithBody("o1", "Other one", "other fact", t0, "misc"),
		refWithBody("o2", "Other two", "other2 fact", t0, "type:area_of_focus"),
	}
	got := string(store.RenderIndex(refs))

	// Sections appear in canonical order; each entry's title appears.
	wantOrder := []string{
		"## user",
		"User new", // newer
		"User old",
		"## feedback",
		"Feedback one",
		"## project",
		"Project one",
		"## reference",
		"Reference one",
		"## other",
		"Other one",
		"Other two",
	}
	prev := -1
	for _, marker := range wantOrder {
		idx := strings.Index(got, marker)
		if idx < 0 {
			t.Errorf("missing marker %q in output:\n%s", marker, got)
			continue
		}
		if idx <= prev {
			t.Errorf("marker %q out of order at %d (prev %d)\noutput:\n%s", marker, idx, prev, got)
		}
		prev = idx
	}
}

func TestRenderIndexLineFormat(t *testing.T) {
	t0 := time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC)
	refs := []store.ReferenceWithBody{
		refWithBody("r1", "Hello", "Short fact line.", t0, "type:user"),
	}
	got := string(store.RenderIndex(refs))
	want := "- [Hello](r1.md) — Short fact line."
	if !strings.Contains(got, want) {
		t.Errorf("expected line %q in output:\n%s", want, got)
	}
}

func TestRenderIndexNoBodyNoEmDash(t *testing.T) {
	t0 := time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC)
	refs := []store.ReferenceWithBody{
		refWithBody("r1", "Hello", "", t0, "type:user"),
	}
	got := string(store.RenderIndex(refs))
	if strings.Contains(got, "—") {
		t.Errorf("expected no em-dash for empty body, got:\n%s", got)
	}
	if !strings.Contains(got, "- [Hello](r1.md)") {
		t.Errorf("missing bullet line in output:\n%s", got)
	}
}

func TestRenderIndexStripsHeadingHash(t *testing.T) {
	t0 := time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC)
	refs := []store.ReferenceWithBody{
		refWithBody("r1", "Hello", "## A heading\n\nbody text", t0, "type:user"),
	}
	got := string(store.RenderIndex(refs))
	if !strings.Contains(got, "— A heading") {
		t.Errorf("expected stripped heading as desc:\n%s", got)
	}
}

func TestRenderIndexTruncatesLongDesc(t *testing.T) {
	t0 := time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC)
	long := strings.Repeat("x", 200)
	refs := []store.ReferenceWithBody{
		refWithBody("r1", "Hello", long, t0, "type:user"),
	}
	got := string(store.RenderIndex(refs))
	// Description portion lives on the bullet line; just check the line
	// length is bounded (header + delimiter + 80 runes + leeway for prefix).
	for line := range strings.SplitSeq(got, "\n") {
		if strings.HasPrefix(line, "- [") {
			descRunes := []rune(line)
			if len(descRunes) > 200 {
				t.Errorf("expected truncated desc; got line of %d runes:\n%s", len(descRunes), line)
			}
			if !strings.HasSuffix(line, "...") {
				t.Errorf("expected ellipsis on truncated line:\n%s", line)
			}
		}
	}
}

func TestRenderIndexUnknownTypeFallsToOther(t *testing.T) {
	t0 := time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC)
	refs := []store.ReferenceWithBody{
		refWithBody("a", "A", "fact", t0, "type:area_of_focus"),
	}
	got := string(store.RenderIndex(refs))
	if !strings.Contains(got, "## other") {
		t.Errorf("expected `## other` section:\n%s", got)
	}
	for _, section := range []string{"## user", "## feedback", "## project", "## reference"} {
		if strings.Contains(got, section) {
			t.Errorf("did not expect section %q in single-other output:\n%s", section, got)
		}
	}
}

func TestRenderIndexTiebreakByID(t *testing.T) {
	tt := time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC)
	refs := []store.ReferenceWithBody{
		refWithBody("zzz", "ZZZ", "z", tt, "type:user"),
		refWithBody("aaa", "AAA", "a", tt, "type:user"),
		refWithBody("mmm", "MMM", "m", tt, "type:user"),
	}
	got := string(store.RenderIndex(refs))
	idxA := strings.Index(got, "AAA")
	idxM := strings.Index(got, "MMM")
	idxZ := strings.Index(got, "ZZZ")
	if idxA >= idxM || idxM >= idxZ {
		t.Errorf("tiebreak by id asc failed: A=%d M=%d Z=%d\n%s", idxA, idxM, idxZ, got)
	}
}

func TestWriteIndexRoundTrip(t *testing.T) {
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatal(err)
	}
	tt := time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC)
	for _, id := range []string{"r1", "r2"} {
		ref := &model.Reference{ID: id, Title: id, CreatedAt: tt, UpdatedAt: tt, Tags: []string{"type:user"}}
		if err := store.WriteRef(store.PathForReferenceActive(cfg, "claude", id), ref, "fact "+id); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.WriteIndex(cfg, "claude"); err != nil {
		t.Fatalf("WriteIndex: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(cfg.ReferenceToolDir("claude"), "INDEX.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(got), "# Reference index\n") {
		t.Errorf("missing header:\n%s", got)
	}
	if !strings.Contains(string(got), "[r1](r1.md)") {
		t.Errorf("missing r1 entry:\n%s", got)
	}

	// Idempotent — second call leaves bytes identical.
	if err := store.WriteIndex(cfg, "claude"); err != nil {
		t.Fatal(err)
	}
	again, err := os.ReadFile(filepath.Join(cfg.ReferenceToolDir("claude"), "INDEX.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, again) {
		t.Error("WriteIndex not idempotent on disk")
	}
}

func TestWriteIndexEmptySetWritesStub(t *testing.T) {
	cfg := newTestCfg(t)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureReferenceToolDir(cfg, "claude"); err != nil {
		t.Fatal(err)
	}
	if err := store.WriteIndex(cfg, "claude"); err != nil {
		t.Fatalf("WriteIndex: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(cfg.ReferenceToolDir("claude"), "INDEX.md"))
	if err != nil {
		t.Fatal(err)
	}
	want := "# Reference index\n\n_No entries._\n"
	if string(got) != want {
		t.Errorf("empty stub mismatch\nwant: %q\n got: %q", want, string(got))
	}
}
