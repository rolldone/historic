package cmd

import (
	"encoding/json"
	"fmt"

	"historic/internal/domain"
	"historic/internal/lifecycle"

	"github.com/spf13/cobra"
)

func newRestoreCommand() *cobra.Command {
	var snapshot string
	var force, jsonOutput bool
	command := &cobra.Command{
		Use:   "restore <id>",
		Short: "Restore a topic from an internal Git snapshot",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspace, err := commandWorkspace()
			if err != nil {
				return writeCommandError(cmd, "restore", jsonOutput, err)
			}
			id, err := domain.ParseTopicIdentity(args[0])
			if err != nil {
				return writeCommandError(cmd, "restore", jsonOutput, err)
			}
			if err := lifecycle.Restore(workspace, id, snapshot, force); err != nil {
				return writeCommandError(cmd, "restore", jsonOutput, err)
			}
			response := restoreOutput{Command: "restore", OK: true, Data: restoreData{ID: id.String(), Snapshot: snapshot}, Error: nil}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(response)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Restored %s from %s\n", id, snapshot)
			return err
		},
	}
	command.Flags().StringVar(&snapshot, "snapshot", "", "Git snapshot/commit ID")
	command.Flags().BoolVar(&force, "force", false, "replace an existing active topic")
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	_ = command.MarkFlagRequired("snapshot")
	return command
}

type restoreOutput struct {
	Command string      `json:"command"`
	OK      bool        `json:"ok"`
	Data    restoreData `json:"data"`
	Error   any         `json:"error"`
}

type restoreData struct {
	ID       string `json:"id"`
	Snapshot string `json:"snapshot"`
}
