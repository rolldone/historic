package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"historic/internal/config"
	"historic/internal/display"
	"historic/internal/domain"
	"historic/internal/repository"

	"github.com/spf13/cobra"
)

func newListCommand() *cobra.Command {
	var closedOnly, jsonOutput bool
	var sortValue string
	command := &cobra.Command{
		Use:   "list",
		Short: "List Historic topics",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			sortKey, err := repository.ParseListSort(sortValue)
			if err != nil {
				return writeCommandError(cmd, "list", jsonOutput, err)
			}
			workspace, err := commandWorkspace()
			if err != nil {
				return writeCommandError(cmd, "list", jsonOutput, err)
			}
			stored, err := repository.NewTopicStore(workspace).ListTopicsSorted(closedOnly, sortKey)
			if err != nil {
				return writeCommandError(cmd, "list", jsonOutput, err)
			}
			views := collapseListCopies(stored)
			result := topicListOutput{Command: "list", OK: true, Data: views, Error: nil}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
			}
			if len(views) == 0 {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "No topics found.")
				return err
			}
			out := cmd.OutOrStdout()
			copies := make(map[string]int, len(views))
			for _, view := range views {
				copies[view.ID]++
			}
			grouped := make(map[string]bool, len(views))
			for _, view := range views {
				// Two rows survive only when the copies carry different stamps,
				// so a repeated ID here always marks a real divergence worth
				// separating with a thin rule.
				if copies[view.ID] > 1 && !grouped[view.ID] {
					grouped[view.ID] = true
					if _, err = fmt.Fprintln(out, display.Divider(display.TerminalWidth())); err != nil {
						return err
					}
				}
				line := fmt.Sprintf("%s  %-6s  %-9s  %s  %s  %s  updated_at=%s", view.ID, view.Storage, displayTopicStatus(view.ComputedStatus), displayProgressPercent(view.ProgressPercent), view.Title, view.Path, displayTopicStamp(view.TopicView))
				if notice := display.NewerCopyNotice(out, domain.CopyRegionCaption(view.NewerCopy), false); notice != "" {
					line += "  " + notice
				}
				if _, err = fmt.Fprintln(out, line); err != nil {
					return err
				}
			}
			return nil
		},
	}
	command.Flags().BoolVar(&closedOnly, "closed", false, "list only closed topics")
	command.Flags().StringVar(&sortValue, "sort", string(repository.ListSortUpdated), "sort order: updated, created, title, or id")
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
				return writeCommandError(cmd, "show", jsonOutput, err)
			}
			id, err := domain.ParseTopicIdentity(args[0])
			if err != nil {
				return writeCommandError(cmd, "show", jsonOutput, err)
			}
			views, err := repository.NewTopicStore(workspace).ShowTopics(id, includeClosed)
			if err != nil {
				return writeCommandError(cmd, "show", jsonOutput, err)
			}
			result := topicShowOutput{Command: "show", OK: true, Data: views, Error: nil}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
			}
			for _, view := range views {
				notice := display.NewerCopyNotice(cmd.OutOrStdout(), domain.CopyRegionCaption(view.NewerCopy), false)
				suffix := ""
				if notice != "" {
					suffix = "  " + notice
				}
				if _, err = fmt.Fprintf(cmd.OutOrStdout(), "%s  %s [%s] status=%s %s updated_at=%s mtime=%s%s\nPath: %s\n", view.ID, view.Title, strings.ToUpper(view.Storage), displayTopicStatus(view.ComputedStatus), displayProgressPercent(view.ProgressPercent), displayTopicStamp(view), view.Mtime, suffix, view.Path); err != nil {
					return err
				}
				if view.Extends != "" {
					if _, err = fmt.Fprintf(cmd.OutOrStdout(), "extends: %s\n", view.Extends); err != nil {
						return err
					}
				}
				for _, file := range view.Files {
					line := fmt.Sprintf("- %s (%s)", file.Path, file.Status)
					if stamp := strings.TrimSpace(file.UpdatedAt); stamp != "" {
						line += " updated_at=" + display.FormatUpdatedStamp(stamp)
					}
					if file.ForwardedTo != "" {
						line += fmt.Sprintf(" → %s", file.ForwardedTo)
					}
					if _, err = fmt.Fprintln(cmd.OutOrStdout(), line); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
	command.Flags().BoolVar(&includeClosed, "closed", false, "show only closed versions")
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	return command
}

type topicListOutput struct {
	Command string            `json:"command"`
	OK      bool              `json:"ok"`
	Data    []listedTopicView `json:"data"`
	Error   any               `json:"error"`
}

// listedTopicView is a list row: the repository view plus the additive flag that
// marks a row standing in for an identical mirrored pair. Embedding keeps every
// existing JSON field and its position, so older consumers are unaffected.
type listedTopicView struct {
	repository.TopicView
	Mirrored bool `json:"mirrored,omitempty"`
}

// collapseListCopies keeps both rows of a mirrored pair only when the copies
// carry different freshness. Identical copies — including pre-v9 pairs that fall
// back to the same calendar date — collapse to the open row, which holds the
// active work, and only the additive Mirrored flag records the second copy.
func collapseListCopies(views []repository.TopicView) []listedTopicView {
	listed := make([]listedTopicView, len(views))
	for index, view := range views {
		listed[index] = listedTopicView{TopicView: view}
	}
	if len(listed) < 2 {
		return listed
	}
	rows := make([]display.CopyRow, len(listed))
	for index, view := range listed {
		rows[index] = display.CopyRow{
			Key:   view.ID,
			Stamp: listCopyStamp(view.TopicView),
			Open:  view.Storage == domain.StorageOpen.String(),
		}
	}
	kept := display.CollapseIdenticalCopies(rows)
	collapsed := make([]listedTopicView, 0, len(kept))
	for _, row := range kept {
		view := listed[row.Index]
		view.Mirrored = row.Mirrored
		collapsed = append(collapsed, view)
	}
	return collapsed
}

// listCopyStamp is the freshness value two copies are compared on: the
// high-resolution manifest stamp, then the calendar `updated` date, then
// `created`. It matches the ordering key so a pair that ties in sort order also
// ties here.
func listCopyStamp(view repository.TopicView) string {
	if stamp := strings.TrimSpace(view.UpdatedAt); stamp != "" {
		return stamp
	}
	if daily := strings.TrimSpace(view.Updated); daily != "" {
		return daily
	}
	return view.Created
}

type topicShowOutput struct {
	Command string                 `json:"command"`
	OK      bool                   `json:"ok"`
	Data    []repository.TopicView `json:"data"`
	Error   any                    `json:"error"`
}

func commandWorkspace() (config.Workspace, error) {
	current, err := os.Getwd()
	if err != nil {
		return config.Workspace{}, fmt.Errorf("get current directory: %w", err)
	}
	root, err := config.DiscoverRootForReadiness(current)
	if err != nil {
		return config.Workspace{}, err
	}
	workspace := config.NewWorkspace(root)
	_, err = config.ValidateWorkspace(workspace)
	return workspace, err
}

func commandWorkspaceForRecovery() (config.Workspace, error) {
	current, err := os.Getwd()
	if err != nil {
		return config.Workspace{}, fmt.Errorf("get current directory: %w", err)
	}
	root, err := config.DiscoverRootForReadiness(current)
	if err != nil {
		return config.Workspace{}, err
	}
	return config.NewWorkspace(root), nil
}

func displayTopicStatus(status string) string {
	if status == "" {
		return "-"
	}
	return status
}

// displayTopicStamp prefers the topic's high-resolution stamp and falls back to
// the calendar date so a legacy topic still prints something meaningful.
func displayTopicStamp(view repository.TopicView) string {
	if stamp := strings.TrimSpace(view.UpdatedAt); stamp != "" {
		return display.FormatUpdatedStamp(stamp)
	}
	if daily := strings.TrimSpace(view.Updated); daily != "" {
		return daily
	}
	return view.Created
}

func displayProgressPercent(percent *int) string {
	if percent == nil {
		return "[progress -]"
	}
	return fmt.Sprintf("[progress %d%%]", *percent)
}

func readinessErrorValue(err error) any {
	var readiness *config.ReadinessError
	if errors.As(err, &readiness) {
		return struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}{Code: string(readiness.Code), Message: readiness.Message}
	}
	return strings.TrimSpace(err.Error())
}

func writeCommandError(cmd *cobra.Command, name string, jsonOutput bool, err error) error {
	if !jsonOutput {
		return err
	}
	response := struct {
		Command string `json:"command"`
		OK      bool   `json:"ok"`
		Data    any    `json:"data"`
		Error   any    `json:"error"`
	}{Command: name, OK: false, Data: nil, Error: readinessErrorValue(err)}
	if encodeErr := json.NewEncoder(cmd.OutOrStdout()).Encode(response); encodeErr != nil {
		return encodeErr
	}
	return SilentError{Err: err}
}
