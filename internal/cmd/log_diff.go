package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"historic/internal/domain"
	"historic/internal/gitproxy"

	"github.com/spf13/cobra"
)

func newLogCommand() *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "log <id>",
		Short: "Show internal Git history for a topic",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspace, err := commandWorkspace()
			if err != nil {
				return writeCommandError(cmd, "log", jsonOutput, err)
			}
			id, err := domain.ParseTopicIdentity(args[0])
			if err != nil {
				return writeCommandError(cmd, "log", jsonOutput, err)
			}
			repository, err := gitproxy.Open(workspace.Database)
			if err != nil {
				return writeCommandError(cmd, "log", jsonOutput, err)
			}
			text, err := repository.Log(id.String() + "-*")
			if err != nil {
				return writeCommandError(cmd, "log", jsonOutput, err)
			}
			entries := parseGitLog(text)
			response := logOutput{Command: "log", OK: true, Data: entries, Error: nil}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(response)
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), text)
			return err
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	return command
}

func newDiffCommand() *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "diff <id>",
		Short: "Show uncommitted changes for a topic",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspace, err := commandWorkspace()
			if err != nil {
				return writeCommandError(cmd, "diff", jsonOutput, err)
			}
			id, err := domain.ParseTopicIdentity(args[0])
			if err != nil {
				return writeCommandError(cmd, "diff", jsonOutput, err)
			}
			repository, err := gitproxy.Open(workspace.Database)
			if err != nil {
				return writeCommandError(cmd, "diff", jsonOutput, err)
			}
			text, err := repository.Diff(id.String() + "-*")
			if err != nil {
				return writeCommandError(cmd, "diff", jsonOutput, err)
			}
			response := diffOutput{Command: "diff", OK: true, Data: diffData{Text: text, Empty: strings.TrimSpace(text) == ""}, Error: nil}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(response)
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), text)
			return err
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	return command
}

type gitLogEntry struct {
	Commit  string `json:"commit"`
	Time    string `json:"time"`
	Message string `json:"message"`
}

type logOutput struct {
	Command string        `json:"command"`
	OK      bool          `json:"ok"`
	Data    []gitLogEntry `json:"data"`
	Error   any           `json:"error"`
}

type diffOutput struct {
	Command string   `json:"command"`
	OK      bool     `json:"ok"`
	Data    diffData `json:"data"`
	Error   any      `json:"error"`
}

type diffData struct {
	Text  string `json:"text"`
	Empty bool   `json:"empty"`
}

func parseGitLog(text string) []gitLogEntry {
	entries := make([]gitLogEntry, 0)
	for _, line := range strings.Split(strings.TrimSpace(text), "\n") {
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) == 3 {
			entries = append(entries, gitLogEntry{Commit: parts[0], Time: parts[1], Message: parts[2]})
		}
	}
	return entries
}
