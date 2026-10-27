package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"historic/internal/domain"
	"historic/internal/lifecycle"
	"historic/internal/repository"

	"github.com/spf13/cobra"
)

type forwardOutput struct {
	Command string       `json:"command"`
	OK      bool         `json:"ok"`
	Data    *forwardData `json:"data,omitempty"`
	Error   any          `json:"error"`
}

type forwardData struct {
	Path         string `json:"path"`
	Previous     string `json:"previous"`
	Current      string `json:"current"`
	ForwardedTo  string `json:"forwarded_to"`
	CreatedTopic string `json:"created_topic"`
}

func newForwardCommand() *cobra.Command {
	var to string
	var createTitle string
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "forward <path> (--to <topic-id> | --create <title>)",
		Short: "Forward a file to a topic that extends the current one",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspace, err := commandWorkspace()
			if err != nil {
				return writeCommandError(cmd, "forward", jsonOutput, err)
			}
			hasTo := strings.TrimSpace(to) != ""
			hasCreate := strings.TrimSpace(createTitle) != ""
			if hasTo == hasCreate {
				return writeCommandError(cmd, "forward", jsonOutput, fmt.Errorf("%w: provide exactly one of --to or --create", domain.ErrConflict))
			}

			var (
				target       domain.ID
				createdTopic string
				createdPath  string
			)
			if hasTo {
				parsed, parseErr := domain.ParseTopicID(strings.TrimSpace(to))
				if parseErr != nil {
					return writeCommandError(cmd, "forward", jsonOutput, parseErr)
				}
				target = domain.ID(parsed)
			} else {
				sourceID, sourceErr := lifecycle.FileTopicID(workspace, args[0])
				if sourceErr != nil {
					return writeCommandError(cmd, "forward", jsonOutput, sourceErr)
				}
				topic, createErr := repository.NewTopicStore(workspace).CreateTopic(createTitle, "", sourceID.String())
				if createErr != nil {
					return writeCommandError(cmd, "forward", jsonOutput, createErr)
				}
				target = topic.ID
				createdTopic = topic.ID.String()
				createdPath = topic.Path
			}

			change, err := lifecycle.ForwardFile(workspace, args[0], target)
			if err != nil {
				if createdPath != "" {
					_ = os.RemoveAll(createdPath)
				}
				return writeCommandError(cmd, "forward", jsonOutput, err)
			}
			data := &forwardData{
				Path: change.Path, Previous: change.Previous.String(), Current: change.Current.String(),
				ForwardedTo: change.ForwardedTo.String(), CreatedTopic: createdTopic,
			}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(forwardOutput{Command: "forward", OK: true, Data: data, Error: nil})
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s: %s → %s (forwarded_to=%s)\n", change.Path, change.Previous, change.Current, change.ForwardedTo); err != nil {
				return err
			}
			if createdTopic != "" {
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "Created topic %s\n", createdTopic)
				return err
			}
			return nil
		},
	}
	command.Flags().StringVar(&to, "to", "", "existing topic ID that extends the source topic")
	command.Flags().StringVar(&createTitle, "create", "", "create a new topic that extends the source topic")
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	return command
}
