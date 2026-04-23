package command_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/takai/htd/internal/config"
	"github.com/takai/htd/internal/model"
	"github.com/takai/htd/internal/store"
	"github.com/takai/htd/internal/command"
)

// runCmd executes the root command with the given args against a temp directory.
// Returns stdout, stderr, and the error (if any).
func runCmd(t *testing.T, dir string, args ...string) (string, string, error) {
	t.Helper()
	root := command.NewRootCommand()
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs(append([]string{"--path", dir}, args...))
	err := root.Execute()
	return out.String(), errOut.String(), err
}

func setupDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cfg := config.New(dir)
	if err := store.EnsureDirs(cfg); err != nil {
		t.Fatal(err)
	}
	return dir
}

func writeItem(t *testing.T, dir string, item *model.Item, body string) string {
	t.Helper()
	cfg := config.New(dir)
	p := store.PathForItem(cfg, item)
	if err := store.Write(p, item, body); err != nil {
		t.Fatalf("writeItem: %v", err)
	}
	return p
}

func readItem(t *testing.T, dir, id string) (*model.Item, string) {
	t.Helper()
	cfg := config.New(dir)
	p, err := store.FindItem(cfg, id)
	if err != nil {
		t.Fatalf("readItem FindItem %q: %v", id, err)
	}
	item, body, err := store.Read(p)
	if err != nil {
		t.Fatalf("readItem Read %q: %v", p, err)
	}
	return item, body
}

func nowItem(id string, kind model.Kind, status model.Status) *model.Item {
	now := time.Now()
	return &model.Item{
		ID: id, Title: id, Kind: kind, Status: status,
		CreatedAt: now, UpdatedAt: now,
	}
}

// ---------- capture ----------

func TestCaptureAdd(t *testing.T) {
	dir := setupDir(t)
	out, _, err := runCmd(t, dir, "capture", "add", "--title", "My new task")
	if err != nil {
		t.Fatalf("capture add: %v", err)
	}
	id := strings.TrimSpace(out)
	if id == "" {
		t.Fatal("expected non-empty ID output")
	}

	item, _ := readItem(t, dir, id)
	if item.Kind != model.KindInbox {
		t.Errorf("kind: want inbox, got %q", item.Kind)
	}
	if item.Status != model.StatusActive {
		t.Errorf("status: want active, got %q", item.Status)
	}
	if item.Title != "My new task" {
		t.Errorf("title: want %q, got %q", "My new task", item.Title)
	}
}

func TestCaptureAddWithOptions(t *testing.T) {
	dir := setupDir(t)
	out, _, err := runCmd(t, dir, "capture", "add",
		"--title", "Task with options",
		"--body", "Detail body",
		"--source", "email",
		"--tag", "cli", "--tag", "docs",
	)
	if err != nil {
		t.Fatalf("capture add: %v", err)
	}
	id := strings.TrimSpace(out)
	item, body := readItem(t, dir, id)
	if item.Source != "email" {
		t.Errorf("source: want email, got %q", item.Source)
	}
	if len(item.Tags) != 2 {
		t.Errorf("tags: want 2, got %d", len(item.Tags))
	}
	if body != "Detail body" {
		t.Errorf("body: want %q, got %q", "Detail body", body)
	}
}

func TestCaptureAddWithRefs(t *testing.T) {
	dir := setupDir(t)
	out, _, err := runCmd(t, dir, "capture", "add",
		"--title", "Review PR",
		"--ref", "https://github.com/foo/bar/pull/1",
		"--ref", "https://notion.so/x",
	)
	if err != nil {
		t.Fatalf("capture add --ref: %v", err)
	}
	id := strings.TrimSpace(out)
	item, _ := readItem(t, dir, id)
	if len(item.Refs) != 2 {
		t.Fatalf("refs: want 2, got %d (%v)", len(item.Refs), item.Refs)
	}
	if item.Refs[0] != "https://github.com/foo/bar/pull/1" || item.Refs[1] != "https://notion.so/x" {
		t.Errorf("refs: got %v", item.Refs)
	}
}

func TestCaptureAddOmitsRefsWhenAbsent(t *testing.T) {
	dir := setupDir(t)
	out, _, err := runCmd(t, dir, "capture", "add", "--title", "No refs")
	if err != nil {
		t.Fatalf("capture add: %v", err)
	}
	id := strings.TrimSpace(out)
	cfg := config.New(dir)
	p := filepath.Join(cfg.DirForKind(model.KindInbox), id+".md")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if strings.Contains(string(data), "refs:") {
		t.Errorf("expected no refs field in YAML when unset:\n%s", string(data))
	}
}

func TestCaptureAddDone(t *testing.T) {
	dir := setupDir(t)
	out, _, err := runCmd(t, dir, "capture", "add", "--title", "Quick thing", "--done")
	if err != nil {
		t.Fatalf("capture add --done: %v", err)
	}
	id := strings.TrimSpace(out)
	if id == "" {
		t.Fatal("expected ID output")
	}

	cfg := config.New(dir)
	archivePath := filepath.Join(cfg.ArchiveItemsDir(), id+".md")
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("expected item at %q, stat err: %v", archivePath, err)
	}
	inboxPath := filepath.Join(cfg.DirForKind(model.KindInbox), id+".md")
	if _, err := os.Stat(inboxPath); !os.IsNotExist(err) {
		t.Errorf("expected no item at %q; stat err: %v", inboxPath, err)
	}

	item, _ := readItem(t, dir, id)
	if item.Kind != model.KindNextAction {
		t.Errorf("kind: want next_action, got %q", item.Kind)
	}
	if item.Status != model.StatusDone {
		t.Errorf("status: want done, got %q", item.Status)
	}
}

func TestCaptureAddDonePreservesMetadata(t *testing.T) {
	dir := setupDir(t)
	out, _, err := runCmd(t, dir, "capture", "add",
		"--title", "Quick thing with detail",
		"--body", "body text",
		"--source", "manual",
		"--tag", "quick",
		"--done",
	)
	if err != nil {
		t.Fatalf("capture add --done: %v", err)
	}
	id := strings.TrimSpace(out)
	item, body := readItem(t, dir, id)
	if item.Source != "manual" {
		t.Errorf("source: want manual, got %q", item.Source)
	}
	if len(item.Tags) != 1 || item.Tags[0] != "quick" {
		t.Errorf("tags: want [quick], got %v", item.Tags)
	}
	if body != "body text" {
		t.Errorf("body: want %q, got %q", "body text", body)
	}
}

func TestCaptureAddIDFormat(t *testing.T) {
	dir := setupDir(t)
	out, _, err := runCmd(t, dir, "capture", "add", "--title", "Write the man page")
	if err != nil {
		t.Fatal(err)
	}
	id := strings.TrimSpace(out)
	today := time.Now().Format("20060102")
	if !strings.HasPrefix(id, today+"-") {
		t.Errorf("ID %q should start with %s-", id, today)
	}
	if !strings.HasSuffix(id, "-write_the_man_page") {
		t.Errorf("ID %q should end with -write_the_man_page", id)
	}
}

// ---------- clarify ----------

func TestClarifyList(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260417-inbox1", model.KindInbox, model.StatusActive), "")
	writeItem(t, dir, nowItem("20260417-inbox2", model.KindInbox, model.StatusActive), "")
	writeItem(t, dir, nowItem("20260417-next1", model.KindNextAction, model.StatusActive), "")

	out, _, err := runCmd(t, dir, "clarify", "list")
	if err != nil {
		t.Fatalf("clarify list: %v", err)
	}
	if !strings.Contains(out, "20260417-inbox1") || !strings.Contains(out, "20260417-inbox2") {
		t.Errorf("clarify list missing inbox items: %q", out)
	}
	if strings.Contains(out, "20260417-next1") {
		t.Errorf("clarify list should not contain non-inbox item: %q", out)
	}
}

func TestClarifyShow(t *testing.T) {
	dir := setupDir(t)
	item := nowItem("20260417-show_me", model.KindInbox, model.StatusActive)
	item.Title = "Show Me Item"
	writeItem(t, dir, item, "body content")

	out, _, err := runCmd(t, dir, "clarify", "show", "20260417-show_me")
	if err != nil {
		t.Fatalf("clarify show: %v", err)
	}
	if !strings.Contains(out, "Show Me Item") {
		t.Errorf("clarify show missing title: %q", out)
	}
	if !strings.Contains(out, "body content") {
		t.Errorf("clarify show missing body: %q", out)
	}
}

func TestClarifyShowNotFound(t *testing.T) {
	dir := setupDir(t)
	_, _, err := runCmd(t, dir, "clarify", "show", "20260417-ghost")
	if !store.IsNotFound(err) {
		t.Errorf("expected NotFoundError, got %v", err)
	}
}

func TestClarifyUpdate(t *testing.T) {
	dir := setupDir(t)
	item := nowItem("20260417-update_me", model.KindInbox, model.StatusActive)
	writeItem(t, dir, item, "old body")

	_, _, err := runCmd(t, dir, "clarify", "update", "20260417-update_me", "--title", "New Title", "--body", "new body")
	if err != nil {
		t.Fatalf("clarify update: %v", err)
	}

	got, gotBody := readItem(t, dir, "20260417-update_me")
	if got.Title != "New Title" {
		t.Errorf("title: want %q, got %q", "New Title", got.Title)
	}
	if gotBody != "new body" {
		t.Errorf("body: want %q, got %q", "new body", gotBody)
	}
}

func TestClarifyUpdateRefs(t *testing.T) {
	dir := setupDir(t)
	item := nowItem("20260417-refs_me", model.KindInbox, model.StatusActive)
	item.Refs = []string{"https://old.example.com/1"}
	writeItem(t, dir, item, "")

	_, _, err := runCmd(t, dir, "clarify", "update", "20260417-refs_me",
		"--ref", "https://new.example.com/a",
		"--ref", "https://new.example.com/b",
	)
	if err != nil {
		t.Fatalf("clarify update --ref: %v", err)
	}

	got, _ := readItem(t, dir, "20260417-refs_me")
	if len(got.Refs) != 2 {
		t.Fatalf("refs: want 2, got %d (%v)", len(got.Refs), got.Refs)
	}
	if got.Refs[0] != "https://new.example.com/a" || got.Refs[1] != "https://new.example.com/b" {
		t.Errorf("refs: got %v", got.Refs)
	}
}

