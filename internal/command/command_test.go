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

// ---------- helper ----------

func readItemFromPath(t *testing.T, path string) (*model.Item, error) {
	t.Helper()
	item, _, err := store.Read(path)
	return item, err
}

// ensure cobra command is accessible
var _ *cobra.Command
