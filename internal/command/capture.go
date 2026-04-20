package command

import (
	"fmt"
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
		title  string
		body   string
		source string
		tags   []string
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new item to the inbox",
		RunE: func(cmd *cobra.Command, args []string) error {
			if title == "" {
				return fmt.Errorf("--title is required")
			}
			now := time.Now()
			itemID := generateUniqueID(c, title, now)

			item := &model.Item{
				ID:        itemID,
				Title:     title,
				Kind:      model.KindInbox,
				Status:    model.StatusActive,
				Source:    source,
				Tags:      tags,
				CreatedAt: now,
				UpdatedAt: now,
			}
			if len(tags) == 0 {
				item.Tags = nil
			}

			path := store.PathForItem(c.cfg, item)
			if err := store.Write(path, item, body); err != nil {
				return err
			}
			c.printer.PrintID(itemID)
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Short description (required)")
	cmd.Flags().StringVar(&body, "body", "", "Detailed description (Markdown)")
	cmd.Flags().StringVar(&source, "source", "", "Origin of the item")
	cmd.Flags().StringArrayVar(&tags, "tag", nil, "Tag (repeatable)")

	return cmd
}

