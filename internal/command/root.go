package command

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/takai/htd/internal/config"
	"github.com/takai/htd/internal/output"
	"github.com/takai/htd/internal/store"
)

// container holds shared state for all subcommands.
type container struct {
	cfg     *config.Config
	printer *output.Printer
}

func NewRootCommand() *cobra.Command {
	var (
		jsonMode bool
		path     string
		c        container
	)

	root := &cobra.Command{
		Use:          "htd",
		Short:        "Headless task management",
		SilenceUsage: true,
	}

	root.PersistentFlags().BoolVar(&jsonMode, "json", false, "Output in JSON format")
	root.PersistentFlags().StringVar(&path, "path", ".", "htd root directory")

	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		c.cfg = config.New(path)
		c.printer = output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode)
		if isCompletionCommand(cmd) {
			return nil
		}
		return store.EnsureDirs(c.cfg)
	}

	root.AddCommand(newInitCommand(&c))
	root.AddCommand(newCaptureCommand(&c))
	root.AddCommand(newClarifyCommand(&c))
	root.AddCommand(newOrganizeCommand(&c))
	root.AddCommand(newReflectCommand(&c))
	root.AddCommand(newEngageCommand(&c))
	root.AddCommand(newItemCommand(&c))

	return root
}

// isCompletionCommand reports whether cmd belongs to the auto-generated
// `completion` subtree, for which we skip directory initialization so that
// users can generate shell scripts without side effects on --path.
func isCompletionCommand(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Name() == "completion" {
			return true
		}
	}
	return false
}

// Execute runs the root command and exits with the appropriate code.
func Execute() {
	root := NewRootCommand()
	root.SetErr(os.Stderr)

	if err := root.Execute(); err != nil {
		if store.IsNotFound(err) {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(output.ExitNotFound)
		}
		os.Exit(output.ExitError)
	}
}
