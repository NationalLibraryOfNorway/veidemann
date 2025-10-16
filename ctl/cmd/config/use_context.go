package config

import (
	"fmt"

	"github.com/NationalLibraryOfNorway/veidemann/ctl/config"
	"github.com/spf13/cobra"
)

// newUseContextCmd returns the use-context subcommand.
func newUseContextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use-context CONTEXT",
		Short: "Set the current context",
		Long: `Set the current context

Examples:
  # Use the context for the prod cluster
  veidemannctl config use-context prod
`,
		Aliases: []string{"use"},
		Args:    cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			c, err := config.ListContexts()
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			return c, cobra.ShellCompDirectiveNoSpace
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// silence usage to avoid printing usage when returning an error
			cmd.SilenceUsage = true

			name := args[0]
			ok, err := config.ContextExists(name)
			if err != nil {
				return fmt.Errorf("failed switching context to '%s': %w", name, err)
			}
			if !ok {
				return fmt.Errorf("non existing context '%s'", name)
			}
			if err := config.SetCurrentContext(name); err != nil {
				return fmt.Errorf("failed switching context to '%s': %w", name, err)
			}
			fmt.Printf("Switched to context \"%v\"\n", name)
			return nil
		},
	}
}
