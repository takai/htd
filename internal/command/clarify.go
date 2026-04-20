package command

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/takai/htd/internal/model"
	"github.com/takai/htd/internal/store"
)

func newClarifyCommand(c *container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clarify",
		Short: "Process inbox items",
	}
	cmd.AddCommand(
		newClarifyListCommand(c),
		newClarifyShowCommand(c),
		newClarifyUpdateCommand(c),
		newClarifyDiscardCommand(c),
	)
	return cmd
}

func newClarifyListCommand(c *container) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all inbox items",
		RunE: func(cmd *cobra.Command, args []string) error {
			kind := model.KindInbox
			status := model.StatusActive
			items, err := store.List(c.cfg, store.Filter{Kind: &kind, Status: &status})
			if err != nil {
				return err
			}
			c.printer.PrintItems(items)
			return nil
		},
	}
}

func newClarifyShowCommand(c *container) *cobra.Command {
	return &cobra.Command{
		Use:   "show ID",
		Short: "Display a single inbox item",
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
			if item.Kind != model.KindInbox {
				return &store.NotFoundError{ID: itemID}
			}
			c.printer.PrintItem(item, body)
			return nil
		},
	}
}

func newClarifyUpdateCommand(c *container) *cobra.Command {
	var (
		title string
		body  string
		refs  []string
	)

	cmd := &cobra.Command{
		Use:   "update ID",
		Short: "Update an inbox item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("title") && !cmd.Flags().Changed("body") && !cmd.Flags().Changed("ref") {
				return fmt.Errorf("at least one of --title, --body, or --ref must be provided")
			}
			itemID := args[0]
			path, err := store.FindItem(c.cfg, itemID)
			if err != nil {
				return err
			}
			item, existingBody, err := store.Read(path)
			if err != nil {
				return err
			}
			if item.Kind != model.KindInbox {
				return &store.NotFoundError{ID: itemID}
			}
			if cmd.Flags().Changed("title") {
				item.Title = title
			}
			if cmd.Flags().Changed("body") {
				existingBody = body
			}
			if cmd.Flags().Changed("ref") {
				if len(refs) == 0 {
					item.Refs = nil
				} else {
					item.Refs = refs
				}
			}
			item.UpdatedAt = time.Now()
			return store.Write(path, item, existingBody)
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "New title")
	cmd.Flags().StringVar(&body, "body", "", "New body content")
	cmd.Flags().StringArrayVar(&refs, "ref", nil, "New reference URLs (repeatable; replaces existing refs)")
	return cmd
}

func newClarifyDiscardCommand(c *container) *cobra.Command {
	return &cobra.Command{
		Use:   "discard ID",
		Short: "Discard an inbox item",
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
			if item.Kind != model.KindInbox {
				return fmt.Errorf("item %q is not in inbox (kind: %s); use 'engage cancel' instead", itemID, item.Kind)
			}
			item.Status = model.StatusDiscarded
			item.UpdatedAt = time.Now()
			newPath := store.PathForItem(c.cfg, item)
			return store.Move(path, newPath, item, body)
		},
	}
}
