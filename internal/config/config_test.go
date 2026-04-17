package config_test

import (
	"path/filepath"
	"testing"

	"github.com/takai/htd/internal/config"
	"github.com/takai/htd/internal/model"
)

func TestDirForKind(t *testing.T) {
	cfg := config.New("/root")
	cases := []struct {
		kind model.Kind
		want string
	}{
		{model.KindInbox, "/root/items/inbox"},
		{model.KindNextAction, "/root/items/next_action"},
		{model.KindProject, "/root/items/project"},
		{model.KindWaitingFor, "/root/items/waiting_for"},
		{model.KindSomeday, "/root/items/someday"},
		{model.KindTickler, "/root/items/tickler"},
	}
	for _, c := range cases {
		got := cfg.DirForKind(c.kind)
		if got != filepath.FromSlash(c.want) {
			t.Errorf("DirForKind(%q) = %q, want %q", c.kind, got, c.want)
		}
	}
}

func TestArchiveItemsDir(t *testing.T) {
	cfg := config.New("/root")
	want := filepath.Join("/root", "archive", "items")
	if got := cfg.ArchiveItemsDir(); got != want {
		t.Errorf("ArchiveItemsDir() = %q, want %q", got, want)
	}
}

func TestReferenceDir(t *testing.T) {
	cfg := config.New("/root")
	want := filepath.Join("/root", "reference")
	if got := cfg.ReferenceDir(); got != want {
		t.Errorf("ReferenceDir() = %q, want %q", got, want)
	}
}

func TestAllDirs(t *testing.T) {
	cfg := config.New("/root")
	dirs := cfg.AllDirs()
	// Must contain dirs for all kinds, archive/items, archive/reference, reference
	required := []string{
		filepath.Join("/root", "items", "inbox"),
		filepath.Join("/root", "items", "next_action"),
		filepath.Join("/root", "items", "project"),
		filepath.Join("/root", "items", "waiting_for"),
		filepath.Join("/root", "items", "someday"),
		filepath.Join("/root", "items", "tickler"),
		filepath.Join("/root", "archive", "items"),
		filepath.Join("/root", "archive", "reference"),
		filepath.Join("/root", "reference"),
	}
	dirSet := make(map[string]bool, len(dirs))
	for _, d := range dirs {
		dirSet[d] = true
	}
	for _, r := range required {
		if !dirSet[r] {
			t.Errorf("AllDirs() missing %q", r)
		}
	}
}
