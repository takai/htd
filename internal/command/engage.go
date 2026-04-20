package command

import (
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/takai/htd/internal/model"
	"github.com/takai/htd/internal/store"
)

func newEngageCommand(c *container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "engage",
		Short: "Choose and complete work",
	}
	cmd.AddCommand(
		newEngageDoneCommand(c),
		newEngageCancelCommand(c),
		newEngageNextActionCommand(c),
		newEngageWaitingCommand(c),
	)
	return cmd
}

func newEngageDoneCommand(c *container) *cobra.Command {
	return &cobra.Command{
		Use:   "done ID",
		Short: "Mark an item as completed",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return terminateItem(c, args[0], model.StatusDone)
		},
	}
}

func newEngageCancelCommand(c *container) *cobra.Command {
	return &cobra.Command{
		Use:   "cancel ID",
		Short: "Cancel an active item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return terminateItem(c, args[0], model.StatusCanceled)
		},
	}
}

func newEngageNextActionCommand(c *container) *cobra.Command {
	var (
		projectID string
		tags      []string
	)

	cmd := &cobra.Command{
		Use:   "next-action",
		Short: "List next actions ready to work on now",
		RunE: func(cmd *cobra.Command, args []string) error {
			kind := model.KindNextAction
			status := model.StatusActive
			items, err := store.List(c.cfg, store.Filter{
				Kind:      &kind,
				Status:    &status,
				ProjectID: projectID,
			})
			if err != nil {
				return err
			}
			now := time.Now()
			var visible []*model.Item
			for _, it := range items {
				if it.DeferUntil != nil && it.DeferUntil.After(now) {
					continue
				}
				if !matchAllTags(it, tags) {
					continue
				}
				visible = append(visible, it)
			}
			sort.Slice(visible, func(i, j int) bool {
				if visible[i].DueAt == nil {
					return false
				}
				if visible[j].DueAt == nil {
					return true
				}
				return visible[i].DueAt.Before(*visible[j].DueAt)
			})
			c.printer.PrintNextActionItems(visible)
			return nil
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "Filter by project ID")
	cmd.Flags().StringSliceVar(&tags, "tag", nil, "Filter by tag (repeatable)")
	return cmd
}

func newEngageWaitingCommand(c *container) *cobra.Command {
	var staleDays int

	cmd := &cobra.Command{
		Use:   "waiting",
		Short: "List waiting-for items that need follow-up",
		RunE: func(cmd *cobra.Command, args []string) error {
			kind := model.KindWaitingFor
			status := model.StatusActive
			items, err := store.List(c.cfg, store.Filter{Kind: &kind, Status: &status})
			if err != nil {
				return err
			}
			now := time.Now()
			threshold := time.Duration(staleDays) * 24 * time.Hour

			type aged struct {
				item *model.Item
				days int
			}
			var stale []aged
			for _, it := range items {
				ref := it.UpdatedAt
				if ref.IsZero() {
					ref = it.CreatedAt
				}
				age := now.Sub(ref)
				if age < threshold {
					continue
				}
				stale = append(stale, aged{item: it, days: int(age / (24 * time.Hour))})
			}
			sort.Slice(stale, func(i, j int) bool {
				return stale[i].days > stale[j].days
			})

			outItems := make([]*model.Item, len(stale))
			ageDays := make([]int, len(stale))
			for i, a := range stale {
				outItems[i] = a.item
				ageDays[i] = a.days
			}
			c.printer.PrintWaitingItems(outItems, ageDays)
			return nil
		},
	}

	cmd.Flags().IntVar(&staleDays, "stale-days", 7, "Stale threshold in days (items older than this are shown)")
	return cmd
}

func matchAllTags(it *model.Item, tags []string) bool {
	for _, t := range tags {
		if !slices.Contains(it.Tags, t) {
			return false
		}
	}
	return true
}

func terminateItem(c *container, itemID string, status model.Status) error {
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
	item.Status = status
	item.UpdatedAt = time.Now()
	newPath := store.PathForItem(c.cfg, item)
	return store.Move(path, newPath, item, body)
}
