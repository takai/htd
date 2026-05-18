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
		newReflectProjectCommand(c),
		newReflectWaitingCommand(c),
		newReflectReviewCommand(c),
		newReflectLogCommand(c),
		newReflectTicklerCommand(c),
	)
	return cmd
}

func newReflectProjectCommand(c *container) *cobra.Command {
	const archiveDefaultDays = 30
	var since string

	cmd := &cobra.Command{
		Use:   "project ID",
		Short: "Show a project with its active and recently-archived children",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]

			path, err := store.FindItem(c.cfg, projectID)
			if err != nil {
				return err
			}
			project, body, err := store.Read(path)
			if err != nil {
				return err
			}
			if project.Kind != model.KindProject {
				return &store.NotFoundError{Kind: store.EntityItem, ID: projectID}
			}

			cutoff, err := resolveSinceCutoff(cmd, since, archiveDefaultDays)
			if err != nil {
				return err
			}

			active := model.StatusActive
			naKind := model.KindNextAction
			wfKind := model.KindWaitingFor
			tkKind := model.KindTickler

			nextActions, err := store.List(c.cfg, store.Filter{Kind: &naKind, Status: &active, ProjectID: projectID})
			if err != nil {
				return err
			}
			sort.Slice(nextActions, func(i, j int) bool {
				return nextActions[i].UpdatedAt.After(nextActions[j].UpdatedAt)
			})

			waitingFor, err := store.List(c.cfg, store.Filter{Kind: &wfKind, Status: &active, ProjectID: projectID})
			if err != nil {
				return err
			}
			sort.Slice(waitingFor, func(i, j int) bool {
				return waitingFor[i].CreatedAt.Before(waitingFor[j].CreatedAt)
			})

			ticklers, err := store.List(c.cfg, store.Filter{Kind: &tkKind, Status: &active, ProjectID: projectID})
			if err != nil {
				return err
			}
			sort.Slice(ticklers, func(i, j int) bool {
				if ticklers[i].DeferUntil == nil {
					return false
				}
				if ticklers[j].DeferUntil == nil {
					return true
				}
				return ticklers[i].DeferUntil.Before(*ticklers[j].DeferUntil)
			})

			linked, err := store.List(c.cfg, store.Filter{ProjectID: projectID})
			if err != nil {
				return err
			}
			var archived []*model.Item
			for _, it := range linked {
				if !model.IsTerminal(it.Status) {
					continue
				}
				if !cutoff.IsZero() && it.UpdatedAt.Before(cutoff) {
					continue
				}
				archived = append(archived, it)
			}
			sort.Slice(archived, func(i, j int) bool {
				return archived[i].UpdatedAt.After(archived[j].UpdatedAt)
			})

			c.printer.PrintProjectView(project, body, nextActions, waitingFor, ticklers, archived, cutoff)
			return nil
		},
	}

	cmd.Flags().StringVar(&since, "since", "",
		"Show archived children updated on or after this date (YYYY-MM-DD); default is 30 days ago, pass --since '' to show all")
	return cmd
}

// resolveSinceCutoff returns the lower-bound time implied by a --since flag
// across commands that share the same three-state semantics (reflect project,
// reflect log). The flag behaves as:
//   - flag absent → default cutoff (now minus defaultDays)
//   - --since YYYY-MM-DD → cutoff at that date, local midnight
//   - --since '' (explicitly empty) → zero time, meaning "no cutoff"
func resolveSinceCutoff(cmd *cobra.Command, since string, defaultDays int) (time.Time, error) {
	flag := cmd.Flags().Lookup("since")
	if flag == nil || !flag.Changed {
		return time.Now().AddDate(0, 0, -defaultDays), nil
	}
	if since == "" {
		return time.Time{}, nil
	}
	t, err := time.ParseInLocation("2006-01-02", since, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("--since: cannot parse %q as date", since)
	}
	return t, nil
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
	const logDefaultDays = 30
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
			sinceTime, err := resolveSinceCutoff(cmd, since, logDefaultDays)
			if err != nil {
				return err
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

	cmd.Flags().StringVar(&since, "since", "",
		"Show items updated on or after this date (YYYY-MM-DD); default is 30 days ago, pass --since '' to show all")
	cmd.Flags().StringVar(&until, "until", "", "Show items updated on or before this date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&kindStr, "kind", "", "Filter by kind")
	cmd.Flags().StringSliceVar(&tags, "tag", nil, "Filter by tag (repeatable)")
	cmd.Flags().StringSliceVar(&statuses, "status", nil, "Filter by terminal status (repeatable; default: done)")
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
