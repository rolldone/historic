package cmd

import (
	"fmt"

	"historic/internal/tui"

	"github.com/spf13/cobra"
)

func newSearchCommand() *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "search",
		Short: "Open the interactive read-only search TUI",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if jsonOutput {
				return fmt.Errorf("historic search does not support --json; use historic find --json")
			}
			workspace, err := commandWorkspace()
			if err != nil {
				return writeCommandError(cmd, "search", jsonOutput, err)
			}
			return tui.Run(workspace, tui.Options{Input: cmd.InOrStdin(), Output: cmd.OutOrStdout()})
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "reject JSON mode; use historic find --json")
	return command
}
