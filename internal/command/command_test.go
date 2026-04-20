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

	_, _, err := runCmd(t, dir, "organize", "move", "20260417-move_me", "next_action")
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

	_, _, err := runCmd(t, dir, "organize", "move", "20260417-some", "inbox")
	if err == nil {
		t.Error("expected error when moving to inbox")
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

func TestEngageTickler(t *testing.T) {
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

	out, _, err := runCmd(t, dir, "engage", "tickler")
	if err != nil {
		t.Fatalf("engage tickler: %v", err)
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

func TestEngageTicklerReviewAtFallback(t *testing.T) {
	dir := setupDir(t)
	now := time.Now()
	past := now.Add(-24 * time.Hour)

	tick := nowItem("20260417-tick_review", model.KindTickler, model.StatusActive)
	tick.ReviewAt = &past
	writeItem(t, dir, tick, "")

	out, _, err := runCmd(t, dir, "engage", "tickler")
	if err != nil {
		t.Fatalf("engage tickler: %v", err)
	}
	if !strings.Contains(out, "20260417-tick_review") {
		t.Errorf("missing review_at-triggered tickler: %q", out)
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

func TestReflectDone(t *testing.T) {
	dir := setupDir(t)
	item := nowItem("20260417-done_item", model.KindNextAction, model.StatusDone)
	writeItem(t, dir, item, "")

	out, _, err := runCmd(t, dir, "reflect", "done", "--since", "2026-01-01")
	if err != nil {
		t.Fatalf("reflect done: %v", err)
	}
	if !strings.Contains(out, "20260417-done_item") {
		t.Errorf("missing done item: %q", out)
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
