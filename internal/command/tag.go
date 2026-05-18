package command

import (
	"sort"

	"github.com/spf13/cobra"

	"github.com/takai/htd/internal/output"
	"github.com/takai/htd/internal/store"
)

func newTagCommand(c *container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag",
		Short: "Tag-level operations across items",
	}
	cmd.AddCommand(newTagListCommand(c))
	return cmd
}

func newTagListCommand(c *container) *cobra.Command {
	var similar string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tags in use with item counts (use --similar to surface near-matches)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Filter{} pulls every item, active + archived, in one call.
			items, err := store.List(c.cfg, store.Filter{})
			if err != nil {
				return err
			}

			counts := make(map[string]int)
			for _, it := range items {
				for _, t := range it.Tags {
					counts[t]++
				}
			}

			useSimilar := cmd.Flags().Changed("similar")
			rows := make([]output.TagCount, 0, len(counts))
			if useSimilar {
				normInput := normalizeTag(similar)
				for tag, count := range counts {
					dist := levenshtein(tag, similar)
					if dist > 3 && (normInput == "" || normalizeTag(tag) != normInput) && tag != similar {
						continue
					}
					rows = append(rows, output.TagCount{Tag: tag, Count: count, Distance: dist})
				}
				sort.Slice(rows, func(i, j int) bool {
					if rows[i].Distance != rows[j].Distance {
						return rows[i].Distance < rows[j].Distance
					}
					if rows[i].Count != rows[j].Count {
						return rows[i].Count > rows[j].Count
					}
					return rows[i].Tag < rows[j].Tag
				})
			} else {
				for tag, count := range counts {
					rows = append(rows, output.TagCount{Tag: tag, Count: count})
				}
				sort.Slice(rows, func(i, j int) bool {
					if rows[i].Count != rows[j].Count {
						return rows[i].Count > rows[j].Count
					}
					return rows[i].Tag < rows[j].Tag
				})
			}

			c.printer.PrintTagCounts(rows, useSimilar)
			return nil
		},
	}

	cmd.Flags().StringVar(&similar, "similar", "",
		"Surface tags within Levenshtein distance 3 of this value, or whose normalized form (lowercase, [a-z0-9] only) collides with it")
	return cmd
}
