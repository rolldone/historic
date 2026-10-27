package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

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
	ID      string `json:"id"`
	Title   string `json:"title"`
	Path    string `json:"path"`
	Extends string `json:"extends"`
}

// extendsFlagValue rejects repeated or empty --extends values so the one-way
// parent relation cannot be ambiguous.
type extendsFlagValue struct {
	value string
	set   bool
}

func (value *extendsFlagValue) String() string { return value.value }

func (value *extendsFlagValue) Set(input string) error {
	if value.set {
		return fmt.Errorf("--extends may be provided only once")
	}
	if strings.TrimSpace(input) == "" {
		return fmt.Errorf("--extends requires a topic ID")
	}
	value.value = input
	value.set = true
	return nil
}

func (value *extendsFlagValue) Type() string { return "topic-id" }

func newCreateCommand() *cobra.Command {
	var requestedID string
	var jsonOutput bool
	extends := &extendsFlagValue{}
	command := &cobra.Command{
		Use:   "create <title>",
		Short: "Create a new topic",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspace, err := commandWorkspace()
			if err != nil {
				return writeCommandError(cmd, "create", jsonOutput, err)
			}
			topic, err := repository.NewTopicStore(workspace).CreateTopic(args[0], requestedID, extends.value)
			if err != nil {
				if jsonOutput {
					if outputErr := writeCreateJSON(cmd, createOutput{Command: "create", OK: false, Error: readinessErrorValue(err)}); outputErr != nil {
						return outputErr
					}
					return SilentError{Err: err}
				}
				return err
			}
			result := createdTopicOutput{ID: topic.ID.String(), Title: topic.Title, Path: workspace.RelativePath(topic.Path), Extends: topic.Extends.String()}
			if jsonOutput {
				return writeCreateJSON(cmd, createOutput{Command: "create", OK: true, Data: result, Error: nil})
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Created topic %s %s\n", result.ID, result.Path)
			return err
		},
	}
	command.Flags().StringVar(&requestedID, "id", "", "use an explicit five-digit topic ID")
	command.Flags().Var(extends, "extends", "parent topic ID to extend (open or closed)")
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	return command
}

func writeCreateJSON(command *cobra.Command, output createOutput) error {
	encoder := json.NewEncoder(command.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
