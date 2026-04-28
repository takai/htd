package command

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/takai/htd/internal/config"
	"github.com/takai/htd/internal/id"
	"github.com/takai/htd/internal/model"
	"github.com/takai/htd/internal/output"
	"github.com/takai/htd/internal/store"
)

// defaultReferenceTool is the value used when --tool is not supplied.
// `claude` is the most common AI-assistant target today; users with other
// tools can pass --tool explicitly.
const defaultReferenceTool = "claude"

func newReferenceCommand(c *container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reference",
		Short: "Manage tool-scoped reference notes",
	}
	cmd.AddCommand(
		newReferenceAddCommand(c),
		newReferenceGetCommand(c),
		newReferenceListCommand(c),
		newReferenceUpdateCommand(c),
		newReferenceArchiveCommand(c),
		newReferenceRestoreCommand(c),
		newReferenceReindexCommand(c),
	)
	return cmd
}

// addToolFlag registers the standard --tool flag on cmd. Default is the
// hardcoded `claude`; this lives in one place so future env-var support is a
// single edit.
func addToolFlag(cmd *cobra.Command, tool *string) {
	cmd.Flags().StringVar(tool, "tool", defaultReferenceTool, "Tool namespace (subdirectory under reference/)")
}

func newReferenceAddCommand(c *container) *cobra.Command {
	var (
		title string
		body  string
		tool  string
		tags  []string
	)
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new reference",
		RunE: func(cmd *cobra.Command, args []string) error {
			if title == "" {
				return fmt.Errorf("--title is required")
			}
			if err := store.EnsureReferenceToolDir(c.cfg, tool); err != nil {
				return err
			}
			now := time.Now()
			refID := generateUniqueReferenceID(c, title, now)
			ref := &model.Reference{
				ID:        refID,
				Title:     title,
				CreatedAt: now,
				UpdatedAt: now,
				Tags:      tags,
			}
			if len(tags) == 0 {
				ref.Tags = nil
			}
			path := store.PathForReferenceActive(c.cfg, tool, refID)
			if err := store.WriteRef(path, ref, body); err != nil {
				return err
			}
			if err := store.WriteIndex(c.cfg, tool); err != nil {
				return err
			}
			c.printer.PrintID(refID)
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "Short description (required)")
	cmd.Flags().StringVar(&body, "body", "", "Body content (Markdown)")
	cmd.Flags().StringArrayVar(&tags, "tag", nil, "Tag (repeatable). Use type:user|feedback|project|reference for INDEX.md grouping.")
	addToolFlag(cmd, &tool)
	return cmd
}

func newReferenceGetCommand(c *container) *cobra.Command {
	return &cobra.Command{
		Use:   "get ID",
		Short: "Retrieve a single reference by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := store.FindReference(c.cfg, args[0])
			if err != nil {
				return err
			}
			ref, body, err := store.ReadRef(res.Path)
			if err != nil {
				return err
			}
			c.printer.PrintReference(ref, body, res.Tool, res.Archived)
			return nil
		},
	}
}

func newReferenceListCommand(c *container) *cobra.Command {
	var (
		tool     string
		tag      string
		archived bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List references for a tool",
		RunE: func(cmd *cobra.Command, args []string) error {
			refs, err := listReferencesForView(c.cfg, tool, archived)
			if err != nil {
				return err
			}
			entries := make([]output.ReferenceListEntry, 0, len(refs))
			for _, r := range refs {
				if tag != "" && !containsRefTag(r.Reference.Tags, tag) {
					continue
				}
				entries = append(entries, output.ReferenceListEntry{
					Reference: r.Reference,
					Tool:      r.Tool,
					Archived:  r.Archived,
				})
			}
			c.printer.PrintReferences(entries)
			return nil
		},
	}
	addToolFlag(cmd, &tool)
	cmd.Flags().StringVar(&tag, "tag", "", "Filter by tag")
	cmd.Flags().BoolVar(&archived, "archived", false, "List archived references instead of active ones")
	return cmd
}

// listReferencesForView returns active references when archived=false, or
// archived ones when archived=true. The active list does not bleed into the
// archived view and vice versa, matching the docs/cli.md §8.3 contract.
func listReferencesForView(cfg *config.Config, tool string, archived bool) ([]store.ReferenceWithBody, error) {
	if !archived {
		return store.ListReferences(cfg, tool, false)
	}
	all, err := store.ListReferences(cfg, tool, true)
	if err != nil {
		return nil, err
	}
	out := make([]store.ReferenceWithBody, 0, len(all))
	for _, r := range all {
		if r.Archived {
			out = append(out, r)
		}
	}
	return out, nil
}

const referenceUpdateLong = `Update fields on a reference.

Each argument is a FIELD=VALUE pair. Multiple pairs are applied in order and
written atomically in a single file update.

Supported fields:
  title  Short description.
  body   Body content (Markdown; the content after front matter).
  tags   Comma-separated, optionally bracketed: foo,bar or [foo,bar].
         Pass tags= (empty) to clear.

Protected fields (cannot be changed): id, created_at, tool. To move a
reference between tools, archive and re-add.`

// referenceUpdateFields lists the field names accepted by `htd reference
// update`, in display order. Keep aligned with applyReferenceField.
var referenceUpdateFields = []string{"title", "body", "tags"}

