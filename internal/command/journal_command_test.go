package command_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/takai/htd/internal/config"
	"github.com/takai/htd/internal/store"
)

// ---------- journal new ----------

func TestJournalNewDailyDefaultsToToday(t *testing.T) {
	dir := setupDir(t)
	out, _, err := runCmd(t, dir, "journal", "new")
	if err != nil {
		t.Fatalf("journal new: %v", err)
	}
	name := strings.TrimSpace(out)
	want := time.Now().Format("2006-01-02")
	if name != want {
		t.Errorf("name: want %q, got %q", want, name)
	}
	cfg := config.New(dir)
	path := store.PathForJournal(cfg, name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read journal: %v", err)
	}
	if !strings.Contains(string(data), "## What I did") {
		t.Errorf("expected daily template scaffold:\n%s", data)
	}
	if !strings.Contains(string(data), "created_at:") {
		t.Errorf("expected frontmatter with created_at:\n%s", data)
	}
}

func TestJournalNewDailyExplicitDate(t *testing.T) {
	dir := setupDir(t)
	out, _, err := runCmd(t, dir, "journal", "new", "--date", "2026-04-15")
	if err != nil {
		t.Fatal(err)
	}
	if name := strings.TrimSpace(out); name != "2026-04-15" {
		t.Errorf("name: want %q, got %q", "2026-04-15", name)
	}
}

func TestJournalNewWeeklySnapsToMonday(t *testing.T) {
	// 2026-04-30 is a Thursday (ISO week 18); Monday of that week is 2026-04-27.
	dir := setupDir(t)
	out, _, err := runCmd(t, dir, "journal", "new", "--type", "weekly", "--date", "2026-04-30")
	if err != nil {
		t.Fatal(err)
	}
	name := strings.TrimSpace(out)
	if name != "2026-04-27-weekly" {
		t.Errorf("name: want %q, got %q", "2026-04-27-weekly", name)
	}
	cfg := config.New(dir)
	data, err := os.ReadFile(store.PathForJournal(cfg, name))
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{"## Wins", "## Misses", "## Lessons", "## Focus next week"} {
		if !strings.Contains(string(data), marker) {
			t.Errorf("missing %q in weekly template:\n%s", marker, data)
		}
	}
}

func TestJournalNewAdhoc(t *testing.T) {
	dir := setupDir(t)
	out, _, err := runCmd(t, dir, "journal", "new", "--type", "adhoc", "--title", "Postmortem on outage")
	if err != nil {
		t.Fatal(err)
	}
	name := strings.TrimSpace(out)
	if name != "postmortem_on_outage" {
		t.Errorf("name: want %q, got %q", "postmortem_on_outage", name)
	}
	cfg := config.New(dir)
	data, err := os.ReadFile(store.PathForJournal(cfg, name))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "# Postmortem on outage") {
		t.Errorf("expected H1 from --title:\n%s", data)
	}
}

func TestJournalNewAdhocRequiresTitle(t *testing.T) {
	dir := setupDir(t)
	_, _, err := runCmd(t, dir, "journal", "new", "--type", "adhoc")
	if err == nil {
		t.Fatal("expected error when --title missing for adhoc")
	}
}

