package command

import (
	"fmt"
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
