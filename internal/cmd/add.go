package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/repository"

	"github.com/spf13/cobra"
)

func newAddCommand() *cobra.Command {
	var topicID string
	var force bool
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a Markdown entry to an active topic",
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
			if topicID == "" {
				topicID, err = activeTopicID(workspace)
				if err != nil {
					return err
				}
			}
			id, err := domain.ParseTopicIdentity(topicID)
			if err != nil {
				return err
			}
			entry, err := repository.NewTopicStore(workspace).AddEntry(id, args[0], force)
			if err != nil {
				return err
			}
			result := addOutput{Command: "add", OK: true, Data: addData{ID: entry.ID.String(), TopicID: id.String(), File: workspace.RelativePath(entry.Path), Title: entry.Title}, Error: nil}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Added %s to %s\n", result.Data.Title, result.Data.File)
			return err
		},
	}
	command.Flags().StringVar(&topicID, "id", "", "target active topic ID")
	command.Flags().BoolVar(&force, "force", false, "overwrite an existing Markdown file")
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	return command
}

type addOutput struct {
	Command string  `json:"command"`
	OK      bool    `json:"ok"`
	Data    addData `json:"data,omitempty"`
	Error   any     `json:"error"`
}

type addData struct {
	ID      string `json:"id"`
	TopicID string `json:"topic_id"`
	File    string `json:"file"`
	Title   string `json:"title"`
}

func activeTopicID(workspace config.Workspace) (string, error) {
	entries, err := os.ReadDir(workspace.Histories)
	if err != nil {
		return "", fmt.Errorf("scan active topics: %w", err)
	}
	var ids []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		match := regexp.MustCompile(`^([0-9]{5}|[0-9]{13})-(.+)$`).FindStringSubmatch(entry.Name())
		if len(match) == 3 {
			if _, err := domain.ParseTopicIdentity(match[1]); err == nil {
				ids = append(ids, match[1])
			}
		}
	}
	if len(ids) != 1 {
		return "", fmt.Errorf("%w: specify --id when active topic count is %d", domain.ErrConflict, len(ids))
	}
	return ids[0], nil
}
