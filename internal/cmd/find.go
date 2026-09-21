package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"historic/internal/domain"
	"historic/internal/search"

	"github.com/spf13/cobra"
)

func newFindCommand() *cobra.Command {
	var status, tag, folder, searchType, id string
	var createdAfter, createdBefore, updatedAfter, updatedBefore string
	var openOnly, closedOnly, activeOnly, archivedOnly, jsonOutput bool
	command := &cobra.Command{
		Use:   "find <keyword>",
		Short: "Find text in the Historic FTS5 index",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspace, err := commandWorkspace()
			if err != nil {
				return writeCommandError(cmd, "find", jsonOutput, err)
			}
			if activeOnly || archivedOnly {
				return writeCommandError(cmd, "find", jsonOutput, fmt.Errorf("--active and --archived are no longer supported; use --open or --closed"))
			}
			var parsedStatus domain.Status
			if status != "" {
				parsedStatus, err = domain.ParseStatus(status)
				if err != nil {
					return writeCommandError(cmd, "find", jsonOutput, err)
				}
			}
			var parsedID domain.ID
			if id != "" {
				parsedID, err = domain.ParseID(id)
				if err != nil {
					return writeCommandError(cmd, "find", jsonOutput, err)
				}
			}
			results, err := search.Find(workspace, search.Options{
				Keyword: args[0], Status: parsedStatus, Tags: splitTags(tag), Folder: folder, Type: searchType, ID: parsedID,
				OpenOnly: openOnly, ClosedOnly: closedOnly, CreatedAfter: createdAfter, CreatedBefore: createdBefore,
				UpdatedAfter: updatedAfter, UpdatedBefore: updatedBefore,
			})
			if err != nil {
				return writeCommandError(cmd, "find", jsonOutput, err)
			}
			response := findOutput{Command: "find", OK: true, Data: results, Error: nil}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(response)
			}
			if len(results) == 0 {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "No matches found.")
				return err
			}
			for _, result := range results {
				line := fmt.Sprintf("%s  %-10s  [%s]  %s  %s\n", result.TopicID, result.Status, strings.ToUpper(result.Storage), result.Path, search.HighlightHuman(result.Snippet, args[0]))
				if _, err = fmt.Fprint(cmd.OutOrStdout(), line); err != nil {
					return err
				}
			}
			return nil
		},
	}
	command.Flags().StringVar(&status, "status", "", "filter by lifecycle status")
	command.Flags().StringVar(&tag, "tag", "", "filter by topic or file tag; repeat as comma-separated values")
	command.Flags().StringVar(&folder, "folder", "", "filter by topic-relative folder")
	command.Flags().StringVar(&searchType, "type", "", "filter by topic or file type")
	command.Flags().StringVar(&id, "topic", "", "filter by five-digit topic ID")
	command.Flags().StringVar(&id, "id", "", "filter by five-digit topic ID")
	command.Flags().StringVar(&createdAfter, "created-after", "", "filter created date from YYYY-MM-DD")
	command.Flags().StringVar(&createdBefore, "created-before", "", "filter created date through YYYY-MM-DD")
	command.Flags().StringVar(&updatedAfter, "updated-after", "", "filter updated date from YYYY-MM-DD")
	command.Flags().StringVar(&updatedBefore, "updated-before", "", "filter updated date through YYYY-MM-DD")
	command.Flags().BoolVar(&openOnly, "open", false, "search open topics only")
	command.Flags().BoolVar(&closedOnly, "closed", false, "search closed topics only")
	command.Flags().BoolVar(&activeOnly, "active", false, "deprecated alias; rejected, use --open")
	command.Flags().BoolVar(&archivedOnly, "archived", false, "deprecated alias; rejected, use --closed")
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	return command
}

func splitTags(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}
	return result
}

type findOutput struct {
	Command string          `json:"command"`
	OK      bool            `json:"ok"`
	Data    []search.Result `json:"data"`
	Error   any             `json:"error"`
}
