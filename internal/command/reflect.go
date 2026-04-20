package command

import (
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/takai/htd/internal/model"
	"github.com/takai/htd/internal/store"
)

func newReflectCommand(c *container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reflect",
		Short: "Review and inspect the state of the system",
	}
	cmd.AddCommand(
		newReflectNextActionsCommand(c),
		newReflectProjectsCommand(c),
		newReflectWaitingCommand(c),
		newReflectReviewCommand(c),
		newReflectLogCommand(c),
		newReflectTicklerCommand(c),
	)
	return cmd
}

func newReflectNextActionsCommand(c *container) *cobra.Command {
	return &cobra.Command{
		Use:   "next-actions",
		Short: "List all active next actions",
		RunE: func(cmd *cobra.Command, args []string) error {
			kind := model.KindNextAction
			status := model.StatusActive
			items, err := store.List(c.cfg, store.Filter{Kind: &kind, Status: &status})
			if err != nil {
				return err
			}
			now := time.Now()
			var visible []*model.Item
			for _, it := range items {
				if it.DeferUntil == nil || !it.DeferUntil.After(now) {
					visible = append(visible, it)
				}
			}
			// Sort by due_at ascending, nil due dates last
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
}

func newReflectProjectsCommand(c *container) *cobra.Command {
	var stalled bool

	cmd := &cobra.Command{
		Use:   "projects",
		Short: "List all active projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			kind := model.KindProject
			status := model.StatusActive
			projects, err := store.List(c.cfg, store.Filter{Kind: &kind, Status: &status})
			if err != nil {
				return err
			}

			// Build set of project IDs that have at least one linked active next action
			naKind := model.KindNextAction
			naStatus := model.StatusActive
			nextActions, err := store.List(c.cfg, store.Filter{Kind: &naKind, Status: &naStatus})
			if err != nil {
				return err
			}
			linkedProjects := make(map[string]int)
			for _, na := range nextActions {
				if na.Project != "" {
					linkedProjects[na.Project]++
				}
			}

			var result []*model.Item
			for _, proj := range projects {
				if stalled && linkedProjects[proj.ID] > 0 {
					continue
				}
				result = append(result, proj)
			}
			c.printer.PrintItems(result)
			return nil
		},
	}

	cmd.Flags().BoolVar(&stalled, "stalled", false, "Show only stalled projects")
	return cmd
}

func newReflectWaitingCommand(c *container) *cobra.Command {
	return &cobra.Command{
		Use:   "waiting",
		Short: "List all active waiting-for items",
		RunE: func(cmd *cobra.Command, args []string) error {
			kind := model.KindWaitingFor
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

func newReflectReviewCommand(c *container) *cobra.Command {
	return &cobra.Command{
		Use:   "review",
		Short: "List items due for review",
		RunE: func(cmd *cobra.Command, args []string) error {
			status := model.StatusActive
			items, err := store.List(c.cfg, store.Filter{Status: &status})
			if err != nil {
				return err
			}
			now := time.Now()
			todayEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
			var due []*model.Item
			for _, it := range items {
				if it.ReviewAt != nil && !it.ReviewAt.After(todayEnd) {
					due = append(due, it)
				}
			}
			sort.Slice(due, func(i, j int) bool {
				return due[i].ReviewAt.Before(*due[j].ReviewAt)
			})
			c.printer.PrintItems(due)
			return nil
		},
	}
}

func newReflectLogCommand(c *container) *cobra.Command {
	var (
		since    string
		until    string
		kindStr  string
		tags     []string
		statuses []string
	)

	cmd := &cobra.Command{
		Use:   "log",
		Short: "List recently resolved items (activity log)",
		RunE: func(cmd *cobra.Command, args []string) error {
			sinceTime, err := time.ParseInLocation("2006-01-02", since, time.Local)
			if err != nil {
				return fmt.Errorf("--since: cannot parse %q as date", since)
			}

			var untilEnd time.Time
			if until != "" {
				u, err := time.ParseInLocation("2006-01-02", until, time.Local)
				if err != nil {
					return fmt.Errorf("--until: cannot parse %q as date", until)
				}
				untilEnd = time.Date(u.Year(), u.Month(), u.Day(), 23, 59, 59, 0, u.Location())
			}

			statusSet, err := parseLogStatuses(statuses)
			if err != nil {
				return err
			}

			f := store.Filter{}
			if kindStr != "" {
				k := model.Kind(kindStr)
				if !isValidKind(k) {
					return fmt.Errorf("invalid kind %q", kindStr)
				}
				f.Kind = &k
			}
			if len(statusSet) == 1 {
				for s := range statusSet {
					f.Status = &s
				}
			}

			items, err := store.List(c.cfg, f)
			if err != nil {
				return err
			}

			var result []*model.Item
			for _, it := range items {
				if _, ok := statusSet[it.Status]; !ok {
					continue
				}
				if it.UpdatedAt.Before(sinceTime) {
					continue
				}
				if !untilEnd.IsZero() && it.UpdatedAt.After(untilEnd) {
					continue
				}
				if !matchAllTags(it, tags) {
					continue
				}
				result = append(result, it)
			}
			sort.Slice(result, func(i, j int) bool {
				return result[i].UpdatedAt.After(result[j].UpdatedAt)
			})
			c.printer.PrintLogItems(result)
			return nil
		},
	}

	cmd.Flags().StringVar(&since, "since", "", "Show items updated on or after this date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&until, "until", "", "Show items updated on or before this date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&kindStr, "kind", "", "Filter by kind")
	cmd.Flags().StringSliceVar(&tags, "tag", nil, "Filter by tag (repeatable)")
	cmd.Flags().StringSliceVar(&statuses, "status", nil, "Filter by terminal status (repeatable; default: done)")
	_ = cmd.MarkFlagRequired("since")
	return cmd
}

func newReflectTicklerCommand(c *container) *cobra.Command {
	var pull bool

	cmd := &cobra.Command{
		Use:   "tickler",
		Short: "List fired tickler items, or pull them into the inbox with --pull",
		RunE: func(cmd *cobra.Command, args []string) error {
			kind := model.KindTickler
			status := model.StatusActive
			items, err := store.List(c.cfg, store.Filter{Kind: &kind, Status: &status})
			if err != nil {
				return err
			}
			now := time.Now()
			todayEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())

			var due []*model.Item
			triggers := make(map[string]time.Time)
			for _, it := range items {
				trigger := ticklerTrigger(it)
				if trigger == nil || trigger.After(todayEnd) {
					continue
				}
				due = append(due, it)
				triggers[it.ID] = *trigger
			}
			sort.Slice(due, func(i, j int) bool {
				return triggers[due[i].ID].Before(triggers[due[j].ID])
			})

			if !pull {
				c.printer.PrintItems(due)
				return nil
			}

			pulled := make([]string, 0, len(due))
			for _, it := range due {
				src := store.PathForItem(c.cfg, it)
				full, body, err := store.Read(src)
				if err != nil {
					return err
				}
				full.Kind = model.KindInbox
				full.DeferUntil = nil
				full.UpdatedAt = time.Now()
				dst := store.PathForItem(c.cfg, full)
				if err := store.Move(src, dst, full, body); err != nil {
					return err
				}
				pulled = append(pulled, full.ID)
			}
			c.printer.PrintPulled(pulled)
			return nil
		},
	}

	cmd.Flags().BoolVar(&pull, "pull", false, "Move fired tickler items into the inbox")
	return cmd
}

func ticklerTrigger(it *model.Item) *time.Time {
	if it.DeferUntil != nil {
		return it.DeferUntil
	}
	if it.ReviewAt != nil {
		return it.ReviewAt
	}
	return nil
}

func parseLogStatuses(raw []string) (map[model.Status]struct{}, error) {
	if len(raw) == 0 {
		return map[model.Status]struct{}{model.StatusDone: {}}, nil
	}
	set := make(map[model.Status]struct{}, len(raw))
	for _, s := range raw {
		status := model.Status(s)
		if !model.IsTerminal(status) {
			return nil, fmt.Errorf("--status: %q is not a terminal status (done, canceled, discarded, archived)", s)
		}
		set[status] = struct{}{}
	}
	return set, nil
}
