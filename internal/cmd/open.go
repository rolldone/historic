package cmd

import (
	"fmt"

	"historic/internal/domain"
	"historic/internal/lifecycle"
	"historic/internal/tui"

	"github.com/spf13/cobra"
)

func newOpenCommand() *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "open [id]",
		Short: "Open one closed topic",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return runStorageCommand(cmd, "open", args[0], jsonOutput, func(service lifecycle.Service, id domain.ID) (lifecycle.Change, error) {
					return service.Open(id)
				})
			}
			workspace, err := commandWorkspace()
			if err != nil {
				return writeCommandError(cmd, "open", jsonOutput, err)
			}
			if jsonOutput {
				return writeCommandError(cmd, "open", true, fmt.Errorf("topic ID is required with --json"))
			}
			selected, err := tui.PickTopic(workspace, tui.TopicPickerOptions{Input: cmd.InOrStdin(), Output: cmd.OutOrStdout(), Closed: true})
			if err != nil {
				return err
			}
			if selected == nil {
				return nil
			}
			return runStorageCommand(cmd, "open", selected.ID, false, func(service lifecycle.Service, id domain.ID) (lifecycle.Change, error) {
				return service.Open(id)
			})
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	return command
}
