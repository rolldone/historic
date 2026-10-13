package cmd

import (
	"encoding/json"
	"fmt"

	"historic/internal/domain"
	"historic/internal/lifecycle"
	"historic/internal/tui"

	"github.com/spf13/cobra"
)

func newSaveCommand() *cobra.Command {
	var jsonOutput, all bool
	command := &cobra.Command{
		Use:   "save [id]",
		Short: "Mirror one or all open topics to closed storage",
		Long:  "Without an ID, interactively select one open topic. Use --all to mirror every open topic.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if all && len(args) > 0 {
				return writeCommandError(cmd, "save", jsonOutput, fmt.Errorf("--all cannot be combined with a topic ID"))
			}
			workspace, err := commandWorkspace()
			if err != nil {
				return writeCommandError(cmd, "save", jsonOutput, err)
			}
			service := lifecycle.NewService(workspace)
			var result lifecycle.MirrorAllResult
			var saveErr error
			if len(args) == 1 {
				result, saveErr = mirrorOne(service, args[0])
			} else if all {
				result, saveErr = service.MirrorAllOpen()
			} else if jsonOutput {
				return writeCommandError(cmd, "save", true, fmt.Errorf("topic ID or --all is required with --json"))
			} else {
				selected, pickErr := tui.PickTopic(workspace, tui.TopicPickerOptions{Input: cmd.InOrStdin(), Output: cmd.OutOrStdout()})
				if pickErr != nil {
					return pickErr
				}
				if selected == nil {
					return nil
				}
				result, saveErr = mirrorOne(service, selected.ID)
			}
			response := struct {
				Command string                    `json:"command"`
				OK      bool                      `json:"ok"`
				Data    lifecycle.MirrorAllResult `json:"data"`
				Error   any                       `json:"error"`
			}{Command: "save", OK: saveErr == nil, Data: result}
			if saveErr != nil {
				response.Error = saveErr.Error()
			}
			if jsonOutput {
				if err := json.NewEncoder(cmd.OutOrStdout()).Encode(response); err != nil {
					return err
				}
			} else if err := writeMirrorAllHuman(cmd, result); err != nil {
				return err
			}
			if saveErr != nil {
				return SilentError{Err: saveErr}
			}
			return nil
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	command.Flags().BoolVar(&all, "all", false, "mirror all open topics")
	return command
}

func mirrorOne(service lifecycle.Service, value string) (lifecycle.MirrorAllResult, error) {
	result := lifecycle.MirrorAllResult{Succeeded: make([]lifecycle.MirrorSuccess, 0), Failed: make([]lifecycle.MirrorFailure, 0), Total: 1}
	id, err := domain.ParseTopicIdentity(value)
	if err != nil {
		result.FailedCount = 1
		result.Failed = append(result.Failed, lifecycle.MirrorFailure{ID: value, Error: err.Error()})
		return result, err
	}
	change, err := service.MirrorTopic(id)
	if err != nil {
		result.FailedCount = 1
		result.Failed = append(result.Failed, lifecycle.MirrorFailure{ID: id.String(), Error: err.Error()})
		return result, err
	}
	result.Saved = 1
	result.Succeeded = append(result.Succeeded, lifecycle.MirrorSuccess{ID: change.ID.String(), Title: change.Title, Path: change.Path})
	return result, nil
}

func writeMirrorAllHuman(cmd *cobra.Command, result lifecycle.MirrorAllResult) error {
	for _, item := range result.Succeeded {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "saved %s %s: %s\n", item.ID, item.Title, item.Path); err != nil {
			return err
		}
	}
	for _, item := range result.Failed {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "failed %s: %s: %s\n", item.ID, item.Path, item.Error); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "save: total %d, saved %d, failed %d\n", result.Total, result.Saved, result.FailedCount)
	return err
}
