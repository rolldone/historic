package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/repository"

	"github.com/spf13/cobra"
)

func newListCommand() *cobra.Command {
	var closedOnly, jsonOutput bool
	command := &cobra.Command{
		Use:   "list",
		Short: "List Historic topics",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			workspace, err := commandWorkspace()
			if err != nil {
				return err
			}
			views, err := repository.NewTopicStore(workspace).ListTopics(closedOnly)
			if err != nil {
				return writeCommandError(cmd, "list", jsonOutput, err)
			}
			result := topicListOutput{Command: "list", OK: true, Data: views, Error: nil}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
			}
			if len(views) == 0 {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "No topics found.")
				return err
			}
			for _, view := range views {
				if _, err = fmt.Fprintf(cmd.OutOrStdout(), "%s  %-6s  %s  %s\n", view.ID, view.Storage, view.Title, view.Path); err != nil {
					return err
				}
			}
			return nil
		},
	}
	command.Flags().BoolVar(&closedOnly, "closed", false, "include closed topics")
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	return command
}

func newShowCommand() *cobra.Command {
	var includeClosed, jsonOutput bool
	command := &cobra.Command{
		Use:   "show <id>",
		Short: "Show a Historic topic",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspace, err := commandWorkspace()
			if err != nil {
				return err
			}
			id, err := domain.ParseTopicIdentity(args[0])
			if err != nil {
				return writeCommandError(cmd, "show", jsonOutput, err)
			}
			view, err := repository.NewTopicStore(workspace).ShowTopic(id, includeClosed)
			if err != nil {
				return writeCommandError(cmd, "show", jsonOutput, err)
			}
			result := topicShowOutput{Command: "show", OK: true, Data: view, Error: nil}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s  %s [%s]\nPath: %s\n", view.ID, view.Title, view.Storage, view.Path)
			if err != nil {
				return err
			}
			for _, file := range view.Files {
				if _, err = fmt.Fprintf(cmd.OutOrStdout(), "- %s (%s)\n", file.Path, file.Status); err != nil {
					return err
				}
			}
			return nil
		},
	}
	command.Flags().BoolVar(&includeClosed, "closed", false, "show a closed topic")
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	return command
}

type topicListOutput struct {
	Command string                 `json:"command"`
	OK      bool                   `json:"ok"`
	Data    []repository.TopicView `json:"data"`
	Error   any                    `json:"error"`
}

type topicShowOutput struct {
	Command string               `json:"command"`
	OK      bool                 `json:"ok"`
	Data    repository.TopicView `json:"data"`
	Error   any                  `json:"error"`
}

func commandWorkspace() (config.Workspace, error) {
	current, err := os.Getwd()
	if err != nil {
		return config.Workspace{}, fmt.Errorf("get current directory: %w", err)
	}
	root, err := config.DiscoverRoot(current)
	if err != nil {
		return config.Workspace{}, err
	}
	return config.Initialize(root)
}

func writeCommandError(cmd *cobra.Command, name string, jsonOutput bool, err error) error {
	if !jsonOutput {
		return err
	}
	response := struct {
		Command string `json:"command"`
		OK      bool   `json:"ok"`
		Data    any    `json:"data"`
		Error   string `json:"error"`
	}{Command: name, OK: false, Data: nil, Error: strings.TrimSpace(err.Error())}
	if encodeErr := json.NewEncoder(cmd.OutOrStdout()).Encode(response); encodeErr != nil {
		return encodeErr
	}
	return SilentError{Err: err}
}