func TestClarifyUpdateLeavesRefsUntouched(t *testing.T) {
	dir := setupDir(t)
	item := nowItem("20260417-keep_refs", model.KindInbox, model.StatusActive)
	item.Refs = []string{"https://keep.example.com/"}
	writeItem(t, dir, item, "")

	_, _, err := runCmd(t, dir, "clarify", "update", "20260417-keep_refs", "--title", "Retitled")
	if err != nil {
		t.Fatalf("clarify update --title: %v", err)
	}
	got, _ := readItem(t, dir, "20260417-keep_refs")
	if len(got.Refs) != 1 || got.Refs[0] != "https://keep.example.com/" {
		t.Errorf("refs: want unchanged, got %v", got.Refs)
	}
}

func TestClarifyDiscard(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260417-discard_me", model.KindInbox, model.StatusActive), "")

	_, _, err := runCmd(t, dir, "clarify", "discard", "20260417-discard_me")
	if err != nil {
		t.Fatalf("clarify discard: %v", err)
	}

	got, _ := readItem(t, dir, "20260417-discard_me")
	if got.Status != model.StatusDiscarded {
		t.Errorf("status: want discarded, got %q", got.Status)
	}
	// Verify it's in archive
	cfg := config.New(dir)
	archivePath := filepath.Join(cfg.ArchiveItemsDir(), "20260417-discard_me.md")
	if _, err2 := readItemFromPath(t, archivePath); err2 != nil {
		t.Errorf("discarded item not in archive: %v", err2)
	}
}

func TestClarifyDiscardNonInbox(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260417-next_item", model.KindNextAction, model.StatusActive), "")

	_, _, err := runCmd(t, dir, "clarify", "discard", "20260417-next_item")
	if err == nil {
		t.Error("expected error when discarding non-inbox item")
	}
}

// ---------- organize ----------

func TestOrganizeMove(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260417-move_me", model.KindInbox, model.StatusActive), "")

	_, _, err := runCmd(t, dir, "organize", "move", "next_action", "20260417-move_me")
	if err != nil {
		t.Fatalf("organize move: %v", err)
	}

	got, _ := readItem(t, dir, "20260417-move_me")
	if got.Kind != model.KindNextAction {
		t.Errorf("kind: want next_action, got %q", got.Kind)
	}
	// Verify file is in new location
	cfg := config.New(dir)
	newPath := filepath.Join(cfg.DirForKind(model.KindNextAction), "20260417-move_me.md")
	if _, err2 := readItemFromPath(t, newPath); err2 != nil {
		t.Errorf("item not in new location: %v", err2)
	}
}

func TestOrganizeMoveToInboxFails(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260417-some", model.KindNextAction, model.StatusActive), "")

	_, _, err := runCmd(t, dir, "organize", "move", "inbox", "20260417-some")
	if err == nil {
		t.Error("expected error when moving to inbox")
	}
}

func TestOrganizeMoveBulk(t *testing.T) {
	dir := setupDir(t)
	for _, id := range []string{"20260417-bm1", "20260417-bm2", "20260417-bm3"} {
		writeItem(t, dir, nowItem(id, model.KindInbox, model.StatusActive), "")
	}

	_, _, err := runCmd(t, dir, "organize", "move", "someday",
		"20260417-bm1", "20260417-bm2", "20260417-bm3")
	if err != nil {
		t.Fatalf("organize move (bulk): %v", err)
	}

	for _, id := range []string{"20260417-bm1", "20260417-bm2", "20260417-bm3"} {
		got, _ := readItem(t, dir, id)
		if got.Kind != model.KindSomeday {
			t.Errorf("%s kind: want someday, got %q", id, got.Kind)
		}
	}
}

func TestOrganizeMoveBulkStopsOnFirstError(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260417-bm_ok", model.KindInbox, model.StatusActive), "")

	_, _, err := runCmd(t, dir, "organize", "move", "someday",
		"20260417-bm_ok", "20260417-bm_missing")
	if err == nil {
		t.Fatal("expected error when an ID is missing")
	}
	got, _ := readItem(t, dir, "20260417-bm_ok")
	if got.Kind != model.KindSomeday {
		t.Errorf("earlier ID should be moved before failure: got kind %q", got.Kind)
	}
}

func TestOrganizeMoveRequiresID(t *testing.T) {
	dir := setupDir(t)
	_, _, err := runCmd(t, dir, "organize", "move", "someday")
	if err == nil {
		t.Error("expected error when no IDs are given")
	}
}

func TestOrganizeLink(t *testing.T) {
	dir := setupDir(t)
	proj := nowItem("20260417-my_proj", model.KindProject, model.StatusActive)
	task := nowItem("20260417-my_task", model.KindNextAction, model.StatusActive)
	writeItem(t, dir, proj, "")
	writeItem(t, dir, task, "")

	_, _, err := runCmd(t, dir, "organize", "link", "20260417-my_task", "--project", "20260417-my_proj")
	if err != nil {
		t.Fatalf("organize link: %v", err)
	}

	got, _ := readItem(t, dir, "20260417-my_task")
	if got.Project != "20260417-my_proj" {
		t.Errorf("project: want %q, got %q", "20260417-my_proj", got.Project)
	}
}

func TestOrganizeSchedule(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260417-sched", model.KindNextAction, model.StatusActive), "")

	_, _, err := runCmd(t, dir, "organize", "schedule", "20260417-sched", "--due", "2026-05-01")
	if err != nil {
		t.Fatalf("organize schedule: %v", err)
	}

	got, _ := readItem(t, dir, "20260417-sched")
	if got.DueAt == nil {
		t.Error("due_at: want non-nil, got nil")
	}
}

func TestOrganizeScheduleRFC3339Due(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260417-rfc_due", model.KindNextAction, model.StatusActive), "")

	const input = "2026-05-01T14:30:00+09:00"
	_, _, err := runCmd(t, dir, "organize", "schedule", "20260417-rfc_due", "--due", input)
	if err != nil {
		t.Fatalf("organize schedule: %v", err)
	}

	want, err := time.Parse(time.RFC3339, input)
	if err != nil {
		t.Fatalf("parse want: %v", err)
	}
	got, _ := readItem(t, dir, "20260417-rfc_due")
	if got.DueAt == nil {
		t.Fatal("due_at: want non-nil, got nil")
	}
	if !got.DueAt.Equal(want) {
		t.Errorf("due_at: want %s, got %s", want, got.DueAt)
	}

	cfg := config.New(dir)
	p := filepath.Join(cfg.DirForKind(model.KindNextAction), "20260417-rfc_due.md")
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if !strings.Contains(string(raw), input) {
		t.Errorf("on-disk YAML should preserve RFC3339 string %q, got:\n%s", input, raw)
	}
}

func TestOrganizeScheduleRFC3339Defer(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260417-rfc_defer_future", model.KindNextAction, model.StatusActive), "")
	writeItem(t, dir, nowItem("20260417-rfc_defer_past", model.KindNextAction, model.StatusActive), "")

	future := time.Now().Add(1 * time.Hour).Format(time.RFC3339)
	past := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)

	if _, _, err := runCmd(t, dir, "organize", "schedule", "20260417-rfc_defer_future", "--defer", future); err != nil {
		t.Fatalf("schedule future defer: %v", err)
	}
	if _, _, err := runCmd(t, dir, "organize", "schedule", "20260417-rfc_defer_past", "--defer", past); err != nil {
		t.Fatalf("schedule past defer: %v", err)
	}

	got, _ := readItem(t, dir, "20260417-rfc_defer_future")
	if got.DeferUntil == nil {
		t.Fatal("defer_until: want non-nil, got nil")
	}
	wantFuture, _ := time.Parse(time.RFC3339, future)
	if !got.DeferUntil.Equal(wantFuture) {
		t.Errorf("defer_until: want %s, got %s", wantFuture, got.DeferUntil)
	}

	out, _, err := runCmd(t, dir, "engage", "next-action")
	if err != nil {
		t.Fatalf("engage next-action: %v", err)
	}
	if strings.Contains(out, "20260417-rfc_defer_future") {
		t.Errorf("item with defer_until in the future should be hidden: %q", out)
	}
	if !strings.Contains(out, "20260417-rfc_defer_past") {
		t.Errorf("item with defer_until in the past should be visible: %q", out)
	}
}

func TestEngageNextActionSortsIntraDay(t *testing.T) {
	dir := setupDir(t)
	day := time.Date(2026, 5, 1, 0, 0, 0, 0, time.Local)
	morning := day.Add(9 * time.Hour)
	afternoon := day.Add(13 * time.Hour)
	evening := day.Add(18 * time.Hour)

	e := nowItem("20260501-c_evening", model.KindNextAction, model.StatusActive)
	e.DueAt = &evening
	m := nowItem("20260501-a_morning", model.KindNextAction, model.StatusActive)
	m.DueAt = &morning
	a := nowItem("20260501-b_afternoon", model.KindNextAction, model.StatusActive)
	a.DueAt = &afternoon
	writeItem(t, dir, e, "")
	writeItem(t, dir, m, "")
	writeItem(t, dir, a, "")

	out, _, err := runCmd(t, dir, "engage", "next-action")
	if err != nil {
		t.Fatalf("engage next-action: %v", err)
	}

	want := []string{
		"20260501-a_morning",
		"20260501-b_afternoon",
		"20260501-c_evening",
	}
	var positions []int
	for _, id := range want {
		idx := strings.Index(out, id)
		if idx < 0 {
			t.Fatalf("missing %s in output: %q", id, out)
		}
		positions = append(positions, idx)
	}
	for i := 1; i < len(positions); i++ {
		if positions[i] < positions[i-1] {
			t.Errorf("sort order: %s should appear after %s\nout=%q", want[i], want[i-1], out)
		}
	}
}