func newReferenceUpdateCommand(c *container) *cobra.Command {
	return &cobra.Command{
		Use:   "update ID FIELD=VALUE...",
		Short: "Update fields on a reference",
		Long:  referenceUpdateLong,
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			refID := args[0]
			pairs := args[1:]

			res, err := store.FindReference(c.cfg, refID)
			if err != nil {
				return err
			}
			ref, body, err := store.ReadRef(res.Path)
			if err != nil {
				return err
			}

			var changes []output.Change
			for _, pair := range pairs {
				k, v, ok := strings.Cut(pair, "=")
				if !ok {
					return fmt.Errorf("invalid field assignment %q (expected KEY=VALUE)", pair)
				}
				if err := applyReferenceField(ref, &body, k, v); err != nil {
					return err
				}
				changes = append(changes, output.Change{Key: k, Value: normalizedReferenceFieldValue(ref, k, v)})
			}
			ref.UpdatedAt = time.Now()

			if err := store.WriteRef(res.Path, ref, body); err != nil {
				return err
			}
			// Active references regenerate the index. Archived ones don't
			// appear in INDEX.md, so skip the rewrite (no-op anyway).
			if !res.Archived {
				if err := store.WriteIndex(c.cfg, res.Tool); err != nil {
					return err
				}
			}
			c.printer.PrintReferenceUpdates([]output.ReferenceUpdate{{
				Reference: ref,
				Tool:      res.Tool,
				Archived:  res.Archived,
				Changes:   changes,
			}})
			return nil
		},
	}
}

func newReferenceArchiveCommand(c *container) *cobra.Command {
	return &cobra.Command{
		Use:   "archive ID",
		Short: "Archive a reference (move to archive/reference/<tool>/)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			refID := args[0]
			res, err := store.FindReference(c.cfg, refID)
			if err != nil {
				return err
			}
			if res.Archived {
				return fmt.Errorf("reference %q is already archived", refID)
			}
			ref, body, err := store.ReadRef(res.Path)
			if err != nil {
				return err
			}
			ref.UpdatedAt = time.Now()
			if err := store.ArchiveReference(c.cfg, res.Tool, ref, body); err != nil {
				return err
			}
			if err := store.WriteIndex(c.cfg, res.Tool); err != nil {
				return err
			}
			c.printer.PrintReferenceUpdates([]output.ReferenceUpdate{{
				Reference: ref,
				Tool:      res.Tool,
				Archived:  true,
				Changes:   []output.Change{{Key: "archived", Value: "true"}},
			}})
			return nil
		},
	}
}

func newReferenceRestoreCommand(c *container) *cobra.Command {
	return &cobra.Command{
		Use:   "restore ID",
		Short: "Restore an archived reference to active",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			refID := args[0]
			res, err := store.FindReference(c.cfg, refID)
			if err != nil {
				return err
			}
			if !res.Archived {
				return fmt.Errorf("reference %q is not archived", refID)
			}
			ref, body, err := store.ReadRef(res.Path)
			if err != nil {
				return err
			}
			ref.UpdatedAt = time.Now()
			if err := store.RestoreReference(c.cfg, res.Tool, ref, body); err != nil {
				return err
			}
			if err := store.WriteIndex(c.cfg, res.Tool); err != nil {
				return err
			}
			c.printer.PrintReferenceUpdates([]output.ReferenceUpdate{{
				Reference: ref,
				Tool:      res.Tool,
				Archived:  false,
				Changes:   []output.Change{{Key: "archived", Value: "false"}},
			}})
			return nil
		},
	}
}

func newReferenceReindexCommand(c *container) *cobra.Command {
	var tool string
	cmd := &cobra.Command{
		Use:   "reindex",
		Short: "Rewrite reference/<tool>/INDEX.md from the current set of references",
		RunE: func(cmd *cobra.Command, args []string) error {
			return store.WriteIndex(c.cfg, tool)
		},
	}
	addToolFlag(cmd, &tool)
	return cmd
}

func applyReferenceField(ref *model.Reference, body *string, key, value string) error {
	switch key {
	case "id", "created_at", "updated_at", "tool":
		return fmt.Errorf("field %q is protected and cannot be changed", key)
	case "title":
		ref.Title = value
	case "body":
		*body = value
	case "tags":
		ref.Tags = parseListField(value)
	default:
		return fmt.Errorf("unknown field %q (supported: %s)", key, strings.Join(referenceUpdateFields, ", "))
	}
	return nil
}

func normalizedReferenceFieldValue(ref *model.Reference, key, raw string) string {
	switch key {
	case "tags":
		return output.FormatList(ref.Tags)
	default:
		return raw
	}
}

// parseListField mirrors the YAML flow-style list parser used by `htd item
// update`: accepts `a,b`, `[a,b]`, or empty (cleared).
func parseListField(value string) []string {
	v := strings.TrimSpace(value)
	v = strings.TrimPrefix(v, "[")
	v = strings.TrimSuffix(v, "]")
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func containsRefTag(tags []string, want string) bool {
	return slices.Contains(tags, want)
}

// generateUniqueReferenceID returns an ID derived from title at time now,
// suffixed with _2, _3, ... when an item or any reference (active or
// archived, in any tool) already uses the base ID. Cross-checking items keeps
// IDs globally unique per docs/datamodel.md §5.2.
func generateUniqueReferenceID(c *container, title string, now time.Time) string {
	base := id.Generate(title, now)
	candidate := base
	for i := 2; ; i++ {
		if !referenceIDInUse(c, candidate) {
			return candidate
		}
		candidate = fmt.Sprintf("%s_%d", base, i)
	}
}

func referenceIDInUse(c *container, candidate string) bool {
	if _, err := store.FindItem(c.cfg, candidate); err == nil {
		return true
	}
	if store.ReferenceExists(c.cfg, candidate) {
		return true
	}
	return false
}
