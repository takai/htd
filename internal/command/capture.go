package command

import (
	"fmt"
	"slices"
	"time"

	"github.com/spf13/cobra"

	"github.com/takai/htd/internal/model"
	"github.com/takai/htd/internal/store"
)

func newCaptureCommand(c *container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "capture",
		Short: "Capture inputs into the inbox",
	}
	cmd.AddCommand(newCaptureAddCommand(c))
	return cmd
}

func newCaptureAddCommand(c *container) *cobra.Command {
	var (
		title    string
		body     string
		source   string
		tags     []string
		refs     []string
		kindFlag string
		children []string
		done     bool
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new item to the inbox",
		RunE: func(cmd *cobra.Command, args []string) error {
			if title == "" {
				return fmt.Errorf("--title is required")
			}
			if done && kindFlag != "" {
				return fmt.Errorf("--done and --kind are mutually exclusive")
			}
			if done && len(children) > 0 {
				return fmt.Errorf("--done and --child are mutually exclusive")
			}

			kind := model.KindInbox
			status := model.StatusActive
			if done {
				// --done lands the item directly as a completed next_action,
				// bypassing the inbox. The item is routed to archive/items/
				// by store.PathForItem because its status is terminal.
				kind = model.KindNextAction
				status = model.StatusDone
			} else if kindFlag != "" {
				k := model.Kind(kindFlag)
				if k == model.KindInbox {
					return fmt.Errorf("--kind inbox is redundant; omit --kind to capture into the inbox")
				}
				if !slices.Contains(model.ValidKinds(), k) {
					return fmt.Errorf("invalid kind %q", kindFlag)
				}
				kind = k
			}

			if len(children) > 0 {
				if kind != model.KindProject {
					return fmt.Errorf("--child requires --kind project")
				}
				if slices.Contains(children, "") {
					return fmt.Errorf("--child title must not be empty")
				}
			}

			now := time.Now()
			itemID := generateUniqueID(c, title, now)

			item := &model.Item{
				ID:        itemID,
				Title:     title,
				Kind:      kind,
				Status:    status,
				Source:    source,
				Tags:      tags,
				Refs:      refs,
				CreatedAt: now,
				UpdatedAt: now,
			}
			if len(tags) == 0 {
				item.Tags = nil
			}
			if len(refs) == 0 {
				item.Refs = nil
			}

			path := store.PathForItem(c.cfg, item)
			if err := store.Write(path, item, body); err != nil {
				return err
			}

			if len(children) == 0 {
				c.printer.PrintID(itemID)
				return nil
			}

			childIDs := make([]string, 0, len(children))
			for _, childTitle := range children {
				childID := generateUniqueID(c, childTitle, now)
				child := &model.Item{
					ID:        childID,
					Title:     childTitle,
					Kind:      model.KindNextAction,
					Status:    model.StatusActive,
					Project:   item.ID,
					CreatedAt: now,
					UpdatedAt: now,
				}
				childPath := store.PathForItem(c.cfg, child)
				if err := store.Write(childPath, child, ""); err != nil {
					return err
				}
				childIDs = append(childIDs, childID)
			}
			c.printer.PrintPromote(item.ID, childIDs)
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Short description (required)")
	cmd.Flags().StringVar(&body, "body", "", "Detailed description (Markdown)")
	cmd.Flags().StringVar(&source, "source", "", "Origin of the item")
	cmd.Flags().StringArrayVar(&tags, "tag", nil, "Tag (repeatable)")
	cmd.Flags().StringArrayVar(&refs, "ref", nil, "External reference URL (repeatable)")
	cmd.Flags().StringVar(&kindFlag, "kind", "", "Land directly as this kind instead of inbox (next_action, project, waiting_for, someday, tickler)")
	cmd.Flags().StringArrayVar(&children, "child", nil, "Child next-action title to create and link (requires --kind project; repeatable)")
	cmd.Flags().BoolVar(&done, "done", false, "Capture the item as already completed (lands in archive/items/ with kind=next_action, status=done)")

	return cmd
}