func TestJournalNewRefusesClobber(t *testing.T) {
	dir := setupDir(t)
	if _, _, err := runCmd(t, dir, "journal", "new", "--date", "2026-04-15"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runCmd(t, dir, "journal", "new", "--date", "2026-04-15"); err == nil {
		t.Fatal("expected error on second create")
	}
}

func TestJournalNewInvalidType(t *testing.T) {
	dir := setupDir(t)
	_, _, err := runCmd(t, dir, "journal", "new", "--type", "monthly")
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

func TestJournalNewInvalidDate(t *testing.T) {
	dir := setupDir(t)
	_, _, err := runCmd(t, dir, "journal", "new", "--date", "not-a-date")
	if err == nil {
		t.Fatal("expected error for invalid date")
	}
}

func TestJournalNewWithTags(t *testing.T) {
	dir := setupDir(t)
	out, _, err := runCmd(t, dir, "journal", "new", "--date", "2026-04-15", "--tag", "personal", "--tag", "work")
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.New(dir)
	data, err := os.ReadFile(store.PathForJournal(cfg, strings.TrimSpace(out)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "personal") || !strings.Contains(string(data), "work") {
		t.Errorf("expected tags in frontmatter:\n%s", data)
	}
}

// ---------- journal list ----------

func TestJournalList(t *testing.T) {
	dir := setupDir(t)
	for _, d := range []string{"2026-04-15", "2026-04-16", "2026-04-17"} {
		if _, _, err := runCmd(t, dir, "journal", "new", "--date", d); err != nil {
			t.Fatal(err)
		}
	}
	out, _, err := runCmd(t, dir, "journal", "list")
	if err != nil {
		t.Fatal(err)
	}
	idx15 := strings.Index(out, "2026-04-15")
	idx16 := strings.Index(out, "2026-04-16")
	idx17 := strings.Index(out, "2026-04-17")
	if idx15 < 0 || idx16 < 0 || idx17 < 0 {
		t.Fatalf("missing entries in output:\n%s", out)
	}
	// Most recent first.
	if idx17 >= idx16 || idx16 >= idx15 {
		t.Errorf("expected desc order: 17=%d 16=%d 15=%d\n%s", idx17, idx16, idx15, out)
	}
}

func TestJournalListSince(t *testing.T) {
	dir := setupDir(t)
	for _, d := range []string{"2026-04-15", "2026-04-16", "2026-04-17"} {
		if _, _, err := runCmd(t, dir, "journal", "new", "--date", d); err != nil {
			t.Fatal(err)
		}
	}
	out, _, err := runCmd(t, dir, "journal", "list", "--since", "2026-04-16")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "2026-04-15") {
		t.Errorf("entry before --since leaked:\n%s", out)
	}
	if !strings.Contains(out, "2026-04-16") || !strings.Contains(out, "2026-04-17") {
		t.Errorf("missing on-or-after entries:\n%s", out)
	}
}

func TestJournalListJSON(t *testing.T) {
	dir := setupDir(t)
	if _, _, err := runCmd(t, dir, "journal", "new", "--date", "2026-04-15", "--tag", "x"); err != nil {
		t.Fatal(err)
	}
	out, _, err := runCmd(t, dir, "--json", "journal", "list")
	if err != nil {
		t.Fatal(err)
	}
	var got []map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	if len(got) != 1 {
		t.Fatalf("len: want 1, got %d", len(got))
	}
	if got[0]["name"] != "2026-04-15" {
		t.Errorf("name: %v", got[0]["name"])
	}
	tags, _ := got[0]["tags"].([]any)
	if len(tags) != 1 || tags[0] != "x" {
		t.Errorf("tags: %v", got[0]["tags"])
	}
}

// ---------- journal show ----------

func TestJournalShow(t *testing.T) {
	dir := setupDir(t)
	if _, _, err := runCmd(t, dir, "journal", "new", "--date", "2026-04-15"); err != nil {
		t.Fatal(err)
	}
	out, _, err := runCmd(t, dir, "journal", "show", "2026-04-15")
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	if !strings.Contains(out, "name:       2026-04-15") {
		t.Errorf("expected name line:\n%s", out)
	}
	if !strings.Contains(out, "## What I did") {
		t.Errorf("expected daily scaffold:\n%s", out)
	}
}

func TestJournalShowPlainMarkdown(t *testing.T) {
	// Hand-written file with no frontmatter must still be readable.
	dir := setupDir(t)
	cfg := config.New(dir)
	path := filepath.Join(cfg.JournalDir(), "draft.md")
	if err := os.WriteFile(path, []byte("# Just a draft\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, err := runCmd(t, dir, "journal", "show", "draft")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "# Just a draft") {
		t.Errorf("expected body in output:\n%s", out)
	}
}

func TestJournalShowNotFound(t *testing.T) {
	dir := setupDir(t)
	_, _, err := runCmd(t, dir, "journal", "show", "ghost")
	if err == nil {
		t.Fatal("expected error")
	}
	if !store.IsNotFound(err) {
		t.Errorf("expected NotFoundError, got %T %v", err, err)
	}
}
