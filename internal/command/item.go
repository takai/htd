package command

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/takai/htd/internal/model"
	"github.com/takai/htd/internal/output"
	"github.com/takai/htd/internal/query"
	"github.com/takai/htd/internal/store"
)

func newItemCommand(c *container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "item",
		Short: "Low-level CRUD access to items",
	}
	cmd.AddCommand(
		newItemGetCommand(c),
		newItemListCommand(c),
		newItemUpdateCommand(c),
		newItemArchiveCommand(c),
		newItemRestoreCommand(c),
	)
	return cmd
}

func newItemGetCommand(c *container) *cobra.Command {
	return &cobra.Command{
		Use:   "get ID",
		Short: "Retrieve a single item by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := store.FindItem(c.cfg, args[0])
			if err != nil {
				return err
			}
			item, body, err := store.Read(path)
			if err != nil {
				return err
			}
			c.printer.PrintItem(item, body)
			return nil
		},
	}
}

func newItemListCommand(c *container) *cobra.Command {
	var (
		kindStr   string
		statusStr string
		tag       string
		projectID string
		queryStr  string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List items with optional filters",
		RunE: func(cmd *cobra.Command, args []string) error {
			f := store.Filter{Tag: tag, ProjectID: projectID}

			if kindStr != "" {
				k := model.Kind(kindStr)
				if !isValidKind(k) {
					return fmt.Errorf("invalid kind %q", kindStr)
				}
				f.Kind = &k
			}

			if statusStr != "" {
				s := model.Status(statusStr)
				f.Status = &s
			} else {
				// Default to active items only
				s := model.StatusActive
				f.Status = &s
			}

			if queryStr == "" {
				items, err := store.List(c.cfg, f)
				if err != nil {
					return err
				}
				c.printer.PrintItems(items)
				return nil
			}

			q, err := query.Parse(queryStr)
			if err != nil {
				return fmt.Errorf("invalid --query: %w", err)
			}
			results, err := store.ListWithBody(c.cfg, f)
			if err != nil {
				return err
			}
			items := make([]*model.Item, 0, len(results))
			for _, r := range results {
				if q.Match(r.Item, r.Body) {
					items = append(items, r.Item)
				}
			}
			c.printer.PrintItems(items)
			return nil
		},
	}

	cmd.Flags().StringVar(&kindStr, "kind", "", "Filter by kind")
	cmd.Flags().StringVar(&statusStr, "status", "", "Filter by status (default: active)")
	cmd.Flags().StringVar(&tag, "tag", "", "Filter by tag")
	cmd.Flags().StringVar(&projectID, "project", "", "Filter by project ID")
	cmd.Flags().StringVar(&queryStr, "query", "", "Filter with query expression (see docs/cli.md §7.2.1)")
	return cmd
}

// itemUpdateFields lists the field names accepted by `htd item update`, in
// the order shown in help text. Keep this list aligned with applyField.
var itemUpdateFields = []string{
	"title", "body", "kind", "status", "project", "source",
	"tags", "refs", "due_at", "defer_until", "review_at",
}

const itemUpdateLong = `Update fields on an item.

Each argument is a FIELD=VALUE pair. Multiple pairs are applied in order and
written atomically in a single file update.

Supported fields:
  title        Short description.
  body         Detailed description (Markdown; the content after front matter).
  kind         One of: inbox, next_action, project, waiting_for, someday, tickler.
  status       One of: active, done, canceled, discarded, archived.
  project      ID of a project-kind item.
  source       Origin string (e.g., manual, email, slack).
  tags         Comma-separated list, optionally bracketed: foo,bar or [foo,bar].
               Pass tags= (empty) to clear.
  refs         Comma-separated URL list, same syntax as tags.
  due_at       YYYY-MM-DD or RFC3339 (e.g., 2026-05-01T14:30:00+09:00).
               Pass due_at= (empty) to clear.
  defer_until  Same format as due_at.
  review_at    Same format as due_at.

Protected fields (cannot be changed): id, created_at.

For the normal workflow, prefer the dedicated commands:
  organize schedule               set due_at / defer_until / review_at
  organize link / organize unlink set or clear project
  organize move                   change kind
  engage done / engage cancel     mark terminal
  item archive / item restore     other status transitions

Use item update for scripting, automation, and agent use where setting several
fields in one call is more convenient than invoking the workflow commands
separately.`

func newItemUpdateCommand(c *container) *cobra.Command {
	return &cobra.Command{
		Use:   "update ID FIELD=VALUE...",
		Short: "Update arbitrary fields on an item",
		Long:  itemUpdateLong,
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			itemID := args[0]
			pairs := args[1:]

			path, err := store.FindItem(c.cfg, itemID)
			if err != nil {
				return err
			}
			item, body, err := store.Read(path)
			if err != nil {
				return err
			}

			oldPath := store.PathForItem(c.cfg, item)

			var changes []output.Change
			for _, pair := range pairs {
				k, v, ok := strings.Cut(pair, "=")
				if !ok {
					return fmt.Errorf("invalid field assignment %q (expected KEY=VALUE)", pair)
				}
				if err := applyField(item, &body, k, v); err != nil {
					return err
				}
				changes = append(changes, output.Change{Key: k, Value: normalizedFieldValue(item, k, v)})
			}
			item.UpdatedAt = time.Now()

			newPath := store.PathForItem(c.cfg, item)
			if oldPath != newPath {
				if err := store.Move(oldPath, newPath, item, body); err != nil {
					return err
				}
			} else if err := store.Write(path, item, body); err != nil {
				return err
			}
			c.printer.PrintUpdates([]output.Update{{Item: item, Changes: changes}})
			return nil
		},
	}
}

