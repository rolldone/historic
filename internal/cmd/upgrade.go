package cmd

import (
	"encoding/json"
	"fmt"

	"historic/internal/indexer"

	"github.com/spf13/cobra"
)

func newUpgradeCommand() *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{Use: "upgrade", Short: "Upgrade the Historic index schema", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		workspace, err := commandWorkspace()
		if err != nil {
			return writeCommandError(cmd, "upgrade", jsonOutput, err)
		}
		count, path, err := indexer.Upgrade(workspace)
		if err != nil {
			return writeCommandError(cmd, "upgrade", jsonOutput, err)
		}
		response := upgradeOutput{Command: "upgrade", OK: true, Data: upgradeData{Records: count, Index: path, SchemaVersion: indexer.SchemaVersion}}
		if jsonOutput {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(response)
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Upgraded index to schema %d with %d records at %s\n", response.Data.SchemaVersion, count, path)
		return err
	}}
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	return command
}

type upgradeOutput struct {
	Command string      `json:"command"`
	OK      bool        `json:"ok"`
	Data    upgradeData `json:"data"`
	Error   any         `json:"error"`
}
type upgradeData struct {
	Records       int    `json:"records"`
	Index         string `json:"index"`
	SchemaVersion int    `json:"schema_version"`
}
