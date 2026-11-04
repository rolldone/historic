package cmd

import (
	"encoding/json"
	"fmt"

	"historic/internal/domain"
	"historic/internal/lifecycle"

	"github.com/spf13/cobra"
)

func newReadCommand() *cobra.Command {
	var head int
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "read <path>",
		Short: "Print one managed Markdown file (open or closed)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if head < 0 {
				return writeCommandError(cmd, "read", jsonOutput, fmt.Errorf("--head must be zero or greater"))
			}
			workspace, err := commandWorkspace()
			if err != nil {
				return writeCommandError(cmd, "read", jsonOutput, err)
			}
			file, err := lifecycle.ReadFile(workspace, args[0], head)
			if err != nil {
				return writeCommandError(cmd, "read", jsonOutput, err)
			}
			if jsonOutput {
				response := readOutput{Command: "read", OK: true, Data: readData{
					Path:        file.Path,
					Frontmatter: newReadFrontmatter(file.Document.Frontmatter),
					Body:        file.Document.Body,
				}, Error: nil}
				return json.NewEncoder(cmd.OutOrStdout()).Encode(response)
			}
			if _, err := fmt.Fprint(cmd.OutOrStdout(), file.Raw); err != nil {
				return err
			}
			if file.Raw != "" && file.Raw[len(file.Raw)-1] != '\n' {
				_, err = fmt.Fprintln(cmd.OutOrStdout())
			}
			return err
		},
	}
	command.Flags().IntVar(&head, "head", 0, "print only the first N lines (default: whole file)")
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	return command
}

type readOutput struct {
	Command string   `json:"command"`
	OK      bool     `json:"ok"`
	Data    readData `json:"data"`
	Error   any      `json:"error"`
}

type readData struct {
	Path        string          `json:"path"`
	Frontmatter readFrontmatter `json:"frontmatter"`
	Body        string          `json:"body"`
}

// readFrontmatter mirrors domain.Frontmatter with the snake_case JSON keys
// used by the rest of the CLI contract.
type readFrontmatter struct {
	ID          string   `json:"id,omitempty"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Status      string   `json:"status"`
	Created     string   `json:"created"`
	Updated     string   `json:"updated,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Related     []string `json:"related,omitempty"`
	ForwardedTo string   `json:"forwarded_to,omitempty"`
}

func newReadFrontmatter(frontmatter domain.Frontmatter) readFrontmatter {
	result := readFrontmatter{
		ID:          frontmatter.ID.String(),
		Title:       frontmatter.Title,
		Description: frontmatter.Description,
		Status:      frontmatter.Status.String(),
		Created:     frontmatter.Created,
		Updated:     frontmatter.Updated,
		Tags:        frontmatter.Tags,
		ForwardedTo: frontmatter.ForwardedTo.String(),
	}
	for _, related := range frontmatter.Related {
		result.Related = append(result.Related, related.String())
	}
	return result
}
