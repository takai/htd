package command

import "github.com/spf13/cobra"

func newInitCommand(c *container) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create the htd directory layout",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c.printer.PrintPaths(c.cfg.AllDirs())
			return nil
		},
	}
}