func TestOrganizePromote(t *testing.T) {
	dir := setupDir(t)
	parent := nowItem("20260420-launch_cli", model.KindInbox, model.StatusActive)
	parent.Title = "Launch CLI"
	writeItem(t, dir, parent, "")

	out, _, err := runCmd(t, dir, "organize", "promote", "20260420-launch_cli",
		"--child", "Verify on staging",
		"--child", "Release to production",
	)
	if err != nil {
		t.Fatalf("organize promote: %v", err)
	}

	gotParent, _ := readItem(t, dir, "20260420-launch_cli")
	if gotParent.Kind != model.KindProject {
		t.Errorf("parent kind: want project, got %q", gotParent.Kind)
	}
	if gotParent.Status != model.StatusActive {
		t.Errorf("parent status: want active, got %q", gotParent.Status)
	}
	cfg := config.New(dir)
	projectPath := filepath.Join(cfg.DirForKind(model.KindProject), "20260420-launch_cli.md")
	if _, err := os.Stat(projectPath); err != nil {
		t.Errorf("parent not moved to project dir: %v", err)
	}

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("want 3 output lines (parent + 2 children), got %d: %q", len(lines), out)
	}
	if lines[0] != "20260420-launch_cli" {
		t.Errorf("line 0: want parent ID, got %q", lines[0])
	}
	childIDs := []string{lines[1], lines[2]}

	today := time.Now().Format("20060102")
	wantSuffixes := []string{"-verify_on_staging", "-release_to_production"}
	for i, id := range childIDs {
		if !strings.HasPrefix(id, today+"-") {
			t.Errorf("child %d ID %q should start with today's date", i, id)
		}
		if !strings.HasSuffix(id, wantSuffixes[i]) {
			t.Errorf("child %d ID %q should end with %q", i, id, wantSuffixes[i])
		}
		child, _ := readItem(t, dir, id)
		if child.Kind != model.KindNextAction {
			t.Errorf("child %s kind: want next_action, got %q", id, child.Kind)
		}
		if child.Status != model.StatusActive {
			t.Errorf("child %s status: want active, got %q", id, child.Status)
		}
		if child.Project != "20260420-launch_cli" {
			t.Errorf("child %s project: want parent ID, got %q", id, child.Project)
		}
	}
	if childIDs[0] == childIDs[1] {
		t.Errorf("expected distinct child IDs, got duplicate %q", childIDs[0])
	}
}

func TestOrganizePromoteParentNotFound(t *testing.T) {
	dir := setupDir(t)
	_, _, err := runCmd(t, dir, "organize", "promote", "20260420-ghost", "--child", "X")
	if !store.IsNotFound(err) {
		t.Errorf("expected NotFoundError, got %v", err)
	}
}

func TestOrganizePromoteParentTerminal(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260420-done_parent", model.KindNextAction, model.StatusDone), "")

	_, _, err := runCmd(t, dir, "organize", "promote", "20260420-done_parent", "--child", "X")
	if err == nil {
		t.Error("expected error when promoting a terminal item")
	}
}

func TestOrganizePromoteAlreadyProject(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260420-existing_proj", model.KindProject, model.StatusActive), "")

	out, _, err := runCmd(t, dir, "organize", "promote", "20260420-existing_proj", "--child", "Next step")
	if err != nil {
		t.Fatalf("organize promote: %v", err)
	}

	gotParent, _ := readItem(t, dir, "20260420-existing_proj")
	if gotParent.Kind != model.KindProject {
		t.Errorf("parent kind: want project, got %q", gotParent.Kind)
	}

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 output lines, got %d: %q", len(lines), out)
	}
	child, _ := readItem(t, dir, lines[1])
	if child.Project != "20260420-existing_proj" {
		t.Errorf("child project: want parent ID, got %q", child.Project)
	}
}

func TestOrganizePromoteFromNextAction(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260420-was_na", model.KindNextAction, model.StatusActive), "")

	_, _, err := runCmd(t, dir, "organize", "promote", "20260420-was_na", "--child", "First step")
	if err != nil {
		t.Fatalf("organize promote: %v", err)
	}
	got, _ := readItem(t, dir, "20260420-was_na")
	if got.Kind != model.KindProject {
		t.Errorf("kind: want project, got %q", got.Kind)
	}
}

func TestOrganizePromoteRequiresChild(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260420-no_kids", model.KindInbox, model.StatusActive), "")

	_, _, err := runCmd(t, dir, "organize", "promote", "20260420-no_kids")
	if err == nil {
		t.Error("expected error when --child is omitted")
	}
}

func TestOrganizePromoteEmptyChildTitle(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260420-empty_kid", model.KindInbox, model.StatusActive), "")

	_, _, err := runCmd(t, dir, "organize", "promote", "20260420-empty_kid", "--child", "")
	if err == nil {
		t.Error("expected error when --child title is empty")
	}
}

func TestOrganizePromoteChildIDCollision(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260420-parent", model.KindInbox, model.StatusActive), "")

	out, _, err := runCmd(t, dir, "organize", "promote", "20260420-parent",
		"--child", "Same Title",
		"--child", "Same Title",
	)
	if err != nil {
		t.Fatalf("organize promote: %v", err)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("want 3 lines, got %d: %q", len(lines), out)
	}
	if lines[1] == lines[2] {
		t.Errorf("children must have distinct IDs, got duplicate %q", lines[1])
	}
	if !strings.HasSuffix(lines[2], "_2") {
		t.Errorf("second colliding child should end with _2, got %q", lines[2])
	}
}

func TestOrganizePromoteJSON(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260420-json_parent", model.KindInbox, model.StatusActive), "")

	out, _, err := runCmd(t, dir, "--json", "organize", "promote", "20260420-json_parent",
		"--child", "Alpha",
		"--child", "Beta",
	)
	if err != nil {
		t.Fatalf("organize promote --json: %v", err)
	}
	var obj struct {
		Parent   string   `json:"parent"`
		Children []string `json:"children"`
	}
	if err := json.Unmarshal([]byte(out), &obj); err != nil {
		t.Fatalf("parse json: %v (out=%q)", err, out)
	}
	if obj.Parent != "20260420-json_parent" {
		t.Errorf("parent: want %q, got %q", "20260420-json_parent", obj.Parent)
	}
	if len(obj.Children) != 2 {
		t.Fatalf("children: want 2, got %d", len(obj.Children))
	}
}

// ---------- engage ----------

func TestEngageDone(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260417-done_me", model.KindNextAction, model.StatusActive), "")

	_, _, err := runCmd(t, dir, "engage", "done", "20260417-done_me")
	if err != nil {
		t.Fatalf("engage done: %v", err)
	}

	got, _ := readItem(t, dir, "20260417-done_me")
	if got.Status != model.StatusDone {
		t.Errorf("status: want done, got %q", got.Status)
	}
}

func TestEngageDoneBulk(t *testing.T) {
	dir := setupDir(t)
	ids := []string{"20260417-bd1", "20260417-bd2", "20260417-bd3"}
	for _, id := range ids {
		writeItem(t, dir, nowItem(id, model.KindNextAction, model.StatusActive), "")
	}

	_, _, err := runCmd(t, dir, append([]string{"engage", "done"}, ids...)...)
	if err != nil {
		t.Fatalf("engage done (bulk): %v", err)
	}
	for _, id := range ids {
		got, _ := readItem(t, dir, id)
		if got.Status != model.StatusDone {
			t.Errorf("%s status: want done, got %q", id, got.Status)
		}
	}
}

func TestEngageCancel(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260417-cancel_me", model.KindNextAction, model.StatusActive), "")

	_, _, err := runCmd(t, dir, "engage", "cancel", "20260417-cancel_me")
	if err != nil {
		t.Fatalf("engage cancel: %v", err)
	}

	got, _ := readItem(t, dir, "20260417-cancel_me")
	if got.Status != model.StatusCanceled {
		t.Errorf("status: want canceled, got %q", got.Status)
	}
}

func TestEngageCancelBulk(t *testing.T) {
	dir := setupDir(t)
	ids := []string{"20260417-bc1", "20260417-bc2"}
	for _, id := range ids {
		writeItem(t, dir, nowItem(id, model.KindNextAction, model.StatusActive), "")
	}

	_, _, err := runCmd(t, dir, append([]string{"engage", "cancel"}, ids...)...)
	if err != nil {
		t.Fatalf("engage cancel (bulk): %v", err)
	}
	for _, id := range ids {
		got, _ := readItem(t, dir, id)
		if got.Status != model.StatusCanceled {
			t.Errorf("%s status: want canceled, got %q", id, got.Status)
		}
	}
}

func TestEngageDoneBulkStopsOnFirstError(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260417-bd_ok", model.KindNextAction, model.StatusActive), "")

	_, _, err := runCmd(t, dir, "engage", "done", "20260417-bd_ok", "20260417-bd_missing")
	if err == nil {
		t.Fatal("expected error when an ID is missing")
	}
	got, _ := readItem(t, dir, "20260417-bd_ok")
	if got.Status != model.StatusDone {
		t.Errorf("earlier ID should be marked done before failure: got status %q", got.Status)
	}
}

func TestEngageNextAction(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260417-na_ready", model.KindNextAction, model.StatusActive), "")
	writeItem(t, dir, nowItem("20260417-na_done", model.KindNextAction, model.StatusDone), "")

	deferred := nowItem("20260417-na_deferred", model.KindNextAction, model.StatusActive)
	future := time.Now().Add(24 * time.Hour)
	deferred.DeferUntil = &future
	writeItem(t, dir, deferred, "")

	out, _, err := runCmd(t, dir, "engage", "next-action")
	if err != nil {
		t.Fatalf("engage next-action: %v", err)
	}
	if !strings.Contains(out, "20260417-na_ready") {
		t.Errorf("missing ready next action: %q", out)
	}
	if strings.Contains(out, "20260417-na_done") {
		t.Errorf("should not include done item: %q", out)
	}
	if strings.Contains(out, "20260417-na_deferred") {
		t.Errorf("should not include deferred item: %q", out)
	}
}

func TestEngageNextActionIncludesDueToday(t *testing.T) {
	dir := setupDir(t)
	now := time.Now()
	todayMidnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	due := nowItem("20260420-due_today", model.KindNextAction, model.StatusActive)
	due.DueAt = &todayMidnight
	writeItem(t, dir, due, "")

	past := todayMidnight.Add(-24 * time.Hour)
	overdue := nowItem("20260420-overdue", model.KindNextAction, model.StatusActive)
	overdue.DueAt = &past
	writeItem(t, dir, overdue, "")

	future := todayMidnight.Add(48 * time.Hour)
	upcoming := nowItem("20260420-upcoming", model.KindNextAction, model.StatusActive)
	upcoming.DueAt = &future
	writeItem(t, dir, upcoming, "")

	out, _, err := runCmd(t, dir, "engage", "next-action")
	if err != nil {
		t.Fatalf("engage next-action: %v", err)
	}
	for _, id := range []string{"20260420-due_today", "20260420-overdue", "20260420-upcoming"} {
		if !strings.Contains(out, id) {
			t.Errorf("missing %s: %q", id, out)
		}
	}
}

