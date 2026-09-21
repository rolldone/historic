package cmd

import (
	"encoding/json"
	"fmt"

	"historic/internal/lifecycle"

	"github.com/spf13/cobra"
)

func newRebuildCommand() *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "rebuild",
		Short: "Rebuild the SQLite index from Markdown",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			workspace, err := commandWorkspace()
			if err != nil {
				return writeCommandError(cmd, "rebuild", jsonOutput, err)
			}
			change, err := lifecycle.RebuildMetadata(workspace)
			if err != nil {
				return writeCommandError(cmd, "rebuild", jsonOutput, fmt.Errorf("unified rebuild failed; run historic doctor for diagnosis: %w", err))
			}
			response := rebuildOutput{Command: "rebuild", OK: true, Data: rebuildData{Topics: change.Topics, Updated: change.Updated, Errors: change.Errors, Records: change.Records, Index: workspace.RelativePath(workspace.Index)}, Error: nil}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(response)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Rebuilt index with %d records at %s\n", response.Data.Records, response.Data.Index)
			return err
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	return command
}

type rebuildOutput struct {
	Command string      `json:"command"`
	OK      bool        `json:"ok"`
	Data    rebuildData `json:"data"`
	Error   any         `json:"error"`
}

type rebuildData struct {
	Topics  int    `json:"topics"`
	Updated int    `json:"updated"`
	Errors  int    `json:"errors"`
	Records int    `json:"records"`
	Index   string `json:"index"`
}
