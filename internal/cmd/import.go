package cmd

import (
	"encoding/json"
	"fmt"

	"historic/internal/domain"
	"historic/internal/repository"

	"github.com/spf13/cobra"
)

func newImportCommand() *cobra.Command {
	var force, jsonOutput bool
	command := &cobra.Command{
		Use:   "import <id>",
		Short: "Import an archived topic into the active workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspace, err := commandWorkspace()
			if err != nil {
				return writeCommandError(cmd, "import", jsonOutput, err)
			}
			id, err := domain.ParseID(args[0])
			if err != nil {
				return writeCommandError(cmd, "import", jsonOutput, err)
			}
			topic, err := repository.NewTopicStore(workspace).ImportTopic(id, force)
			if err != nil {
				return writeCommandError(cmd, "import", jsonOutput, err)
			}
			response := importOutput{Command: "import", OK: true, Data: importData{ID: topic.ID.String(), Title: topic.Title, Status: topic.Status.String(), Storage: "open", Path: workspace.RelativePath(topic.Path)}, Error: nil}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(response)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Imported %s to %s\n", topic.ID, response.Data.Path)
			return err
		},
	}
	command.Flags().BoolVar(&force, "force", false, "replace an existing active topic")
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	return command
}

type importOutput struct {
	Command string     `json:"command"`
	OK      bool       `json:"ok"`
	Data    importData `json:"data"`
	Error   any        `json:"error"`
}

type importData struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	Storage string `json:"storage"`
	Path    string `json:"path"`
}
