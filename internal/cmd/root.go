package cmd

import (
	"fmt"
	"io"

	"historic/internal/domain"
	"historic/internal/lifecycle"

	"github.com/spf13/cobra"
)

const Version = "0.2.0"

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
	root.AddCommand(newListCommand())
	root.AddCommand(newShowCommand())
	root.AddCommand(newFindCommand())
	root.AddCommand(newRebuildCommand())
	root.AddCommand(newDoctorCommand())
	root.AddCommand(newUpgradeCommand())
	root.AddCommand(newSearchCommand())
	root.AddCommand(newImportCommand())
	root.AddCommand(newFileStatusCommand())
	root.AddCommand(newSyncMetaCommand())
	root.AddCommand(newSaveCommand())
	root.AddCommand(newLogCommand())
	root.AddCommand(newDiffCommand())
	root.AddCommand(newRestoreCommand())
	root.AddCommand(newDeleteCommand())
	root.AddCommand(newDeleteTopicCommand())
	root.AddCommand(newPurgeCommand())
	root.AddCommand(newStorageCommand("close", func(service lifecycle.Service, id domain.ID) (lifecycle.Change, error) { return service.Close(id) }))
	root.AddCommand(newStorageCommand("open", func(service lifecycle.Service, id domain.ID) (lifecycle.Change, error) { return service.Open(id) }))
	for _, status := range []domain.Status{domain.StatusDraft, domain.StatusProgress, domain.StatusPending, domain.StatusReview, domain.StatusBlocked, domain.StatusComplete, domain.StatusFailed, domain.StatusCancelled, domain.StatusArchived} {
		root.AddCommand(newLifecycleCommand(status))
	}
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
