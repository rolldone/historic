package cmd

import (
	"encoding/json"
	"fmt"

	"historic/internal/config"
	"historic/internal/indexer"

	"github.com/spf13/cobra"
)

func newDoctorCommand() *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{Use: "doctor", Short: "Diagnose Historic workspace compatibility", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		workspace, err := commandWorkspace()
		if err != nil {
			return writeCommandError(cmd, "doctor", jsonOutput, err)
		}
		diagnosis := indexer.Diagnose(workspace, Version)
		data := doctorData{BinaryVersion: diagnosis.BinaryVersion, ExecutablePath: diagnosis.ExecutablePath, WorkspaceFormatVersion: diagnosis.WorkspaceFormat, CurrentIndexSchemaVersion: diagnosis.CurrentIndexSchema, RequiredIndexSchemaVersion: diagnosis.RequiredIndexSchema, Status: diagnosis.Status, MarkdownValid: diagnosis.MarkdownValid, IndexExists: diagnosis.IndexExists, Recommendation: diagnosis.Recommendation}
		response := doctorOutput{Command: "doctor", OK: diagnosis.Status == "compatible", Data: data}
		if jsonOutput {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(response)
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\nBinary: %s\nIndex schema: %d (required %d)\nRecommendation: %s\n", data.Status, data.BinaryVersion, data.CurrentIndexSchemaVersion, data.RequiredIndexSchemaVersion, data.Recommendation)
		return err
	}}
	command.Flags().BoolVar(&jsonOutput, "json", false, "output stable JSON")
	return command
}

type doctorData struct {
	BinaryVersion              string `json:"binary_version"`
	ExecutablePath             string `json:"executable_path"`
	WorkspaceFormatVersion     int    `json:"workspace_format_version"`
	CurrentIndexSchemaVersion  int    `json:"current_index_schema_version"`
	RequiredIndexSchemaVersion int    `json:"required_index_schema_version"`
	Status                     string `json:"status"`
	MarkdownValid              bool   `json:"markdown_valid"`
	IndexExists                bool   `json:"index_exists"`
	Recommendation             string `json:"recommendation"`
}
type doctorOutput struct {
	Command string     `json:"command"`
	OK      bool       `json:"ok"`
	Data    doctorData `json:"data"`
	Error   any        `json:"error"`
}

var _ = config.WorkspaceFormatVersion
