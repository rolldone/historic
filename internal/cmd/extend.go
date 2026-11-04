package cmd

import (
	"encoding/json"
	"fmt"

	"historic/internal/domain"
	"historic/internal/lifecycle"
	"historic/internal/repository"

	"github.com/spf13/cobra"
)

type extendOutput struct {
	Command string      `json:"command"`
	OK      bool        `json:"ok"`
	Data    interface{} `json:"data,omitempty"`
	Error   interface{} `json:"error"`
}

type extendedTopicOutput struct {
	ID      string `json:"id"`
	Extends string `json:"extends"`
	Path    string `json:"path"`
}

func newExtendCommand() *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "extend <topic-id> <parent-id>",
		Short: "Add an immutable extends relation to an existing topic",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspace, err := commandWorkspace()
			if err != nil {
				return writeCommandError(cmd, "extend", jsonOutput, err)
			}
			topicID, parseErr := domain.ParseTopicID(args[0])
			if parseErr != nil {
				return writeCommandError(cmd, "extend", jsonOutput, parseErr)
			}
			result, err := repository.NewTopicStore(workspace).ExtendTopic(topicID, args[1])
			if err != nil {
				return writeCommandError(cmd, "extend", jsonOutput, err)
			}
			var commit *lifecycle.AutoCommitResult
			if result.Storage == domain.StorageClosed {
				autoResult := lifecycle.AutoCommitTopic(workspace, "extend", domain.ID(topicID))
				commit = &autoResult
			}
			data := extendedTopicOutput{ID: result.ID.String(), Extends: result.Extends.String(), Path: result.Path}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(extendOutput{Command: "extend", OK: true, Data: data, Error: nil})
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Extended topic %s extends %s (%s)\n", data.ID, data.Extends, data.Path); err != nil {
				return err
			}
			return writeAutoCommitHuman(cmd, commit)
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	return command
}