// normalizedFieldValue returns the on-disk representation of the field after
// applyField ran, so verbose output shows the effective value (e.g., a
// date-only input expanded to RFC3339, or `[a,b]` for tag lists). Falls
// back to the raw input for free-form fields.
func normalizedFieldValue(item *model.Item, key, raw string) string {
	switch key {
	case "due_at":
		return output.FormatTimePtr(item.DueAt)
	case "defer_until":
		return output.FormatTimePtr(item.DeferUntil)
	case "review_at":
		return output.FormatTimePtr(item.ReviewAt)
	case "tags":
		return output.FormatList(item.Tags)
	case "refs":
		return output.FormatList(item.Refs)
	default:
		return raw
	}
}

func applyField(item *model.Item, body *string, key, value string) error {
	switch key {
	case "id", "created_at":
		return fmt.Errorf("field %q is protected and cannot be changed", key)
	case "title":
		item.Title = value
	case "kind":
		k := model.Kind(value)
		if !isValidKind(k) {
			return fmt.Errorf("invalid kind %q", value)
		}
		item.Kind = k
	case "status":
		s := model.Status(value)
		item.Status = s
	case "project":
		item.Project = value
	case "source":
		item.Source = value
	case "tags":
		// Parse YAML flow-style list: [a,b,c] or plain comma-separated
		v := strings.TrimSpace(value)
		v = strings.TrimPrefix(v, "[")
		v = strings.TrimSuffix(v, "]")
		if v == "" {
			item.Tags = nil
		} else {
			parts := strings.Split(v, ",")
			tags := make([]string, 0, len(parts))
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					tags = append(tags, p)
				}
			}
			item.Tags = tags
		}
	case "refs":
		// Parse YAML flow-style list: [a,b,c] or plain comma-separated
		v := strings.TrimSpace(value)
		v = strings.TrimPrefix(v, "[")
		v = strings.TrimSuffix(v, "]")
		if v == "" {
			item.Refs = nil
		} else {
			parts := strings.Split(v, ",")
			refs := make([]string, 0, len(parts))
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					refs = append(refs, p)
				}
			}
			item.Refs = refs
		}
	case "due_at":
		t, err := parseDate(value)
		if err != nil {
			return err
		}
		item.DueAt = t
	case "defer_until":
		t, err := parseDate(value)
		if err != nil {
			return err
		}
		item.DeferUntil = t
	case "review_at":
		t, err := parseDate(value)
		if err != nil {
			return err
		}
		item.ReviewAt = t
	case "body":
		*body = value
	default:
		return fmt.Errorf("unknown field %q (supported: %s)", key, strings.Join(itemUpdateFields, ", "))
	}
	return nil
}

func newItemArchiveCommand(c *container) *cobra.Command {
	return &cobra.Command{
		Use:   "archive ID",
		Short: "Archive an active item (last resort)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			itemID := args[0]
			path, err := store.FindItem(c.cfg, itemID)
			if err != nil {
				return err
			}
			item, body, err := store.Read(path)
			if err != nil {
				return err
			}
			if !model.IsActive(item.Status) {
				return fmt.Errorf("item %q is not active (status: %s)", itemID, item.Status)
			}
			item.Status = model.StatusArchived
			item.UpdatedAt = time.Now()
			newPath := store.PathForItem(c.cfg, item)
			if err := store.Move(path, newPath, item, body); err != nil {
				return err
			}
			c.printer.PrintUpdates([]output.Update{{
				Item:    item,
				Changes: []output.Change{{Key: "status", Value: string(model.StatusArchived)}},
			}})
			return nil
		},
	}
}

func newItemRestoreCommand(c *container) *cobra.Command {
	return &cobra.Command{
		Use:   "restore ID",
		Short: "Restore a terminal item to active status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			itemID := args[0]
			path, err := store.FindItem(c.cfg, itemID)
			if err != nil {
				return err
			}
			item, body, err := store.Read(path)
			if err != nil {
				return err
			}
			if model.IsActive(item.Status) {
				return fmt.Errorf("item %q is already active", itemID)
			}
			item.Status = model.StatusActive
			item.UpdatedAt = time.Now()
			newPath := store.PathForItem(c.cfg, item)
			if err := store.Move(path, newPath, item, body); err != nil {
				return err
			}
			c.printer.PrintUpdates([]output.Update{{
				Item:    item,
				Changes: []output.Change{{Key: "status", Value: string(model.StatusActive)}},
			}})
			return nil
		},
	}
}
