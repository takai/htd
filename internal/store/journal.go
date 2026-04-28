package store

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/takai/htd/internal/config"
	"github.com/takai/htd/internal/model"
)

// JournalEntry bundles a journal file's metadata, body, and filename. The
// filename (without extension) is the identifier used by `htd journal show`.
type JournalEntry struct {
	Name    string
	Journal *model.Journal
	Body    string
}

// EntityJournal is the NotFoundError discriminator for journals.
const EntityJournal EntityKind = "journal"

// ReadJournal reads a journal Markdown file at path. Files without YAML
// front matter are accepted: the metadata struct is zero-valued and the
// entire file becomes the body.
func ReadJournal(path string) (*model.Journal, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	if !hasFrontmatter(data) {
		return &model.Journal{}, string(data), nil
	}
	yamlBytes, body := splitFrontmatter(data)
	var j model.Journal
	if err := yaml.Unmarshal(yamlBytes, &j); err != nil {
		return nil, "", err
	}
	return &j, body, nil
}

// WriteJournal writes a journal entry to path atomically. When j is nil, the
// file is written without a frontmatter block (plain Markdown).
func WriteJournal(path string, j *model.Journal, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var data []byte
	if j == nil {
		data = []byte(ensureTrailingNewline(body))
	} else {
		var err error
		data, err = marshalFrontmatter(j, body)
		if err != nil {
			return err
		}
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// PathForJournal returns the canonical path for a journal entry given its
// name (filename without `.md` extension).
func PathForJournal(cfg *config.Config, name string) string {
	return filepath.Join(cfg.JournalDir(), name+".md")
}

// FindJournal locates a journal entry by name. Returns NotFoundError when
// no matching file exists.
func FindJournal(cfg *config.Config, name string) (string, error) {
	p := PathForJournal(cfg, name)
	if _, err := os.Stat(p); err == nil {
		return p, nil
	}
	return "", &NotFoundError{Kind: EntityJournal, ID: name}
}

// ListJournals returns every entry under journal/, sorted by filename
// descending (most recent first when names are date-prefixed). When since is
// non-zero, entries whose `created_at` (or filename-derived date as a
// fallback) is earlier than since are filtered out.
func ListJournals(cfg *config.Config, since time.Time) ([]JournalEntry, error) {
	entries, err := os.ReadDir(cfg.JournalDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []JournalEntry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		path := filepath.Join(cfg.JournalDir(), e.Name())
		j, body, err := ReadJournal(path)
		if err != nil {
			return nil, err
		}
		if !since.IsZero() {
			// Prefer the filename-derived date for filtering: the
			// filename is the source of truth for *when* the journal is
			// about. created_at is wall-clock (when the file was
			// written), which often differs from the entry's logical
			// date (e.g., backfilling yesterday's daily this morning).
			ts := inferDateFromName(name)
			if ts.IsZero() {
				ts = j.CreatedAt
			}
			if !ts.IsZero() && ts.Before(since) {
				continue
			}
		}
		out = append(out, JournalEntry{Name: name, Journal: j, Body: body})
	}
	sort.SliceStable(out, func(i, k int) bool {
		return out[i].Name > out[k].Name
	})
	return out, nil
}

// hasFrontmatter reports whether data begins with a YAML front matter
// delimiter. Permits a single leading blank line.
func hasFrontmatter(data []byte) bool {
	s := string(data)
	s = strings.TrimLeft(s, "\n")
	return strings.HasPrefix(s, "---\n") || s == "---" || strings.HasPrefix(s, "---\r\n")
}

// inferDateFromName extracts the leading YYYY-MM-DD prefix from name and
// returns it as midnight local time. Returns zero Time when name has no
// recognizable date prefix (e.g. ad-hoc slugs).
func inferDateFromName(name string) time.Time {
	if len(name) < 10 {
		return time.Time{}
	}
	t, err := time.ParseInLocation("2006-01-02", name[:10], time.Local)
	if err != nil {
		return time.Time{}
	}
	return t
}

func ensureTrailingNewline(s string) string {
	if s == "" {
		return ""
	}
	if strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}
