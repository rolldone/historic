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
		Use:   "sync-meta [id|topic-path]",
		Short: "Synchronize topic Files and Assets metadata",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspace, err := commandWorkspace()
			if err != nil {
				return writeCommandError(cmd, "sync-meta", jsonOutput, err)
			}
			if len(args) == 0 {
				batch := lifecycle.SyncMetaBatch(workspace)
				response := syncMetaBatchOutput{Command: "sync-meta", OK: batch.Errors == 0, Data: syncMetaBatchData{Topics: batch.Topics, Total: batch.Total, Updated: batch.Updated, Errors: batch.Errors}}
				if jsonOutput {
					if err := json.NewEncoder(cmd.OutOrStdout()).Encode(response); err != nil {
						return err
					}
				} else {
					printBatchOutput(cmd, batch)
				}
				if batch.Errors > 0 {
					batchError := fmt.Errorf("sync-meta batch completed with %d error(s)", batch.Errors)
					if jsonOutput {
						return SilentError{Err: batchError}
					}
					return batchError
				}
				return nil
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

type syncMetaBatchOutput struct {
	Command string            `json:"command"`
	OK      bool              `json:"ok"`
	Data    syncMetaBatchData `json:"data"`
	Error   any               `json:"error"`
}

type syncMetaBatchData struct {
	Topics  []lifecycle.BatchItem `json:"topics"`
	Total   int                   `json:"total"`
	Updated int                   `json:"updated"`
	Errors  int                   `json:"errors"`
}

func printBatchOutput(cmd *cobra.Command, batch lifecycle.BatchChange) {
	for _, topic := range batch.Topics {
		if topic.Error != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s: error: %s\n", topic.ID, topic.Path, topic.Error)
			continue
		}
		state := "unchanged"
		if topic.Updated {
			state = "updated"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s %s: %d Files, %d Assets, %s\n", topic.ID, topic.Path, topic.Files, topic.Assets, state)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Total: %d topic(s), %d updated, %d error(s)\n", batch.Total, batch.Updated, batch.Errors)
}