func TestEngageNextActionSortsByDueAt(t *testing.T) {
	dir := setupDir(t)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterday := today.Add(-24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)

	a := nowItem("20260420-a_tomorrow", model.KindNextAction, model.StatusActive)
	a.DueAt = &tomorrow
	b := nowItem("20260420-b_yesterday", model.KindNextAction, model.StatusActive)
	b.DueAt = &yesterday
	c := nowItem("20260420-c_nodue", model.KindNextAction, model.StatusActive)
	d := nowItem("20260420-d_today", model.KindNextAction, model.StatusActive)
	d.DueAt = &today
	writeItem(t, dir, a, "")
	writeItem(t, dir, b, "")
	writeItem(t, dir, c, "")
	writeItem(t, dir, d, "")

	out, _, err := runCmd(t, dir, "engage", "next-action")
	if err != nil {
		t.Fatalf("engage next-action: %v", err)
	}

	want := []string{
		"20260420-b_yesterday",
		"20260420-d_today",
		"20260420-a_tomorrow",
		"20260420-c_nodue",
	}
	var positions []int
	for _, id := range want {
		idx := strings.Index(out, id)
		if idx < 0 {
			t.Fatalf("missing %s in output: %q", id, out)
		}
		positions = append(positions, idx)
	}
	for i := 1; i < len(positions); i++ {
		if positions[i] < positions[i-1] {
			t.Errorf("sort order: %s should appear after %s\nout=%q", want[i], want[i-1], out)
		}
	}
}

func TestEngageNextActionShowsDueAtColumn(t *testing.T) {
	dir := setupDir(t)
	due := time.Date(2026, 4, 20, 0, 0, 0, 0, time.Local)

	it := nowItem("20260420-dated", model.KindNextAction, model.StatusActive)
	it.DueAt = &due
	it.Project = "20260420-some_proj"
	writeItem(t, dir, it, "")

	out, _, err := runCmd(t, dir, "engage", "next-action")
	if err != nil {
		t.Fatalf("engage next-action: %v", err)
	}
	if !strings.Contains(out, "DUE_AT") {
		t.Errorf("header should include DUE_AT column: %q", out)
	}
	if !strings.Contains(out, "PROJECT") {
		t.Errorf("header should include PROJECT column: %q", out)
	}
	if !strings.Contains(out, "2026-04-20") {
		t.Errorf("due_at value should appear: %q", out)
	}
	if !strings.Contains(out, "20260420-some_proj") {
		t.Errorf("project value should appear: %q", out)
	}
}

func TestEngageNextActionFilterProject(t *testing.T) {
	dir := setupDir(t)
	a := nowItem("20260417-na_a", model.KindNextAction, model.StatusActive)
	a.Project = "20260417-proj_a"
	b := nowItem("20260417-na_b", model.KindNextAction, model.StatusActive)
	b.Project = "20260417-proj_b"
	writeItem(t, dir, a, "")
	writeItem(t, dir, b, "")

	out, _, err := runCmd(t, dir, "engage", "next-action", "--project", "20260417-proj_a")
	if err != nil {
		t.Fatalf("engage next-action --project: %v", err)
	}
	if !strings.Contains(out, "20260417-na_a") {
		t.Errorf("missing project-a next action: %q", out)
	}
	if strings.Contains(out, "20260417-na_b") {
		t.Errorf("should not include project-b next action: %q", out)
	}
}

func TestEngageNextActionFilterTag(t *testing.T) {
	dir := setupDir(t)
	tagged := nowItem("20260417-na_tagged", model.KindNextAction, model.StatusActive)
	tagged.Tags = []string{"urgent", "home"}
	other := nowItem("20260417-na_other", model.KindNextAction, model.StatusActive)
	other.Tags = []string{"work"}
	writeItem(t, dir, tagged, "")
	writeItem(t, dir, other, "")

	out, _, err := runCmd(t, dir, "engage", "next-action", "--tag", "urgent")
	if err != nil {
		t.Fatalf("engage next-action --tag: %v", err)
	}
	if !strings.Contains(out, "20260417-na_tagged") {
		t.Errorf("missing tagged next action: %q", out)
	}
	if strings.Contains(out, "20260417-na_other") {
		t.Errorf("should not include untagged next action: %q", out)
	}
}

func TestEngageWaiting(t *testing.T) {
	dir := setupDir(t)
	now := time.Now()

	stale := nowItem("20260417-wait_stale", model.KindWaitingFor, model.StatusActive)
	stale.UpdatedAt = now.Add(-10 * 24 * time.Hour)
	stale.CreatedAt = stale.UpdatedAt
	writeItem(t, dir, stale, "")

	fresh := nowItem("20260417-wait_fresh", model.KindWaitingFor, model.StatusActive)
	writeItem(t, dir, fresh, "")

	out, _, err := runCmd(t, dir, "engage", "waiting")
	if err != nil {
		t.Fatalf("engage waiting: %v", err)
	}
	if !strings.Contains(out, "20260417-wait_stale") {
		t.Errorf("missing stale waiting item: %q", out)
	}
	if strings.Contains(out, "20260417-wait_fresh") {
		t.Errorf("should not include fresh waiting item: %q", out)
	}
}

func TestEngageWaitingStaleDaysFlag(t *testing.T) {
	dir := setupDir(t)
	now := time.Now()

	item := nowItem("20260417-wait_3d", model.KindWaitingFor, model.StatusActive)
	item.UpdatedAt = now.Add(-3 * 24 * time.Hour)
	item.CreatedAt = item.UpdatedAt
	writeItem(t, dir, item, "")

	out, _, err := runCmd(t, dir, "engage", "waiting")
	if err != nil {
		t.Fatalf("engage waiting (default): %v", err)
	}
	if strings.Contains(out, "20260417-wait_3d") {
		t.Errorf("3-day item should be excluded under default stale-days (7): %q", out)
	}

	out, _, err = runCmd(t, dir, "engage", "waiting", "--stale-days", "2")
	if err != nil {
		t.Fatalf("engage waiting --stale-days 2: %v", err)
	}
	if !strings.Contains(out, "20260417-wait_3d") {
		t.Errorf("3-day item should be included under --stale-days 2: %q", out)
	}
}

func TestEngageWaitingJSONIncludesAgeDays(t *testing.T) {
	dir := setupDir(t)
	now := time.Now()

	stale := nowItem("20260417-wait_aged", model.KindWaitingFor, model.StatusActive)
	stale.UpdatedAt = now.Add(-10 * 24 * time.Hour)
	stale.CreatedAt = stale.UpdatedAt
	writeItem(t, dir, stale, "")

	out, _, err := runCmd(t, dir, "--json", "engage", "waiting")
	if err != nil {
		t.Fatalf("engage waiting --json: %v", err)
	}

	var arr []map[string]any
	if err := json.Unmarshal([]byte(out), &arr); err != nil {
		t.Fatalf("parse json: %v (out=%q)", err, out)
	}
	if len(arr) != 1 {
		t.Fatalf("want 1 item, got %d", len(arr))
	}
	age, ok := arr[0]["age_days"].(float64)
	if !ok {
		t.Fatalf("age_days missing or wrong type: %v", arr[0])
	}
	if age < 9 || age > 11 {
		t.Errorf("age_days: want ~10, got %v", age)
	}
}

// ---------- reflect ----------

func TestReflectNextActions(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260417-na1", model.KindNextAction, model.StatusActive), "")
	writeItem(t, dir, nowItem("20260417-na2", model.KindNextAction, model.StatusDone), "")
	writeItem(t, dir, nowItem("20260417-inbox1", model.KindInbox, model.StatusActive), "")

	out, _, err := runCmd(t, dir, "reflect", "next-actions")
	if err != nil {
		t.Fatalf("reflect next-actions: %v", err)
	}
	if !strings.Contains(out, "20260417-na1") {
		t.Errorf("missing active next action: %q", out)
	}
	if strings.Contains(out, "20260417-na2") {
		t.Errorf("should not include done item: %q", out)
	}
	if strings.Contains(out, "20260417-inbox1") {
		t.Errorf("should not include inbox item: %q", out)
	}
}

func TestReflectNextActionsSortsByDueAt(t *testing.T) {
	dir := setupDir(t)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterday := today.Add(-24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)

	a := nowItem("20260420-r_tomorrow", model.KindNextAction, model.StatusActive)
	a.DueAt = &tomorrow
	b := nowItem("20260420-r_yesterday", model.KindNextAction, model.StatusActive)
	b.DueAt = &yesterday
	c := nowItem("20260420-r_nodue", model.KindNextAction, model.StatusActive)
	writeItem(t, dir, a, "")
	writeItem(t, dir, b, "")
	writeItem(t, dir, c, "")

	out, _, err := runCmd(t, dir, "reflect", "next-actions")
	if err != nil {
		t.Fatalf("reflect next-actions: %v", err)
	}
	want := []string{"20260420-r_yesterday", "20260420-r_tomorrow", "20260420-r_nodue"}
	var positions []int
	for _, id := range want {
		idx := strings.Index(out, id)
		if idx < 0 {
			t.Fatalf("missing %s: %q", id, out)
		}
		positions = append(positions, idx)
	}
	for i := 1; i < len(positions); i++ {
		if positions[i] < positions[i-1] {
			t.Errorf("sort order: %s should appear after %s\nout=%q", want[i], want[i-1], out)
		}
	}
}

func TestReflectNextActionsShowsDueAtColumn(t *testing.T) {
	dir := setupDir(t)
	due := time.Date(2026, 4, 20, 0, 0, 0, 0, time.Local)
	it := nowItem("20260420-r_dated", model.KindNextAction, model.StatusActive)
	it.DueAt = &due
	writeItem(t, dir, it, "")

	out, _, err := runCmd(t, dir, "reflect", "next-actions")
	if err != nil {
		t.Fatalf("reflect next-actions: %v", err)
	}
	if !strings.Contains(out, "DUE_AT") {
		t.Errorf("header should include DUE_AT column: %q", out)
	}
	if !strings.Contains(out, "2026-04-20") {
		t.Errorf("due_at value should appear: %q", out)
	}
}

func TestReflectProjects(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260417-proj1", model.KindProject, model.StatusActive), "")

	out, _, err := runCmd(t, dir, "reflect", "projects")
	if err != nil {
		t.Fatalf("reflect projects: %v", err)
	}
	if !strings.Contains(out, "20260417-proj1") {
		t.Errorf("missing project: %q", out)
	}
}

