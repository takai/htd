package command

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/takai/htd/internal/id"
	"github.com/takai/htd/internal/model"
	"github.com/takai/htd/internal/output"
	"github.com/takai/htd/internal/store"
)

// Journal types accepted by `htd journal new --type`.
const (
	journalTypeDaily  = "daily"
	journalTypeWeekly = "weekly"
	journalTypeAdhoc  = "adhoc"
)

func newJournalCommand(c *container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "journal",
		Short: "Manage time-stamped journals and retros",
	}
	cmd.AddCommand(
		newJournalNewCommand(c),
		newJournalListCommand(c),
		newJournalShowCommand(c),
	)
	return cmd
}

const journalNewLong = `Create a new journal entry.

Three types are supported:
  daily   YYYY-MM-DD.md          (defaults to today)
  weekly  YYYY-MM-DD-weekly.md   (defaults to Monday of the current ISO week)
  adhoc   <slug>.md              (slug derived from --title; date metadata
                                  recorded in frontmatter only)

The file is written with optional YAML frontmatter (created_at, updated_at,
tags) plus a small Markdown scaffold. Existing files are not overwritten.`

func newJournalNewCommand(c *container) *cobra.Command {
	var (
		journalType string
		dateStr     string
		title       string
		tags        []string
	)
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new journal entry",
		Long:  journalNewLong,
		RunE: func(cmd *cobra.Command, args []string) error {
			now := time.Now()
			d, err := resolveJournalDate(dateStr, now, journalType)
			if err != nil {
				return err
			}
			name, body, err := buildJournalEntry(journalType, d, title)
			if err != nil {
				return err
			}
			path := store.PathForJournal(c.cfg, name)
			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("journal %q already exists at %s", name, path)
			}
			j := &model.Journal{
				CreatedAt: now,
				UpdatedAt: now,
				Tags:      tags,
			}
			if len(tags) == 0 {
				j.Tags = nil
			}
			if err := store.WriteJournal(path, j, body); err != nil {
				return err
			}
			c.printer.PrintID(name)
			return nil
		},
	}
	cmd.Flags().StringVar(&journalType, "type", journalTypeDaily, "Entry type: daily, weekly, or adhoc")
	cmd.Flags().StringVar(&dateStr, "date", "", "Date (YYYY-MM-DD). Defaults to today; for weekly entries, snaps to Monday of that week.")
	cmd.Flags().StringVar(&title, "title", "", "Title for ad-hoc entries (required when --type=adhoc)")
	cmd.Flags().StringArrayVar(&tags, "tag", nil, "Tag (repeatable)")
	return cmd
}

func newJournalListCommand(c *container) *cobra.Command {
	var sinceStr string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List journal entries (most recent first)",
		RunE: func(cmd *cobra.Command, args []string) error {
			var since time.Time
			if sinceStr != "" {
				t, err := time.ParseInLocation("2006-01-02", sinceStr, time.Local)
				if err != nil {
					return fmt.Errorf("invalid --since %q (want YYYY-MM-DD)", sinceStr)
				}
				since = t
			}
			entries, err := store.ListJournals(c.cfg, since)
			if err != nil {
				return err
			}
			out := make([]output.JournalListEntry, len(entries))
			for i, e := range entries {
				out[i] = output.JournalListEntry{Name: e.Name, Journal: e.Journal}
			}
			c.printer.PrintJournals(out)
			return nil
		},
	}
	cmd.Flags().StringVar(&sinceStr, "since", "", "Only show entries created on or after this date (YYYY-MM-DD)")
	return cmd
}

func newJournalShowCommand(c *container) *cobra.Command {
	return &cobra.Command{
		Use:   "show NAME",
		Short: "Show a single journal entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			path, err := store.FindJournal(c.cfg, name)
			if err != nil {
				return err
			}
			j, body, err := store.ReadJournal(path)
			if err != nil {
				return err
			}
			c.printer.PrintJournal(name, j, body)
			return nil
		},
	}
}

// resolveJournalDate parses dateStr (or returns now) and snaps to Monday for
// weekly entries.
func resolveJournalDate(dateStr string, now time.Time, journalType string) (time.Time, error) {
	var d time.Time
	if dateStr == "" {
		d = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	} else {
		t, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid --date %q (want YYYY-MM-DD)", dateStr)
		}
		d = t
	}
	if journalType == journalTypeWeekly {
		d = snapToMonday(d)
	}
	return d, nil
}

// snapToMonday returns the Monday of the ISO week containing d (in d's
// timezone).
func snapToMonday(d time.Time) time.Time {
	weekday := int(d.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday → 7 in ISO terms
	}
	return d.AddDate(0, 0, 1-weekday)
}

// buildJournalEntry derives the filename (without extension) and template
// body for the given type/date/title.
func buildJournalEntry(journalType string, d time.Time, title string) (name, body string, err error) {
	switch journalType {
	case journalTypeDaily:
		name = d.Format("2006-01-02")
		body = fmt.Sprintf("# %s\n\n## What I did\n\n## What I learned\n\n## Tomorrow\n", name)
	case journalTypeWeekly:
		name = d.Format("2006-01-02") + "-weekly"
		body = fmt.Sprintf(
			"# Week of %s\n\n## Wins\n\n## Misses\n\n## Lessons\n\n## Focus next week\n",
			d.Format("2006-01-02"),
		)
	case journalTypeAdhoc:
		if title == "" {
			return "", "", fmt.Errorf("--title is required for --type=adhoc")
		}
		slug := id.Generate(title, d)
		// Drop the date prefix from the slug for ad-hoc filenames; the
		// frontmatter carries the date instead.
		// id.Generate returns "YYYYMMDD-<slug>"; keep only the slug part so
		// the filename reads like a topic.
		if idx := strings.Index(slug, "-"); idx >= 0 {
			slug = slug[idx+1:]
		}
		if slug == "" {
			return "", "", fmt.Errorf("--title produced an empty slug")
		}
		name = slug
		body = fmt.Sprintf("# %s\n", title)
	default:
		return "", "", fmt.Errorf("invalid --type %q (want daily, weekly, or adhoc)", journalType)
	}
	return name, body, nil
}
