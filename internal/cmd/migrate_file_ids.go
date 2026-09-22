package cmd

import (
	"encoding/json"
	"fmt"

	"historic/internal/domain"
	"historic/internal/lifecycle"

	"github.com/spf13/cobra"
)

func newMigrateFileIDsCommand() *cobra.Command {
	var dryRun, openOnly, closedOnly, jsonOutput bool
	var topicID string
	command := &cobra.Command{
		Use:   "migrate-file-ids",
		Short: "Migrate legacy managed-file IDs to UUIDv7",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if openOnly && closedOnly {
				return writeCommandError(cmd, "migrate-file-ids", jsonOutput, fmt.Errorf("%w: --open and --closed cannot be combined", domain.ErrConflict))
			}
			if cmd.Flags().Changed("force") {
				return writeCommandError(cmd, "migrate-file-ids", jsonOutput, fmt.Errorf("%w: --force is not supported", domain.ErrConflict))
			}
			workspace, err := commandWorkspace()
			if err != nil {
				return writeCommandError(cmd, "migrate-file-ids", jsonOutput, err)
			}
			var parsedID domain.ID
			var idPointer *domain.ID
			if topicID != "" {
				parsedID, err = domain.ParseID(topicID)
				if err != nil {
					return writeCommandError(cmd, "migrate-file-ids", jsonOutput, err)
				}
				idPointer = &parsedID
			}
			var storagePointer *domain.StorageState
			if openOnly {
				storage := domain.StorageOpen
				storagePointer = &storage
			}
			if closedOnly {
				storage := domain.StorageClosed
				storagePointer = &storage
			}
			report, err := lifecycle.NewService(workspace).MigrateFileIDs(lifecycle.FileIDMigrationOptions{DryRun: dryRun, TopicID: idPointer, Storage: storagePointer})
			if err != nil {
				return writeCommandError(cmd, "migrate-file-ids", jsonOutput, err)
			}
			response := migrateFileIDsOutput{Command: "migrate-file-ids", OK: true, Data: report, Error: nil}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(response)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Migrated file IDs: topics=%d manifests=%d migrated=%d preserved=%d skipped=%d\n", report.TopicsScanned, report.ManifestsUpdated, report.FilesMigrated, report.FilesPreserved, report.FilesSkipped)
			return err
		},
	}
	command.Flags().BoolVar(&dryRun, "dry-run", false, "show migration plan without writing metadata")
	command.Flags().BoolVar(&openOnly, "open", false, "target open topics only")
	command.Flags().BoolVar(&closedOnly, "closed", false, "target closed topics only")
	command.Flags().StringVar(&topicID, "id", "", "target one five-digit topic ID")
	command.Flags().Bool("force", false, "unsupported; migration is atomic and idempotent")
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	return command
}

type migrateFileIDsOutput struct {
	Command string                          `json:"command"`
	OK      bool                            `json:"ok"`
	Data    lifecycle.FileIDMigrationReport `json:"data"`
	Error   any                             `json:"error"`
}