func TestReflectProjectsStalled(t *testing.T) {
	dir := setupDir(t)
	proj := nowItem("20260417-stalled_proj", model.KindProject, model.StatusActive)
	active := nowItem("20260417-active_proj", model.KindProject, model.StatusActive)
	na := nowItem("20260417-linked_na", model.KindNextAction, model.StatusActive)
	na.Project = "20260417-active_proj"

	writeItem(t, dir, proj, "")
	writeItem(t, dir, active, "")
	writeItem(t, dir, na, "")

	out, _, err := runCmd(t, dir, "reflect", "projects", "--stalled")
	if err != nil {
		t.Fatalf("reflect projects --stalled: %v", err)
	}
	if !strings.Contains(out, "20260417-stalled_proj") {
		t.Errorf("missing stalled project: %q", out)
	}
	if strings.Contains(out, "20260417-active_proj") {
		t.Errorf("should not include non-stalled project: %q", out)
	}
}

func TestReflectWaiting(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260417-wait1", model.KindWaitingFor, model.StatusActive), "")

	out, _, err := runCmd(t, dir, "reflect", "waiting")
	if err != nil {
		t.Fatalf("reflect waiting: %v", err)
	}
	if !strings.Contains(out, "20260417-wait1") {
		t.Errorf("missing waiting item: %q", out)
	}
}

func TestReflectLogDefault(t *testing.T) {
	dir := setupDir(t)
	done := nowItem("20260417-log_done", model.KindNextAction, model.StatusDone)
	canceled := nowItem("20260417-log_canceled", model.KindNextAction, model.StatusCanceled)
	writeItem(t, dir, done, "")
	writeItem(t, dir, canceled, "")

	out, _, err := runCmd(t, dir, "reflect", "log", "--since", "2026-01-01")
	if err != nil {
		t.Fatalf("reflect log: %v", err)
	}
	if !strings.Contains(out, "20260417-log_done") {
		t.Errorf("missing done item: %q", out)
	}
	if strings.Contains(out, "20260417-log_canceled") {
		t.Errorf("default status filter should exclude canceled: %q", out)
	}
	if !strings.Contains(out, "UPDATED_AT") {
		t.Errorf("log output should include UPDATED_AT header: %q", out)
	}
}

func TestReflectLogKindFilter(t *testing.T) {
	dir := setupDir(t)
	na := nowItem("20260417-log_na", model.KindNextAction, model.StatusDone)
	proj := nowItem("20260417-log_proj", model.KindProject, model.StatusDone)
	writeItem(t, dir, na, "")
	writeItem(t, dir, proj, "")

	out, _, err := runCmd(t, dir, "reflect", "log",
		"--since", "2026-01-01", "--kind", "project")
	if err != nil {
		t.Fatalf("reflect log --kind: %v", err)
	}
	if !strings.Contains(out, "20260417-log_proj") {
		t.Errorf("missing project item: %q", out)
	}
	if strings.Contains(out, "20260417-log_na") {
		t.Errorf("kind filter should exclude next_action: %q", out)
	}
}

func TestReflectLogTagFilter(t *testing.T) {
	dir := setupDir(t)
	tagged := nowItem("20260417-log_tagged", model.KindNextAction, model.StatusDone)
	tagged.Tags = []string{"cli", "docs"}
	other := nowItem("20260417-log_other", model.KindNextAction, model.StatusDone)
	other.Tags = []string{"cli"}
	writeItem(t, dir, tagged, "")
	writeItem(t, dir, other, "")

	out, _, err := runCmd(t, dir, "reflect", "log",
		"--since", "2026-01-01", "--tag", "cli", "--tag", "docs")
	if err != nil {
		t.Fatalf("reflect log --tag: %v", err)
	}
	if !strings.Contains(out, "20260417-log_tagged") {
		t.Errorf("missing all-tags match: %q", out)
	}
	if strings.Contains(out, "20260417-log_other") {
		t.Errorf("partial tag match should be excluded: %q", out)
	}
}

func TestReflectLogStatusMulti(t *testing.T) {
	dir := setupDir(t)
	done := nowItem("20260417-log_d", model.KindNextAction, model.StatusDone)
	canceled := nowItem("20260417-log_c", model.KindNextAction, model.StatusCanceled)
	archived := nowItem("20260417-log_a", model.KindNextAction, model.StatusArchived)
	writeItem(t, dir, done, "")
	writeItem(t, dir, canceled, "")
	writeItem(t, dir, archived, "")

	out, _, err := runCmd(t, dir, "reflect", "log",
		"--since", "2026-01-01",
		"--status", "done", "--status", "canceled")
	if err != nil {
		t.Fatalf("reflect log --status: %v", err)
	}
	if !strings.Contains(out, "20260417-log_d") || !strings.Contains(out, "20260417-log_c") {
		t.Errorf("multi-status should include done and canceled: %q", out)
	}
	if strings.Contains(out, "20260417-log_a") {
		t.Errorf("archived should be excluded: %q", out)
	}
}

func TestReflectLogSinceBoundary(t *testing.T) {
	dir := setupDir(t)
	old := &model.Item{
		ID: "20260101-old_done", Title: "old", Kind: model.KindNextAction, Status: model.StatusDone,
		CreatedAt: time.Date(2026, 1, 1, 9, 0, 0, 0, time.Local),
		UpdatedAt: time.Date(2026, 1, 1, 9, 0, 0, 0, time.Local),
	}
	recent := nowItem("20260417-recent_done", model.KindNextAction, model.StatusDone)
	writeItem(t, dir, old, "")
	writeItem(t, dir, recent, "")

	out, _, err := runCmd(t, dir, "reflect", "log", "--since", "2026-04-01")
	if err != nil {
		t.Fatalf("reflect log --since: %v", err)
	}
	if !strings.Contains(out, "20260417-recent_done") {
		t.Errorf("recent item should be included: %q", out)
	}
	if strings.Contains(out, "20260101-old_done") {
		t.Errorf("item before --since should be excluded: %q", out)
	}
}

func TestReflectLogInvalidStatus(t *testing.T) {
	dir := setupDir(t)
	_, stderr, err := runCmd(t, dir, "reflect", "log",
		"--since", "2026-01-01", "--status", "active")
	if err == nil {
		t.Fatalf("expected error for non-terminal status, got nil; stderr=%q", stderr)
	}
}

func TestReflectTickler(t *testing.T) {
	dir := setupDir(t)
	now := time.Now()
	past := now.Add(-24 * time.Hour)
	future := now.Add(24 * time.Hour)

	due := nowItem("20260417-tick_due", model.KindTickler, model.StatusActive)
	due.DeferUntil = &past
	writeItem(t, dir, due, "")

	notYet := nowItem("20260417-tick_future", model.KindTickler, model.StatusActive)
	notYet.DeferUntil = &future
	writeItem(t, dir, notYet, "")

	noDate := nowItem("20260417-tick_nodate", model.KindTickler, model.StatusActive)
	writeItem(t, dir, noDate, "")

	out, _, err := runCmd(t, dir, "reflect", "tickler")
	if err != nil {
		t.Fatalf("reflect tickler: %v", err)
	}
	if !strings.Contains(out, "20260417-tick_due") {
		t.Errorf("missing due tickler: %q", out)
	}
	if strings.Contains(out, "20260417-tick_future") {
		t.Errorf("should not include future tickler: %q", out)
	}
	if strings.Contains(out, "20260417-tick_nodate") {
		t.Errorf("should not include dateless tickler: %q", out)
	}
}

func TestReflectTicklerReviewAtFallback(t *testing.T) {
	dir := setupDir(t)
	now := time.Now()
	past := now.Add(-24 * time.Hour)

	tick := nowItem("20260417-tick_review", model.KindTickler, model.StatusActive)
	tick.ReviewAt = &past
	writeItem(t, dir, tick, "")

	out, _, err := runCmd(t, dir, "reflect", "tickler")
	if err != nil {
		t.Fatalf("reflect tickler: %v", err)
	}
	if !strings.Contains(out, "20260417-tick_review") {
		t.Errorf("missing review_at-triggered tickler: %q", out)
	}
}

func TestReflectTicklerPull(t *testing.T) {
	dir := setupDir(t)
	now := time.Now()
	past := now.Add(-24 * time.Hour)
	future := now.Add(24 * time.Hour)
	reviewDate := now.Add(-48 * time.Hour)

	fired := nowItem("20260417-pull_fired", model.KindTickler, model.StatusActive)
	fired.DeferUntil = &past
	fired.ReviewAt = &reviewDate
	fired.CreatedAt = now.Add(-72 * time.Hour)
	fired.UpdatedAt = now.Add(-72 * time.Hour)
	writeItem(t, dir, fired, "body content")

	waiting := nowItem("20260417-pull_future", model.KindTickler, model.StatusActive)
	waiting.DeferUntil = &future
	writeItem(t, dir, waiting, "")

	out, _, err := runCmd(t, dir, "reflect", "tickler", "--pull")
	if err != nil {
		t.Fatalf("reflect tickler --pull: %v", err)
	}
	if !strings.Contains(out, "20260417-pull_fired") {
		t.Errorf("pulled ID not printed: %q", out)
	}
	if strings.Contains(out, "20260417-pull_future") {
		t.Errorf("non-fired tickler should not be pulled: %q", out)
	}

	moved, body := readItem(t, dir, "20260417-pull_fired")
	if moved.Kind != model.KindInbox {
		t.Errorf("kind: want inbox, got %s", moved.Kind)
	}
	if moved.DeferUntil != nil {
		t.Errorf("defer_until should be cleared, got %v", moved.DeferUntil)
	}
	if moved.ReviewAt == nil {
		t.Errorf("review_at should be preserved")
	}
	if !moved.UpdatedAt.After(fired.UpdatedAt) {
		t.Errorf("updated_at should be refreshed")
	}
	if body != "body content" {
		t.Errorf("body: want %q, got %q", "body content", body)
	}

	cfg := config.New(dir)
	oldPath := filepath.Join(cfg.DirForKind(model.KindTickler), "20260417-pull_fired.md")
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("old tickler file should be gone, stat err=%v", err)
	}

	stillThere, _ := readItem(t, dir, "20260417-pull_future")
	if stillThere.Kind != model.KindTickler {
		t.Errorf("non-fired tickler moved: kind=%s", stillThere.Kind)
	}
}

