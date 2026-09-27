package cmd

import (
	"encoding/json"
	"fmt"

	"historic/internal/domain"
	"historic/internal/lifecycle"

	"github.com/spf13/cobra"
)

func newCloseCommand() *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "close <id|all>",
		Short: "close a topic or all open topics",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] != "all" {
				return runStorageCommand(cmd, "close", args[0], jsonOutput, func(service lifecycle.Service, id domain.ID) (lifecycle.Change, error) {
					return service.Close(id)
				})
			}

			workspace, err := commandWorkspace()
			if err != nil {
				return writeCommandError(cmd, "close", jsonOutput, err)
			}
			result, closeErr := lifecycle.NewService(workspace).CloseAll()
			response := struct {
				Command string                   `json:"command"`
				OK      bool                     `json:"ok"`
				Data    lifecycle.CloseAllResult `json:"data"`
				Error   any                      `json:"error"`
			}{Command: "close", OK: closeErr == nil, Data: result}
			if closeErr != nil {
				response.Error = closeErr.Error()
			}
			if jsonOutput {
				if encodeErr := json.NewEncoder(cmd.OutOrStdout()).Encode(response); encodeErr != nil {
					return encodeErr
				}
			} else {
				if err := writeCloseAllHuman(cmd, result); err != nil {
					return err
				}
			}
			if closeErr != nil {
				return SilentError{Err: closeErr}
			}
			return nil
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	return command
}

func runStorageCommand(cmd *cobra.Command, name, value string, jsonOutput bool, move func(lifecycle.Service, domain.ID) (lifecycle.Change, error)) error {
	workspace, err := commandWorkspace()
	if err != nil {
		return writeCommandError(cmd, name, jsonOutput, err)
	}
	id, err := domain.ParseTopicIdentity(value)
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
}

func writeCloseAllHuman(cmd *cobra.Command, result lifecycle.CloseAllResult) error {
	for _, item := range result.Succeeded {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "closed %s %s: %s\n", item.ID, item.Title, item.Path); err != nil {
			return err
		}
	}
	for _, item := range result.Failed {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "failed %s: %s: %s\n", item.ID, item.Path, item.Error); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "close all: total %d, closed %d, failed %d\n", result.Total, result.Closed, result.FailedCount)
	return err
}
