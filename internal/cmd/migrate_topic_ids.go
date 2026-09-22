package cmd

import (
	"encoding/json"
	"fmt"

	"historic/internal/repository"

	"github.com/spf13/cobra"
)

func newMigrateTopicIDsCommand() *cobra.Command {
	var dryRun, jsonOutput bool
	var legacyID string
	command := &cobra.Command{
		Use:   "migrate-topic-ids",
		Short: "Migrate legacy five-digit topic IDs to modern 13-digit identities",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			workspace, err := commandWorkspace()
			if err != nil {
				return writeCommandError(cmd, "migrate-topic-ids", jsonOutput, err)
			}
			report, err := repository.NewTopicStore(workspace).MigrateTopicIDs(repository.TopicIDMigrationOptions{
				DryRun:   dryRun,
				LegacyID: legacyID,
			})
			if err != nil {
				return writeCommandError(cmd, "migrate-topic-ids", jsonOutput, err)
			}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(migrateTopicIDsOutput{Command: "migrate-topic-ids", OK: true, Data: report})
			}
			if len(report.Mappings) == 0 {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "No legacy topic IDs to migrate.")
				return err
			}
			for _, m := range report.Mappings {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s → %s  %s\n", m.OldID, m.NewID, m.Action); err != nil {
					return err
				}
			}
			if dryRun {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "Dry run: no files changed.")
			}
			return err
		},
	}
	command.Flags().BoolVar(&dryRun, "dry-run", false, "show migration plan without writing metadata")
	command.Flags().StringVar(&legacyID, "id", "", "migrate only one five-digit topic ID")
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	return command
}

type migrateTopicIDsOutput struct {
	Command string                            `json:"command"`
	OK      bool                              `json:"ok"`
	Data    repository.TopicIDMigrationReport `json:"data"`
	Error   any                               `json:"error"`
}
