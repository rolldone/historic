package cmd

import (
	"encoding/json"
	"fmt"

	"historic/internal/domain"
	"historic/internal/lifecycle"

	"github.com/spf13/cobra"
)

// newDeleteCommand deletes one file from the current open topic. Requiring
// --yes makes scripts and humans acknowledge the destructive consequence.
func newDeleteCommand() *cobra.Command {
	var confirmed, jsonOutput bool
	command := &cobra.Command{
		Use:   "delete <path>",
		Short: "Delete one file or asset from the current open topic",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirmed {
				return writeCommandError(cmd, "delete", jsonOutput, fmt.Errorf("%w: deleting %q requires --yes", domain.ErrConflict, args[0]))
			}
			workspace, err := commandWorkspace()
			if err != nil {
				return writeCommandError(cmd, "delete", jsonOutput, err)
			}
			change, err := lifecycle.NewService(workspace).DeleteFile(args[0])
			if err != nil {
				return writeCommandError(cmd, "delete", jsonOutput, err)
			}
			return writeDeleteResult(cmd, "delete", change, jsonOutput)
		},
	}
	command.Flags().BoolVar(&confirmed, "yes", false, "confirm permanent deletion")
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	return command
}

func newDeleteTopicCommand() *cobra.Command {
	return newTopicDestructiveCommand("delete-topic", "Delete one selected open or closed topic", false)
}

func newPurgeCommand() *cobra.Command {
	return newTopicDestructiveCommand("purge", "Permanently purge one selected open or closed topic", true)
}

func newTopicDestructiveCommand(name, short string, purge bool) *cobra.Command {
	var openOnly, closedOnly, confirmed, jsonOutput bool
	var confirmation string
	command := &cobra.Command{
		Use:   name + " <id>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if openOnly && closedOnly {
				return writeCommandError(cmd, name, jsonOutput, fmt.Errorf("%w: --open and --closed cannot be combined", domain.ErrConflict))
			}
			if !confirmed {
				return writeCommandError(cmd, name, jsonOutput, fmt.Errorf("%w: %s requires --yes", domain.ErrConflict, name))
			}
			if purge && confirmation != args[0] {
				return writeCommandError(cmd, name, jsonOutput, fmt.Errorf("%w: purge requires --confirm %s", domain.ErrConflict, args[0]))
			}
			id, err := domain.ParseTopicIdentity(args[0])
			if err != nil {
				return writeCommandError(cmd, name, jsonOutput, err)
			}
			workspace, err := commandWorkspace()
			if err != nil {
				return writeCommandError(cmd, name, jsonOutput, err)
			}
			scope := domain.StorageState("")
			if openOnly {
				scope = domain.StorageOpen
			} else if closedOnly {
				scope = domain.StorageClosed
			}
			service := lifecycle.NewService(workspace)
			var change lifecycle.DeleteChange
			if purge {
				change, err = service.PurgeTopic(id, scope)
			} else {
				change, err = service.DeleteTopic(id, scope)
			}
			if err != nil {
				return writeCommandError(cmd, name, jsonOutput, err)
			}
			return writeDeleteResult(cmd, name, change, jsonOutput)
		},
	}
	command.Flags().BoolVar(&openOnly, "open", false, "target the open workdir")
	command.Flags().BoolVar(&closedOnly, "closed", false, "target the closed snapshot")
	command.Flags().BoolVar(&confirmed, "yes", false, "confirm destructive deletion")
	if purge {
		command.Flags().StringVar(&confirmation, "confirm", "", "type the exact topic ID to confirm permanent purge")
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	return command
}

func writeDeleteResult(cmd *cobra.Command, name string, change lifecycle.DeleteChange, jsonOutput bool) error {
	data := deleteData{ID: change.ID.String(), Title: change.Title, Path: change.Path, Storage: change.Storage.String(), Target: change.Target, Permanent: change.Permanent}
	if jsonOutput {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(deleteOutput{Command: name, OK: true, Data: data, Error: nil})
	}
	consequence := "moved to protected trash; the closed snapshot is unchanged"
	if change.Target == "file" {
		consequence = "removed from the current read model; closed storage is unchanged"
	}
	if change.Permanent {
		consequence = "permanently removed"
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Deleted %s %s [%s] at %s: %s\n", data.ID, data.Target, data.Storage, data.Path, consequence)
	return err
}

type deleteOutput struct {
	Command string     `json:"command"`
	OK      bool       `json:"ok"`
	Data    deleteData `json:"data"`
	Error   any        `json:"error"`
}

type deleteData struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Path      string `json:"path"`
	Storage   string `json:"storage"`
	Target    string `json:"target"`
	Permanent bool   `json:"permanent"`
}
