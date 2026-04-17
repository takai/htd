package command

import (
	"fmt"
	"slices"
	"time"

	"github.com/spf13/cobra"

	"github.com/takai/htd/internal/model"
	"github.com/takai/htd/internal/store"
)

func newOrganizeCommand(c *container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "organize",
		Short: "Categorize, link, and schedule items",
	}
	cmd.AddCommand(
		newOrganizeMoveCommand(c),
		newOrganizeLinkCommand(c),
		newOrganizeScheduleCommand(c),
	)
	return cmd
}

func newOrganizeMoveCommand(c *container) *cobra.Command {
	return &cobra.Command{
		Use:   "move ID KIND",
		Short: "Change an item's category",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			itemID := args[0]
			newKind := model.Kind(args[1])

			if newKind == model.KindInbox {
				return fmt.Errorf("cannot move items to inbox; items enter inbox only via 'capture add'")
			}
			if !isValidKind(newKind) {
				return fmt.Errorf("invalid kind %q", newKind)
			}

			path, err := store.FindItem(c.cfg, itemID)
			if err != nil {
				return err
			}
			item, body, err := store.Read(path)
			if err != nil {
				return err
			}
			if !model.IsActive(item.Status) {
				return fmt.Errorf("cannot move item with status %q", item.Status)
			}

			item.Kind = newKind
			item.UpdatedAt = time.Now()
			newPath := store.PathForItem(c.cfg, item)
			if path == newPath {
				return store.Write(path, item, body)
			}
			return store.Move(path, newPath, item, body)
		},
	}
}

func newOrganizeLinkCommand(c *container) *cobra.Command {
	var projectID string

	cmd := &cobra.Command{
		Use:   "link ID",
		Short: "Link an item to a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			itemID := args[0]

			if projectID != "" {
				projPath, err := store.FindItem(c.cfg, projectID)
				if err != nil {
					return fmt.Errorf("project %q not found: %w", projectID, err)
				}
				proj, _, err := store.Read(projPath)
				if err != nil {
					return err
				}
				if proj.Kind != model.KindProject {
					return fmt.Errorf("item %q is not a project (kind: %s)", projectID, proj.Kind)
				}
			}

			path, err := store.FindItem(c.cfg, itemID)
			if err != nil {
				return err
			}
			item, body, err := store.Read(path)
			if err != nil {
				return err
			}
			item.Project = projectID
			item.UpdatedAt = time.Now()
			return store.Write(path, item, body)
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "Project ID to link to (empty string to unlink)")
	_ = cmd.MarkFlagRequired("project")
	return cmd
}

func newOrganizeScheduleCommand(c *container) *cobra.Command {
	var (
		due    string
		defer_ string
		review string
	)

	cmd := &cobra.Command{
		Use:   "schedule ID",
		Short: "Set scheduling dates on an item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("due") && !cmd.Flags().Changed("defer") && !cmd.Flags().Changed("review") {
				return fmt.Errorf("at least one of --due, --defer, or --review must be provided")
			}

			itemID := args[0]
			path, err := store.FindItem(c.cfg, itemID)
			if err != nil {
				return err
			}
			item, body, err := store.Read(path)
			if err != nil {
				return err
			}

			if cmd.Flags().Changed("due") {
				t, err := parseDate(due)
				if err != nil {
					return fmt.Errorf("--due: %w", err)
				}
				item.DueAt = t
			}
			if cmd.Flags().Changed("defer") {
				t, err := parseDate(defer_)
				if err != nil {
					return fmt.Errorf("--defer: %w", err)
				}
				item.DeferUntil = t
			}
			if cmd.Flags().Changed("review") {
				t, err := parseDate(review)
				if err != nil {
					return fmt.Errorf("--review: %w", err)
				}
				item.ReviewAt = t
			}

			item.UpdatedAt = time.Now()
			return store.Write(path, item, body)
		},
	}

	cmd.Flags().StringVar(&due, "due", "", "Due date (YYYY-MM-DD or RFC3339)")
	cmd.Flags().StringVar(&defer_, "defer", "", "Defer-until date")
	cmd.Flags().StringVar(&review, "review", "", "Next review date")
	return cmd
}

// parseDate parses a date string, returning nil for empty string (clears the field).
func parseDate(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	// Try RFC3339 first
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return &t, nil
	}
	// Fall back to date-only (midnight local time)
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return nil, fmt.Errorf("cannot parse %q as date", s)
	}
	return &t, nil
}

func isValidKind(k model.Kind) bool {
	return slices.Contains(model.ValidKinds(), k)
}
