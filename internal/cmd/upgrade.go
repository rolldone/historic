package cmd

import (
	"encoding/json"
	"fmt"

	"historic/internal/repository"

	"github.com/spf13/cobra"
)

func newUpgradeCommand() *cobra.Command {
	var jsonOutput, dryRun, noFileIDs, noTopicIDs bool
	var backupDir string
	command := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade a legacy Historic workspace",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			workspace, err := commandWorkspace()
			if err != nil {
				return writeCommandError(cmd, "upgrade", jsonOutput, err)
			}
			report, err := repository.UpgradeWorkspace(workspace, repository.UpgradeOptions{DryRun: dryRun, BackupDir: backupDir, NoFileIDs: noFileIDs, NoTopicIDs: noTopicIDs})
			if err != nil {
				return writeCommandError(cmd, "upgrade", jsonOutput, err)
			}
			response := upgradeOutput{Command: "upgrade", OK: true, Data: report, Error: nil}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(response)
			}
			if report.DryRun {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Upgrade dry-run: topics=%d file_ids=%d topic_ids=%d\n", report.TopicsScanned, report.FileIDsMigrated, report.TopicIDsMigrated)
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Workspace upgraded: topics=%d file_ids=%d topic_ids=%d index_rebuilt=%t\n", report.TopicsScanned, report.FileIDsMigrated, report.TopicIDsMigrated, report.IndexRebuilt)
			return err
		},
	}
	command.Flags().BoolVar(&dryRun, "dry-run", false, "scan and show the plan without writing")
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	command.Flags().StringVar(&backupDir, "backup-dir", "", "immutable backup directory outside .historic")
	command.Flags().BoolVar(&noFileIDs, "no-file-ids", false, "admin/recovery: skip FileID migration")
	command.Flags().BoolVar(&noTopicIDs, "no-topic-ids", false, "admin/recovery: skip TopicID migration")
	return command
}

type upgradeOutput struct {
	Command string                   `json:"command"`
	OK      bool                     `json:"ok"`
	Data    repository.UpgradeReport `json:"data"`
	Error   any                      `json:"error"`
}
