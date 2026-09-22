package cmd

import (
	"encoding/json"
	"fmt"

	"historic/internal/domain"
	"historic/internal/lifecycle"

	"github.com/spf13/cobra"
)

func newStorageCommand(name string, move func(lifecycle.Service, domain.ID) (lifecycle.Change, error)) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: name + " <id>", Args: cobra.ExactArgs(1), Short: name + " a topic",
		RunE: func(cmd *cobra.Command, args []string) error {
			workspace, err := commandWorkspace()
			if err != nil {
				return writeCommandError(cmd, name, jsonOutput, err)
			}
			id, err := domain.ParseTopicIdentity(args[0])
			if err != nil {
				return writeCommandError(cmd, name, jsonOutput, err)
			}
			change, err := move(lifecycle.NewService(workspace), id)
			if err != nil {
				return writeCommandError(cmd, name, jsonOutput, err)
			}
			data := lifecycleData{ID: change.ID.String(), Title: change.Title, Previous: change.Previous.String(), Current: change.Current.String(), Path: change.Path, Storage: change.Storage.String(), PreviousStorage: change.PreviousStorage.String(), Archived: change.Archived}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(lifecycleOutput{Command: name, OK: true, Data: data})
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s %s: [%s] → [%s]\n", change.ID, change.Title, change.PreviousStorage, change.Storage)
			return err
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	return command
}
