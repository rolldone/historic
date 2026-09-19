package cmd

import (
	"encoding/json"
	"fmt"

	"historic/internal/domain"
	"historic/internal/lifecycle"

	"github.com/spf13/cobra"
)

func newFileStatusCommand() *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "status <path> <status>",
		Short: "Set the status of one Markdown file",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspace, err := commandWorkspace()
			if err != nil {
				return writeCommandError(cmd, "status", jsonOutput, err)
			}
			next, err := domain.ParseStatus(args[1])
			if err != nil {
				return writeCommandError(cmd, "status", jsonOutput, err)
			}
			change, err := lifecycle.ChangeFileStatus(workspace, args[0], next)
			if err != nil {
				return writeCommandError(cmd, "status", jsonOutput, err)
			}
			response := fileStatusOutput{Command: "status", OK: true, Data: fileStatusData{
				ID: change.ID.String(), Title: change.Title, Previous: change.Previous.String(),
				Current: change.Current.String(), Path: change.Path, Updated: change.Updated,
			}, Error: nil}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(response)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s %s: %s → %s (%s)\n", change.ID, change.Path, change.Previous, change.Current, change.Updated)
			return err
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	return command
}

type fileStatusOutput struct {
	Command string         `json:"command"`
	OK      bool           `json:"ok"`
	Data    fileStatusData `json:"data"`
	Error   any            `json:"error"`
}

type fileStatusData struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Previous string `json:"previous"`
	Current  string `json:"current"`
	Path     string `json:"path"`
	Updated  string `json:"updated"`
}