func TestReflectTicklerPullEmpty(t *testing.T) {
	dir := setupDir(t)
	future := time.Now().Add(24 * time.Hour)

	pending := nowItem("20260417-tick_pending", model.KindTickler, model.StatusActive)
	pending.DeferUntil = &future
	writeItem(t, dir, pending, "")

	out, _, err := runCmd(t, dir, "reflect", "tickler", "--pull")
	if err != nil {
		t.Fatalf("reflect tickler --pull: %v", err)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("expected empty output, got %q", out)
	}
}

func TestReflectTicklerPullJSON(t *testing.T) {
	dir := setupDir(t)
	past := time.Now().Add(-24 * time.Hour)

	fired := nowItem("20260417-json_fired", model.KindTickler, model.StatusActive)
	fired.DeferUntil = &past
	writeItem(t, dir, fired, "")

	out, _, err := runCmd(t, dir, "--json", "reflect", "tickler", "--pull")
	if err != nil {
		t.Fatalf("reflect tickler --pull --json: %v", err)
	}
	var got struct {
		Pulled []string `json:"pulled"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("json parse: %v; out=%q", err, out)
	}
	if len(got.Pulled) != 1 || got.Pulled[0] != "20260417-json_fired" {
		t.Errorf("pulled: want [20260417-json_fired], got %v", got.Pulled)
	}
}

func TestReflectTicklerPullJSONEmpty(t *testing.T) {
	dir := setupDir(t)
	out, _, err := runCmd(t, dir, "--json", "reflect", "tickler", "--pull")
	if err != nil {
		t.Fatalf("reflect tickler --pull --json: %v", err)
	}
	var got struct {
		Pulled []string `json:"pulled"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("json parse: %v; out=%q", err, out)
	}
	if got.Pulled == nil {
		t.Errorf("pulled should be [] not null: %q", out)
	}
	if len(got.Pulled) != 0 {
		t.Errorf("pulled should be empty, got %v", got.Pulled)
	}
}

// ---------- item ----------

func TestItemGet(t *testing.T) {
	dir := setupDir(t)
	item := nowItem("20260417-get_me", model.KindProject, model.StatusActive)
	item.Title = "Get Me"
	writeItem(t, dir, item, "")

	out, _, err := runCmd(t, dir, "item", "get", "20260417-get_me")
	if err != nil {
		t.Fatalf("item get: %v", err)
	}
	if !strings.Contains(out, "Get Me") {
		t.Errorf("item get missing title: %q", out)
	}
}

func TestItemGetShowsRefsText(t *testing.T) {
	dir := setupDir(t)
	item := nowItem("20260417-show_refs", model.KindNextAction, model.StatusActive)
	item.Refs = []string{"https://example.com/pr/1", "https://notion.so/x"}
	writeItem(t, dir, item, "")

	out, _, err := runCmd(t, dir, "item", "get", "20260417-show_refs")
	if err != nil {
		t.Fatalf("item get: %v", err)
	}
	if !strings.Contains(out, "refs:") {
		t.Errorf("item get missing refs label: %q", out)
	}
	if !strings.Contains(out, "https://example.com/pr/1") || !strings.Contains(out, "https://notion.so/x") {
		t.Errorf("item get missing refs values: %q", out)
	}
}

func TestItemGetJSONIncludesRefs(t *testing.T) {
	dir := setupDir(t)
	item := nowItem("20260417-json_refs", model.KindNextAction, model.StatusActive)
	item.Refs = []string{"https://example.com/pr/1", "https://notion.so/x"}
	writeItem(t, dir, item, "")

	out, _, err := runCmd(t, dir, "--json", "item", "get", "20260417-json_refs")
	if err != nil {
		t.Fatalf("item get --json: %v", err)
	}
	var got struct {
		Refs []string `json:"refs"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, out)
	}
	if len(got.Refs) != 2 || got.Refs[0] != "https://example.com/pr/1" || got.Refs[1] != "https://notion.so/x" {
		t.Errorf("refs: got %v", got.Refs)
	}
}

func TestItemGetJSONOmitsRefsWhenEmpty(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260417-no_refs", model.KindInbox, model.StatusActive), "")

	out, _, err := runCmd(t, dir, "--json", "item", "get", "20260417-no_refs")
	if err != nil {
		t.Fatalf("item get --json: %v", err)
	}
	if strings.Contains(out, `"refs"`) {
		t.Errorf("expected no refs key in JSON when empty: %s", out)
	}
}

func TestItemList(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260417-list1", model.KindInbox, model.StatusActive), "")
	writeItem(t, dir, nowItem("20260417-list2", model.KindNextAction, model.StatusActive), "")

	out, _, err := runCmd(t, dir, "item", "list")
	if err != nil {
		t.Fatalf("item list: %v", err)
	}
	if !strings.Contains(out, "20260417-list1") || !strings.Contains(out, "20260417-list2") {
		t.Errorf("item list missing items: %q", out)
	}
}

func TestItemListFilterKind(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260417-inbox1", model.KindInbox, model.StatusActive), "")
	writeItem(t, dir, nowItem("20260417-next1", model.KindNextAction, model.StatusActive), "")

	out, _, err := runCmd(t, dir, "item", "list", "--kind", "inbox")
	if err != nil {
		t.Fatalf("item list --kind: %v", err)
	}
	if !strings.Contains(out, "20260417-inbox1") {
		t.Errorf("missing inbox item: %q", out)
	}
	if strings.Contains(out, "20260417-next1") {
		t.Errorf("should not contain next_action item: %q", out)
	}
}

func TestItemListQueryUnfielded(t *testing.T) {
	dir := setupDir(t)
	a := nowItem("20260417-fix_panic", model.KindInbox, model.StatusActive)
	a.Title = "Fix panic in parser"
	b := nowItem("20260417-write_docs", model.KindInbox, model.StatusActive)
	b.Title = "Write query docs"
	writeItem(t, dir, a, "")
	writeItem(t, dir, b, "")

	out, _, err := runCmd(t, dir, "item", "list", "--query", "panic")
	if err != nil {
		t.Fatalf("item list --query: %v", err)
	}
	if !strings.Contains(out, "20260417-fix_panic") {
		t.Errorf("expected fix_panic in output: %q", out)
	}
	if strings.Contains(out, "20260417-write_docs") {
		t.Errorf("unexpected write_docs in output: %q", out)
	}
}

func TestItemListQueryFieldedTitleQuoted(t *testing.T) {
	dir := setupDir(t)
	a := nowItem("20260417-fix_parser_panic", model.KindInbox, model.StatusActive)
	a.Title = "Fix parser panic"
	b := nowItem("20260417-refactor_parser", model.KindInbox, model.StatusActive)
	b.Title = "Refactor parser"
	writeItem(t, dir, a, "")
	writeItem(t, dir, b, "")

	out, _, err := runCmd(t, dir, "item", "list", "--query", `title:"fix parser"`)
	if err != nil {
		t.Fatalf("item list --query: %v", err)
	}
	if !strings.Contains(out, "20260417-fix_parser_panic") {
		t.Errorf("expected fix_parser_panic in output: %q", out)
	}
	if strings.Contains(out, "20260417-refactor_parser") {
		t.Errorf("unexpected refactor_parser in output: %q", out)
	}
}

func TestItemListQueryUnfieldedMatchesBody(t *testing.T) {
	dir := setupDir(t)
	a := nowItem("20260417-body_hit", model.KindInbox, model.StatusActive)
	a.Title = "Task A"
	b := nowItem("20260417-body_miss", model.KindInbox, model.StatusActive)
	b.Title = "Task B"
	writeItem(t, dir, a, "Reproduces on macOS during startup.")
	writeItem(t, dir, b, "Not relevant.")

	out, _, err := runCmd(t, dir, "item", "list", "--query", "macos")
	if err != nil {
		t.Fatalf("item list --query: %v", err)
	}
	if !strings.Contains(out, "20260417-body_hit") {
		t.Errorf("expected body_hit in output: %q", out)
	}
	if strings.Contains(out, "20260417-body_miss") {
		t.Errorf("unexpected body_miss in output: %q", out)
	}
}

func TestItemListQueryTagArrayAny(t *testing.T) {
	dir := setupDir(t)
	a := nowItem("20260417-bug_item", model.KindInbox, model.StatusActive)
	a.Tags = []string{"cli", "docs"}
	writeItem(t, dir, a, "")

	out, _, err := runCmd(t, dir, "item", "list", "--query", "tag:do")
	if err != nil {
		t.Fatalf("item list --query: %v", err)
	}
	if !strings.Contains(out, "20260417-bug_item") {
		t.Errorf("expected bug_item (tags contain 'docs') in output: %q", out)
	}
}

func TestItemListQueryRefArrayAny(t *testing.T) {
	dir := setupDir(t)
	a := nowItem("20260417-ref_hit", model.KindInbox, model.StatusActive)
	a.Refs = []string{"https://github.com/foo/bar/pull/42"}
	b := nowItem("20260417-ref_miss", model.KindInbox, model.StatusActive)
	b.Refs = []string{"https://notion.so/xyz"}
	writeItem(t, dir, a, "")
	writeItem(t, dir, b, "")

	out, _, err := runCmd(t, dir, "item", "list", "--query", "ref:github.com")
	if err != nil {
		t.Fatalf("item list --query: %v", err)
	}
	if !strings.Contains(out, "20260417-ref_hit") {
		t.Errorf("expected ref_hit in output: %q", out)
	}
	if strings.Contains(out, "20260417-ref_miss") {
		t.Errorf("unexpected ref_miss in output: %q", out)
	}
}

func TestItemListQueryBooleanWithExplicitStatus(t *testing.T) {
	dir := setupDir(t)
	a := nowItem("20260417-bug_active", model.KindInbox, model.StatusActive)
	a.Tags = []string{"bug"}
	b := nowItem("20260417-cli_done", model.KindNextAction, model.StatusDone)
	b.Tags = []string{"cli"}
	c := nowItem("20260417-bug_done", model.KindNextAction, model.StatusDone)
	c.Tags = []string{"bug"}
	writeItem(t, dir, a, "")
	writeItem(t, dir, b, "")
	writeItem(t, dir, c, "")

	// (tag:bug OR tag:cli) NOT status:done — across all statuses.
	out, _, err := runCmd(t, dir, "item", "list",
		"--status", "", // disable default status=active so archive is scanned
		"--query", "(tag:bug OR tag:cli) NOT status:done")
	if err != nil {
		t.Fatalf("item list --query: %v", err)
	}
	if !strings.Contains(out, "20260417-bug_active") {
		t.Errorf("expected bug_active in output: %q", out)
	}
	if strings.Contains(out, "20260417-cli_done") || strings.Contains(out, "20260417-bug_done") {
		t.Errorf("unexpected done items in output: %q", out)
	}
}

func TestItemListQueryComposesWithFlags(t *testing.T) {
	dir := setupDir(t)
	a := nowItem("20260417-inbox_foo", model.KindInbox, model.StatusActive)
	a.Title = "foo in inbox"
	b := nowItem("20260417-next_foo", model.KindNextAction, model.StatusActive)
	b.Title = "foo in next"
	writeItem(t, dir, a, "")
	writeItem(t, dir, b, "")

	out, _, err := runCmd(t, dir, "item", "list", "--kind", "inbox", "--query", "foo")
	if err != nil {
		t.Fatalf("item list --kind --query: %v", err)
	}
	if !strings.Contains(out, "20260417-inbox_foo") {
		t.Errorf("expected inbox_foo in output: %q", out)
	}
	if strings.Contains(out, "20260417-next_foo") {
		t.Errorf("unexpected next_foo (filtered out by --kind): %q", out)
	}
}

func TestItemListQueryEmptyMatchesAll(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260417-list1", model.KindInbox, model.StatusActive), "")
	writeItem(t, dir, nowItem("20260417-list2", model.KindNextAction, model.StatusActive), "")

	out, _, err := runCmd(t, dir, "item", "list", "--query", "")
	if err != nil {
		t.Fatalf("item list --query '': %v", err)
	}
	if !strings.Contains(out, "20260417-list1") || !strings.Contains(out, "20260417-list2") {
		t.Errorf("expected both items with empty --query: %q", out)
	}
}

func TestItemListQueryInvalidExpression(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260417-any", model.KindInbox, model.StatusActive), "")

	cases := []struct {
		name  string
		query string
	}{
		{"unterminated quote", `title:"oops`},
		{"unknown field", "nope:foo"},
		{"trailing operator", "foo OR"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, _, err := runCmd(t, dir, "item", "list", "--query", c.query)
			if err == nil {
				t.Fatalf("expected error for invalid query %q", c.query)
			}
			if !strings.Contains(err.Error(), "invalid --query") {
				t.Errorf("expected 'invalid --query' in error, got: %v", err)
			}
		})
	}
}

