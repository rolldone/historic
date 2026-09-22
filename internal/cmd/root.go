package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/lifecycle"

	"github.com/spf13/cobra"
)

const Version = config.VersionName

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
	root.AddCommand(newMigrateFileIDsCommand())
	root.AddCommand(newDoctorCommand())
	root.AddCommand(newUpgradeCommand())
	root.AddCommand(newSearchCommand())
	root.AddCommand(newImportCommand())
	root.AddCommand(newFileStatusCommand())
	root.AddCommand(newSaveCommand())
	root.AddCommand(newLogCommand())
	root.AddCommand(newDiffCommand())
	root.AddCommand(newRestoreCommand())
	root.AddCommand(newDeleteCommand())
	root.AddCommand(newDeleteTopicCommand())
	root.AddCommand(newPurgeCommand())
	root.AddCommand(newStorageCommand("close", func(service lifecycle.Service, id domain.ID) (lifecycle.Change, error) { return service.Close(id) }))
	root.AddCommand(newStorageCommand("open", func(service lifecycle.Service, id domain.ID) (lifecycle.Change, error) { return service.Open(id) }))
	return root
}

func newVersionCommand() *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "version",
		Short: "Print the Historic version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
					"version_name": config.VersionName, "version_code": config.VersionCode,
					"workspace_format_version": config.WorkspaceFormatVersion, "index_schema_version": config.IndexSchemaVersion,
				})
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "Historic %s (version code %d)\nworkspace format %d, index schema %d\n", config.VersionName, config.VersionCode, config.WorkspaceFormatVersion, config.IndexSchemaVersion)
			return err
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	return command
}

// ConfigureOutput makes command output deterministic for callers and tests.
func ConfigureOutput(root *cobra.Command, out, errOut io.Writer) {
	root.SetOut(out)
	root.SetErr(errOut)
}
