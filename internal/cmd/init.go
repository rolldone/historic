package cmd

import (
	"fmt"
	"os"

	"historic/internal/config"

	"github.com/spf13/cobra"
)

func newInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize a Historic workspace",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			current, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get current directory: %w", err)
			}
			root, err := config.DiscoverRoot(current)
			if err != nil {
				return err
			}
			workspace, err := config.Initialize(root)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Initialized Historic in %s\n", workspace.Root)
			return err
		},
	}
}
