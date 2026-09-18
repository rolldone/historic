package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

const Version = "0.1.0"

// NewRootCommand creates the root command for the Historic CLI.
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "historic",
		Short:         "A local history for every kind of work",
		Long:          "Historic stores work history as portable Markdown files.",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	root.AddCommand(newVersionCommand())
	root.AddCommand(newInitCommand())
	root.AddCommand(newCreateCommand())
	root.AddCommand(newAddCommand())
	return root
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the Historic version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), Version)
			return err
		},
	}
}

// ConfigureOutput makes command output deterministic for callers and tests.
func ConfigureOutput(root *cobra.Command, out, errOut io.Writer) {
	root.SetOut(out)
	root.SetErr(errOut)
}
