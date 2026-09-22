package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"historic/internal/config"
	"historic/internal/repository"

	"github.com/spf13/cobra"
)

type createOutput struct {
	Command string      `json:"command"`
	OK      bool        `json:"ok"`
	Data    interface{} `json:"data,omitempty"`
	Error   interface{} `json:"error"`
}

type createdTopicOutput struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Path  string `json:"path"`
}

func newCreateCommand() *cobra.Command {
	var requestedID string
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "create <title>",
		Short: "Create a new topic",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			current, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get current directory: %w", err)
			}
			root, err := config.DiscoverRoot(current)
			if err != nil {
				return err
			}
			workspace, err := config.Initialize(root)
			if err != nil {
				return err
			}
			topic, err := repository.NewTopicStore(workspace).CreateTopic(args[0], requestedID)
			if err != nil {
				if jsonOutput {
					if outputErr := writeCreateJSON(cmd, createOutput{Command: "create", OK: false, Error: err.Error()}); outputErr != nil {
						return outputErr
					}
				}
				return err
			}
			result := createdTopicOutput{ID: topic.ID.String(), Title: topic.Title, Path: workspace.RelativePath(topic.Path)}
			if jsonOutput {
				return writeCreateJSON(cmd, createOutput{Command: "create", OK: true, Data: result, Error: nil})
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Created topic %s %s\n", result.ID, result.Path)
			return err
		},
	}
	command.Flags().StringVar(&requestedID, "id", "", "use an explicit five-digit topic ID")
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	return command
}

func writeCreateJSON(command *cobra.Command, output createOutput) error {
	encoder := json.NewEncoder(command.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
