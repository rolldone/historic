package cmd

import (
	"encoding/json"
	"fmt"

	"historic/internal/config"
	"historic/internal/lifecycle"

	"github.com/spf13/cobra"
)

func newRebuildCommand() *cobra.Command {
	var jsonOutput, noBump bool
	command := &cobra.Command{
		Use:   "rebuild",
		Short: "Rebuild the SQLite index from Markdown",
		Long:  "Reconcile every topic manifest and rebuild the SQLite index from Markdown. Use --no-bump to skip advancing the frontmatter updated date of content-changed files.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			workspace, err := commandWorkspaceForRecovery()
			if err != nil {
				return writeCommandError(cmd, "rebuild", jsonOutput, err)
			}
			if err := config.ValidateRebuildWorkspace(workspace); err != nil {
				return writeCommandError(cmd, "rebuild", jsonOutput, err)
			}
			var bump lifecycle.BumpResult
			if !noBump {
				bump = lifecycle.BumpAllChangedFiles(workspace)
			}
			change, err := lifecycle.RebuildMetadata(workspace)
			if err != nil {
				return writeCommandError(cmd, "rebuild", jsonOutput, fmt.Errorf("unified rebuild failed; run historic doctor for diagnosis: %w", err))
			}
			commit := lifecycle.AutoCommitRecovery(workspace, "rebuild")
			response := rebuildOutput{Command: "rebuild", OK: true, Data: rebuildData{Topics: change.Topics, Updated: change.Updated, Errors: change.Errors, Records: change.Records, Index: workspace.RelativePath(workspace.Index), BumpedFiles: bump.Count(), Commit: &commit}, Error: nil}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(response)
			}
			if _, err = fmt.Fprintf(cmd.OutOrStdout(), "Rebuilt index with %d records at %s\n", response.Data.Records, response.Data.Index); err != nil {
				return err
			}
			if err = writeBumpHuman(cmd, bump, noBump); err != nil {
				return err
			}
			return writeAutoCommitHuman(cmd, &commit)
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	command.Flags().BoolVar(&noBump, "no-bump", false, "skip advancing the frontmatter updated date of content-changed files")
	return command
}

type rebuildOutput struct {
	Command string      `json:"command"`
	OK      bool        `json:"ok"`
	Data    rebuildData `json:"data"`
	Error   any         `json:"error"`
}

type rebuildData struct {
	Topics      int                         `json:"topics"`
	Updated     int                         `json:"updated"`
	Errors      int                         `json:"errors"`
	Records     int                         `json:"records"`
	Index       string                      `json:"index"`
	BumpedFiles int                         `json:"bumped_files"`
	Commit      *lifecycle.AutoCommitResult `json:"commit"`
}
