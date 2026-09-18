package cmd

import (
	"encoding/json"
	"fmt"

	"historic/internal/domain"
	"historic/internal/lifecycle"

	"github.com/spf13/cobra"
)

func newLifecycleCommand(status domain.Status) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use:   string(status) + " <id>",
		Short: "Set topic status to " + string(status),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspace, err := commandWorkspace()
			if err != nil {
				return writeCommandError(cmd, string(status), jsonOutput, err)
			}
			id, err := domain.ParseID(args[0])
			if err != nil {
				return writeCommandError(cmd, string(status), jsonOutput, err)
			}
			change, err := lifecycle.NewService(workspace).ChangeStatus(id, status)
			if err != nil {
				return writeCommandError(cmd, string(status), jsonOutput, err)
			}
			response := lifecycleOutput{Command: string(status), OK: true, Data: lifecycleData{
				ID: change.ID.String(), Title: change.Title, Previous: change.Previous.String(), Current: change.Current.String(), Path: change.Path, Archived: change.Archived,
			}, Error: nil}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(response)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s %s: %s → %s\n", change.ID, change.Title, change.Previous, change.Current)
			return err
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	return command
}

type lifecycleOutput struct {
	Command string        `json:"command"`
	OK      bool          `json:"ok"`
	Data    lifecycleData `json:"data"`
	Error   any           `json:"error"`
}

type lifecycleData struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Previous string `json:"previous"`
	Current  string `json:"current"`
	Path     string `json:"path"`
	Archived bool   `json:"archived"`
}
