package cmd

import (
	"encoding/json"
	"fmt"

	"historic/internal/lifecycle"

	"github.com/spf13/cobra"
)

func newSyncMetaCommand() *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "sync-meta <id|topic-path>",
		Short: "Synchronize topic Files and Assets metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspace, err := commandWorkspace()
			if err != nil {
				return writeCommandError(cmd, "sync-meta", jsonOutput, err)
			}
			change, err := lifecycle.SyncMeta(workspace, args[0])
			if err != nil {
				return writeCommandError(cmd, "sync-meta", jsonOutput, err)
			}
			response := syncMetaOutput{Command: "sync-meta", OK: true, Data: syncMetaData{
				ID: change.ID.String(), Path: change.Path, Files: change.Files, Assets: change.Assets, Updated: change.Updated,
			}, Error: nil}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(response)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s metadata synchronized: %d Files, %d Assets\n", change.ID, change.Files, change.Assets)
			return err
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	return command
}

type syncMetaOutput struct {
	Command string       `json:"command"`
	OK      bool         `json:"ok"`
	Data    syncMetaData `json:"data"`
	Error   any          `json:"error"`
}

type syncMetaData struct {
	ID      string `json:"id"`
	Path    string `json:"path"`
	Files   int    `json:"files"`
	Assets  int    `json:"assets"`
	Updated bool   `json:"updated"`
}
