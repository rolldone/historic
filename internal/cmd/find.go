package cmd

import (
	"encoding/json"
	"fmt"

	"historic/internal/domain"
	"historic/internal/search"

	"github.com/spf13/cobra"
)

func newFindCommand() *cobra.Command {
	var status, folder, searchType, id string
	var activeOnly, archivedOnly, jsonOutput bool
	command := &cobra.Command{
		Use:   "find <keyword>",
		Short: "Find text in the Historic FTS5 index",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspace, err := commandWorkspace()
			if err != nil {
				return writeCommandError(cmd, "find", jsonOutput, err)
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
				Keyword: args[0], Status: parsedStatus, Folder: folder, Type: searchType, ID: parsedID,
				ActiveOnly: activeOnly, ArchivedOnly: archivedOnly,
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
				line := fmt.Sprintf("%s  %-10s  %s  %s\n", result.ID, result.Status, result.Path, search.HighlightHuman(result.Snippet, args[0]))
				if _, err = fmt.Fprint(cmd.OutOrStdout(), line); err != nil {
					return err
				}
			}
			return nil
		},
	}
	command.Flags().StringVar(&status, "status", "", "filter by lifecycle status")
	command.Flags().StringVar(&folder, "folder", "", "filter by workspace-relative folder")
	command.Flags().StringVar(&searchType, "type", "", "filter by inferred file type")
	command.Flags().StringVar(&id, "id", "", "filter by five-digit topic ID")
	command.Flags().BoolVar(&activeOnly, "active", false, "search active topics only")
	command.Flags().BoolVar(&archivedOnly, "archived", false, "search archived topics only")
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	return command
}

type findOutput struct {
	Command string          `json:"command"`
	OK      bool            `json:"ok"`
	Data    []search.Result `json:"data"`
	Error   any             `json:"error"`
}