func TestItemUpdate(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260417-upd", model.KindInbox, model.StatusActive), "")

	_, _, err := runCmd(t, dir, "item", "update", "20260417-upd", "title=Updated Title")
	if err != nil {
		t.Fatalf("item update: %v", err)
	}

	got, _ := readItem(t, dir, "20260417-upd")
	if got.Title != "Updated Title" {
		t.Errorf("title: want %q, got %q", "Updated Title", got.Title)
	}
}

func TestItemUpdateKindMovesFile(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260417-kindchg", model.KindInbox, model.StatusActive), "")

	_, _, err := runCmd(t, dir, "item", "update", "20260417-kindchg", "kind=next_action")
	if err != nil {
		t.Fatalf("item update kind: %v", err)
	}

	got, _ := readItem(t, dir, "20260417-kindchg")
	if got.Kind != model.KindNextAction {
		t.Errorf("kind: want next_action, got %q", got.Kind)
	}
}

func TestItemUpdateRefsField(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260417-refset", model.KindNextAction, model.StatusActive), "")

	_, _, err := runCmd(t, dir, "item", "update", "20260417-refset",
		"refs=[https://a.example, https://b.example, https://c.example]")
	if err != nil {
		t.Fatalf("item update refs=[...]: %v", err)
	}
	got, _ := readItem(t, dir, "20260417-refset")
	want := []string{"https://a.example", "https://b.example", "https://c.example"}
	if len(got.Refs) != len(want) {
		t.Fatalf("refs: want %d, got %d (%v)", len(want), len(got.Refs), got.Refs)
	}
	for i, w := range want {
		if got.Refs[i] != w {
			t.Errorf("refs[%d]: want %q, got %q", i, w, got.Refs[i])
		}
	}

	_, _, err = runCmd(t, dir, "item", "update", "20260417-refset", "refs=")
	if err != nil {
		t.Fatalf("item update refs=: %v", err)
	}
	got, _ = readItem(t, dir, "20260417-refset")
	if len(got.Refs) != 0 {
		t.Errorf("refs: want empty after clear, got %v", got.Refs)
	}
}

func TestItemArchive(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260417-arch", model.KindNextAction, model.StatusActive), "")

	_, _, err := runCmd(t, dir, "item", "archive", "20260417-arch")
	if err != nil {
		t.Fatalf("item archive: %v", err)
	}

	got, _ := readItem(t, dir, "20260417-arch")
	if got.Status != model.StatusArchived {
		t.Errorf("status: want archived, got %q", got.Status)
	}
}

func TestItemRestoreDone(t *testing.T) {
	dir := setupDir(t)
	done := nowItem("20260417-restore_done", model.KindNextAction, model.StatusDone)
	writeItem(t, dir, done, "body preserved")

	_, _, err := runCmd(t, dir, "item", "restore", "20260417-restore_done")
	if err != nil {
		t.Fatalf("item restore: %v", err)
	}

	got, body := readItem(t, dir, "20260417-restore_done")
	if got.Status != model.StatusActive {
		t.Errorf("status: want active, got %q", got.Status)
	}
	if got.Kind != model.KindNextAction {
		t.Errorf("kind: want next_action, got %q", got.Kind)
	}
	if body != "body preserved" {
		t.Errorf("body: want %q, got %q", "body preserved", body)
	}
	if !got.UpdatedAt.After(done.UpdatedAt) {
		t.Errorf("updated_at should be refreshed")
	}

	cfg := config.New(dir)
	newPath := filepath.Join(cfg.DirForKind(model.KindNextAction), "20260417-restore_done.md")
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("restored item not at %q: %v", newPath, err)
	}
	archivePath := filepath.Join(cfg.ArchiveItemsDir(), "20260417-restore_done.md")
	if _, err := os.Stat(archivePath); !os.IsNotExist(err) {
		t.Errorf("archive copy should be removed, stat err=%v", err)
	}
}

func TestItemRestoreCanceled(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260417-restore_cancel", model.KindProject, model.StatusCanceled), "")

	_, _, err := runCmd(t, dir, "item", "restore", "20260417-restore_cancel")
	if err != nil {
		t.Fatalf("item restore: %v", err)
	}
	got, _ := readItem(t, dir, "20260417-restore_cancel")
	if got.Status != model.StatusActive {
		t.Errorf("status: want active, got %q", got.Status)
	}
	cfg := config.New(dir)
	newPath := filepath.Join(cfg.DirForKind(model.KindProject), "20260417-restore_cancel.md")
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("restored item not in project dir: %v", err)
	}
}

func TestItemRestoreDiscardedReturnsToInbox(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260417-restore_disc", model.KindInbox, model.StatusDiscarded), "")

	_, _, err := runCmd(t, dir, "item", "restore", "20260417-restore_disc")
	if err != nil {
		t.Fatalf("item restore: %v", err)
	}
	got, _ := readItem(t, dir, "20260417-restore_disc")
	if got.Status != model.StatusActive {
		t.Errorf("status: want active, got %q", got.Status)
	}
	if got.Kind != model.KindInbox {
		t.Errorf("kind: want inbox, got %q", got.Kind)
	}
	cfg := config.New(dir)
	newPath := filepath.Join(cfg.DirForKind(model.KindInbox), "20260417-restore_disc.md")
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("restored item not in inbox dir: %v", err)
	}
}

func TestItemRestoreArchived(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260417-restore_arch", model.KindSomeday, model.StatusArchived), "")

	_, _, err := runCmd(t, dir, "item", "restore", "20260417-restore_arch")
	if err != nil {
		t.Fatalf("item restore: %v", err)
	}
	got, _ := readItem(t, dir, "20260417-restore_arch")
	if got.Status != model.StatusActive {
		t.Errorf("status: want active, got %q", got.Status)
	}
}

func TestItemRestoreActiveFails(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260417-restore_live", model.KindNextAction, model.StatusActive), "")

	_, _, err := runCmd(t, dir, "item", "restore", "20260417-restore_live")
	if err == nil {
		t.Error("expected error when restoring an already-active item")
	}
}

func TestItemRestoreNotFound(t *testing.T) {
	dir := setupDir(t)
	_, _, err := runCmd(t, dir, "item", "restore", "20260417-ghost")
	if !store.IsNotFound(err) {
		t.Errorf("expected NotFoundError, got %v", err)
	}
}

// ---------- init ----------

func TestInitCreatesDirectories(t *testing.T) {
	dir := t.TempDir()

	out, _, err := runCmd(t, dir, "init")
	if err != nil {
		t.Fatalf("init: %v", err)
	}

	want := []string{
		filepath.Join(dir, "items", "inbox"),
		filepath.Join(dir, "items", "next_action"),
		filepath.Join(dir, "items", "project"),
		filepath.Join(dir, "items", "waiting_for"),
		filepath.Join(dir, "items", "someday"),
		filepath.Join(dir, "items", "tickler"),
		filepath.Join(dir, "archive", "items"),
		filepath.Join(dir, "archive", "reference"),
		filepath.Join(dir, "reference"),
	}

	gotLines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(gotLines) != len(want) {
		t.Fatalf("init lines: want %d, got %d (%q)", len(want), len(gotLines), out)
	}
	for i, w := range want {
		if gotLines[i] != w {
			t.Errorf("line %d: want %q, got %q", i, w, gotLines[i])
		}
		info, err := os.Stat(w)
		if err != nil {
			t.Errorf("stat %q: %v", w, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%q is not a directory", w)
		}
	}
}

func TestInitJSON(t *testing.T) {
	dir := t.TempDir()

	out, _, err := runCmd(t, dir, "--json", "init")
	if err != nil {
		t.Fatalf("init --json: %v", err)
	}

	var paths []string
	if err := json.Unmarshal([]byte(out), &paths); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}
	if len(paths) != 9 {
		t.Errorf("paths: want 9, got %d (%v)", len(paths), paths)
	}
	if paths[0] != filepath.Join(dir, "items", "inbox") {
		t.Errorf("paths[0]: want %q, got %q", filepath.Join(dir, "items", "inbox"), paths[0])
	}
}

