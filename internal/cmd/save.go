package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"historic/internal/gitproxy"

	"github.com/spf13/cobra"
)

func newSaveCommand() *cobra.Command {
	var message string
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "save",
		Short: "Commit workspace changes to the internal Git repository",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			workspace, err := commandWorkspace()
			if err != nil {
				return writeCommandError(cmd, "save", jsonOutput, err)
			}
			repository, err := gitproxy.Open(workspace.Database)
			if err != nil {
				return writeCommandError(cmd, "save", jsonOutput, err)
			}
			if err := repository.AddAll(); err != nil {
				return writeCommandError(cmd, "save", jsonOutput, err)
			}
			commitID, err := repository.Commit(message)
			if err != nil {
				if err == gitproxy.ErrNothingToCommit {
					response := saveOutput{Command: "save", OK: true, Data: saveData{Committed: false}, Error: nil}
					if jsonOutput {
						return json.NewEncoder(cmd.OutOrStdout()).Encode(response)
					}
					_, printErr := fmt.Fprintln(cmd.OutOrStdout(), "Nothing to save.")
					return printErr
				}
				return writeCommandError(cmd, "save", jsonOutput, err)
			}
			response := saveOutput{Command: "save", OK: true, Data: saveData{Committed: true, Commit: commitID, Message: strings.TrimSpace(message)}, Error: nil}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(response)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Saved %s\n", commitID)
			return err
		},
	}
	command.Flags().StringVarP(&message, "message", "m", "", "commit message")
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	return command
}

type saveOutput struct {
	Command string   `json:"command"`
	OK      bool     `json:"ok"`
	Data    saveData `json:"data"`
	Error   any      `json:"error"`
}

type saveData struct {
	Committed bool   `json:"committed"`
	Commit    string `json:"commit,omitempty"`
	Message   string `json:"message,omitempty"`
}
