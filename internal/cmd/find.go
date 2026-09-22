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
	var status, statusNot, tag, folder, searchType, id, sortOption string
	var createdAfter, createdBefore, updatedAfter, updatedBefore string
	var page, pageSize int
	var openOnly, closedOnly, jsonOutput bool
	command := &cobra.Command{
		Use:   "find <keyword>",
		Short: "Find text in the Historic FTS5 index",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspace, err := commandWorkspace()
			if err != nil {
				return writeCommandError(cmd, "find", jsonOutput, err)
			}
			if cmd.Flags().Changed("active") || cmd.Flags().Changed("archived") {
				return writeCommandError(cmd, "find", jsonOutput, fmt.Errorf("--active and --archived are no longer supported; use --open or --closed"))
			}
			if page < 1 {
				return writeCommandError(cmd, "find", jsonOutput, fmt.Errorf("page must be at least 1"))
			}
			if pageSize < 1 || pageSize > search.MaxPageSize {
				return writeCommandError(cmd, "find", jsonOutput, fmt.Errorf("page-size must be between 1 and %d", search.MaxPageSize))
			}
			parsedStatusIn, err := parseStatusList(status, "--status")
			if err != nil {
				return writeCommandError(cmd, "find", jsonOutput, err)
			}
			parsedStatusNot, err := parseStatusList(statusNot, "--status-not")
			if err != nil {
				return writeCommandError(cmd, "find", jsonOutput, err)
			}
			var parsedStatus domain.Status
			if len(parsedStatusIn) == 1 {
				parsedStatus = parsedStatusIn[0]
			}
			var parsedID domain.ID
			if id != "" {
				parsedID, err = domain.ParseTopicIdentity(id)
				if err != nil {
					return writeCommandError(cmd, "find", jsonOutput, err)
				}
			}
			options := search.Options{Status: parsedStatus, StatusIn: parsedStatusIn, StatusNot: parsedStatusNot, Tags: splitTags(tag), Folder: folder, Type: searchType, ID: parsedID, Sort: sortOption, Page: page, PageSize: pageSize, OpenOnly: openOnly, ClosedOnly: closedOnly, CreatedAfter: createdAfter, CreatedBefore: createdBefore, UpdatedAfter: updatedAfter, UpdatedBefore: updatedBefore}
			var resultPage search.SearchPage
			if strings.TrimSpace(args[0]) == "" {
				resultPage, err = search.RecentTopicsPage(workspace, options)
			} else {
				options.Keyword = args[0]
				resultPage, err = search.FindPage(workspace, options)
			}
			if err != nil {
				return writeCommandError(cmd, "find", jsonOutput, err)
			}
			response := findOutput{Command: "find", OK: true, Data: resultPage, Error: nil}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(response)
			}
			results := resultPage.Items
			if len(results) == 0 {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "No matches found.")
				return err
			}
			for _, result := range results {
				snippet := result.Snippet
				if strings.TrimSpace(args[0]) == "" {
					snippet = result.Title
				}
				line := fmt.Sprintf("%s  %-10s  [%s]  %s  %s\n", result.TopicID, result.Status, strings.ToUpper(result.Storage), result.Path, search.HighlightHuman(snippet, args[0]))
				if _, err = fmt.Fprint(cmd.OutOrStdout(), line); err != nil {
					return err
				}
			}
			pagination := resultPage.Pagination
			if pagination.HasMore {
				if _, err = fmt.Fprintf(cmd.OutOrStdout(), "Page %d · showing %d results · more results available\nUse --page %d --page-size %d\n", pagination.Page, len(results), pagination.Page+1, pagination.PageSize); err != nil {
					return err
				}
			} else if _, err = fmt.Fprintf(cmd.OutOrStdout(), "Page %d · showing %d results · no more results\n", pagination.Page, len(results)); err != nil {
				return err
			}
			return nil
		},
	}
	command.Flags().StringVar(&status, "status", "", "filter by lifecycle status")
	command.Flags().StringVar(&statusNot, "status-not", "", "exclude lifecycle statuses; comma-separated")
	command.Flags().StringVar(&tag, "tag", "", "filter by topic or file tag; repeat as comma-separated values")
	command.Flags().StringVar(&folder, "folder", "", "filter by topic-relative folder")
	command.Flags().StringVar(&searchType, "type", "", "filter by topic or file type")
	command.Flags().StringVar(&sortOption, "sort", "", "sort by relevance, updated, created, or title")
	command.Flags().StringVar(&id, "topic", "", "filter by five-digit topic ID")
	command.Flags().StringVar(&id, "id", "", "filter by five-digit topic ID")
	command.Flags().StringVar(&createdAfter, "created-after", "", "filter created date from YYYY-MM-DD")
	command.Flags().StringVar(&createdBefore, "created-before", "", "filter created date through YYYY-MM-DD")
	command.Flags().StringVar(&updatedAfter, "updated-after", "", "filter updated date from YYYY-MM-DD")
	command.Flags().StringVar(&updatedBefore, "updated-before", "", "filter updated date through YYYY-MM-DD")
	command.Flags().IntVar(&page, "page", search.DefaultPage, "result page number")
	command.Flags().IntVar(&pageSize, "page-size", search.DefaultPageSize, "results per page (maximum 100)")
	command.Flags().BoolVar(&openOnly, "open", false, "search open topics only")
	command.Flags().BoolVar(&closedOnly, "closed", false, "search closed topics only")
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	_ = command.Flags().Bool("active", false, "deprecated; rejected, use --open")
	_ = command.Flags().Bool("archived", false, "deprecated; rejected, use --closed")
	return command
}

func parseStatusList(value, flag string) ([]domain.Status, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parts := strings.Split(value, ",")
	result := make([]domain.Status, 0, len(parts))
	seen := make(map[domain.Status]struct{}, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("invalid empty status in %s; use comma-separated status names", flag)
		}
		status, err := domain.ParseStatus(part)
		if err != nil {
			return nil, fmt.Errorf("invalid %s value %q: %w", flag, part, err)
		}
		if _, exists := seen[status]; !exists {
			result = append(result, status)
			seen[status] = struct{}{}
		}
	}
	return result, nil
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
	Command string            `json:"command"`
	OK      bool              `json:"ok"`
	Data    search.SearchPage `json:"data"`
	Error   any               `json:"error"`
}