func TestInitIdempotent(t *testing.T) {
	dir := t.TempDir()

	out1, _, err := runCmd(t, dir, "init")
	if err != nil {
		t.Fatalf("init first: %v", err)
	}
	out2, _, err := runCmd(t, dir, "init")
	if err != nil {
		t.Fatalf("init second: %v", err)
	}
	if out1 != out2 {
		t.Errorf("init not idempotent:\nfirst: %q\nsecond: %q", out1, out2)
	}
}

// ---------- completion ----------

func TestCompletionBash(t *testing.T) {
	dir := t.TempDir()
	out, _, err := runCmd(t, dir, "completion", "bash")
	if err != nil {
		t.Fatalf("completion bash: %v", err)
	}
	if !strings.Contains(out, "bash completion") {
		t.Errorf("bash completion output missing expected header; got first bytes: %q", firstN(out, 120))
	}
}

func TestCompletionZsh(t *testing.T) {
	dir := t.TempDir()
	out, _, err := runCmd(t, dir, "completion", "zsh")
	if err != nil {
		t.Fatalf("completion zsh: %v", err)
	}
	if !strings.Contains(out, "#compdef htd") {
		t.Errorf("zsh completion output missing #compdef htd; got first bytes: %q", firstN(out, 120))
	}
}

func TestCompletionDoesNotTouchPath(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := runCmd(t, dir, "completion", "bash"); err != nil {
		t.Fatalf("completion bash: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "items")); !os.IsNotExist(err) {
		t.Errorf("completion should not create items/ in --path target (stat err: %v)", err)
	}
}

// ---------- HTD_PATH env var ----------

func runCmdNoPath(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	root := command.NewRootCommand()
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), errOut.String(), err
}

func TestHTDPathEnvVar(t *testing.T) {
	dir := setupDir(t)
	t.Setenv("HTD_PATH", dir)

	out, _, err := runCmdNoPath(t, "capture", "add", "--title", "env-rooted")
	if err != nil {
		t.Fatalf("capture add: %v", err)
	}
	id := strings.TrimSpace(out)
	if id == "" {
		t.Fatal("expected ID")
	}
	p := filepath.Join(dir, "items", "inbox", id+".md")
	if _, err := os.Stat(p); err != nil {
		t.Errorf("expected item at %q, stat err: %v", p, err)
	}
}

func TestHTDPathFlagOverridesEnvVar(t *testing.T) {
	envDir := setupDir(t)
	flagDir := setupDir(t)
	t.Setenv("HTD_PATH", envDir)

	out, _, err := runCmd(t, flagDir, "capture", "add", "--title", "flag-wins")
	if err != nil {
		t.Fatalf("capture add: %v", err)
	}
	id := strings.TrimSpace(out)
	if id == "" {
		t.Fatal("expected ID")
	}

	inFlag := filepath.Join(flagDir, "items", "inbox", id+".md")
	if _, err := os.Stat(inFlag); err != nil {
		t.Errorf("expected item under flagDir, stat %q: %v", inFlag, err)
	}
	inEnv := filepath.Join(envDir, "items", "inbox", id+".md")
	if _, err := os.Stat(inEnv); !os.IsNotExist(err) {
		t.Errorf("did not expect item under envDir; stat %q: %v", inEnv, err)
	}
}

// ---------- verbose ----------

func TestMutationsSilentByDefault(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260421-mute", model.KindNextAction, model.StatusActive), "")

	out, errOut, err := runCmd(t, dir, "engage", "done", "20260421-mute")
	if err != nil {
		t.Fatalf("engage done: %v", err)
	}
	if out != "" {
		t.Errorf("default stdout should be silent, got %q", out)
	}
	if errOut != "" {
		t.Errorf("default stderr should be silent, got %q", errOut)
	}
}

func TestEngageDoneVerboseText(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260421-vd1", model.KindNextAction, model.StatusActive), "")
	writeItem(t, dir, nowItem("20260421-vd2", model.KindNextAction, model.StatusActive), "")

	out, _, err := runCmd(t, dir, "--verbose", "engage", "done", "20260421-vd1", "20260421-vd2")
	if err != nil {
		t.Fatalf("engage done --verbose: %v", err)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 lines, got %d: %q", len(lines), out)
	}
	want := []string{
		"updated 20260421-vd1: status=done",
		"updated 20260421-vd2: status=done",
	}
	for i, w := range want {
		if lines[i] != w {
			t.Errorf("line %d: want %q, got %q", i, w, lines[i])
		}
	}
}

func TestEngageDoneVerboseJSON(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260421-vj1", model.KindNextAction, model.StatusActive), "")
	writeItem(t, dir, nowItem("20260421-vj2", model.KindNextAction, model.StatusActive), "")

	out, _, err := runCmd(t, dir, "--verbose", "--json", "engage", "done", "20260421-vj1", "20260421-vj2")
	if err != nil {
		t.Fatalf("engage done --verbose --json: %v", err)
	}
	var arr []map[string]any
	if err := json.Unmarshal([]byte(out), &arr); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}
	if len(arr) != 2 {
		t.Fatalf("want 2 items, got %d", len(arr))
	}
	for i, id := range []string{"20260421-vj1", "20260421-vj2"} {
		if arr[i]["id"] != id {
			t.Errorf("item %d id: want %q, got %v", i, id, arr[i]["id"])
		}
		if arr[i]["status"] != "done" {
			t.Errorf("item %d status: want done, got %v", i, arr[i]["status"])
		}
	}
}

func TestOrganizeScheduleVerboseShowsRFC3339(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260421-vs", model.KindNextAction, model.StatusActive), "")

	out, _, err := runCmd(t, dir, "-v", "organize", "schedule", "20260421-vs", "--defer", "2026-04-27")
	if err != nil {
		t.Fatalf("organize schedule -v: %v", err)
	}
	line := strings.TrimRight(out, "\n")
	if !strings.HasPrefix(line, "updated 20260421-vs: defer_until=2026-04-27T00:00:00") {
		t.Errorf("expected defer_until expanded to RFC3339 in verbose output, got %q", line)
	}
}

func TestOrganizeMoveVerbose(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260421-vm", model.KindInbox, model.StatusActive), "")

	out, _, err := runCmd(t, dir, "-v", "organize", "move", "next_action", "20260421-vm")
	if err != nil {
		t.Fatalf("organize move -v: %v", err)
	}
	want := "updated 20260421-vm: kind=next_action\n"
	if out != want {
		t.Errorf("want %q, got %q", want, out)
	}
}

func TestOrganizeLinkVerbose(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260421-vp", model.KindProject, model.StatusActive), "")
	writeItem(t, dir, nowItem("20260421-vt", model.KindNextAction, model.StatusActive), "")

	out, _, err := runCmd(t, dir, "-v", "organize", "link", "20260421-vt", "--project", "20260421-vp")
	if err != nil {
		t.Fatalf("organize link -v: %v", err)
	}
	want := "updated 20260421-vt: project=20260421-vp\n"
	if out != want {
		t.Errorf("want %q, got %q", want, out)
	}
}

func TestClarifyDiscardVerbose(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260421-vcd", model.KindInbox, model.StatusActive), "")

	out, _, err := runCmd(t, dir, "-v", "clarify", "discard", "20260421-vcd")
	if err != nil {
		t.Fatalf("clarify discard -v: %v", err)
	}
	want := "updated 20260421-vcd: status=discarded\n"
	if out != want {
		t.Errorf("want %q, got %q", want, out)
	}
}

func TestClarifyUpdateVerbose(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260421-vcu", model.KindInbox, model.StatusActive), "")

	out, _, err := runCmd(t, dir, "-v", "clarify", "update", "20260421-vcu",
		"--title", "Renamed", "--ref", "https://example.com/1")
	if err != nil {
		t.Fatalf("clarify update -v: %v", err)
	}
	line := strings.TrimRight(out, "\n")
	if !strings.Contains(line, "title=Renamed") {
		t.Errorf("missing title change: %q", line)
	}
	if !strings.Contains(line, "refs=[https://example.com/1]") {
		t.Errorf("missing refs change: %q", line)
	}
}

func TestItemUpdateVerboseMultiField(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260421-viu", model.KindNextAction, model.StatusActive), "")

	out, _, err := runCmd(t, dir, "-v", "item", "update", "20260421-viu",
		"title=Renamed", "defer_until=2026-04-27", "tags=[cli,docs]")
	if err != nil {
		t.Fatalf("item update -v: %v", err)
	}
	line := strings.TrimRight(out, "\n")
	if !strings.Contains(line, "title=Renamed") {
		t.Errorf("missing title=Renamed: %q", line)
	}
	if !strings.Contains(line, "defer_until=2026-04-27T00:00:00") {
		t.Errorf("defer_until should expand to RFC3339: %q", line)
	}
	if !strings.Contains(line, "tags=[cli,docs]") {
		t.Errorf("tags should be normalized: %q", line)
	}
}

func TestItemArchiveVerbose(t *testing.T) {
	dir := setupDir(t)
	writeItem(t, dir, nowItem("20260421-via", model.KindNextAction, model.StatusActive), "")

	out, _, err := runCmd(t, dir, "-v", "item", "archive", "20260421-via")
	if err != nil {
		t.Fatalf("item archive -v: %v", err)
	}
	want := "updated 20260421-via: status=archived\n"
	if out != want {
		t.Errorf("want %q, got %q", want, out)
	}
}

func TestItemRestoreVerbose(t *testing.T) {
	dir := setupDir(t)
	it := nowItem("20260421-vir", model.KindNextAction, model.StatusDone)
	writeItem(t, dir, it, "")

	out, _, err := runCmd(t, dir, "-v", "item", "restore", "20260421-vir")
	if err != nil {
		t.Fatalf("item restore -v: %v", err)
	}
	want := "updated 20260421-vir: status=active\n"
	if out != want {
		t.Errorf("want %q, got %q", want, out)
	}
}

// ---------- helper ----------

func firstN(s string, n int) string {
	if len(s) < n {
		return s
	}
	return s[:n]
}


func readItemFromPath(t *testing.T, path string) (*model.Item, error) {
	t.Helper()
	item, _, err := store.Read(path)
	return item, err
}

// ensure cobra command is accessible
var _ *cobra.Command
