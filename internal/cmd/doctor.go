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
		data := doctorData{BinaryVersion: diagnosis.BinaryVersion, ExecutablePath: diagnosis.ExecutablePath, WorkspaceFormatVersion: diagnosis.WorkspaceFormat, CurrentIndexSchemaVersion: diagnosis.CurrentIndexSchema, RequiredIndexSchemaVersion: diagnosis.RequiredIndexSchema, Status: diagnosis.Status, MarkdownValid: diagnosis.MarkdownValid, IndexExists: diagnosis.IndexExists, Recommendation: diagnosis.Recommendation, AppVersionName: diagnosis.AppVersionName, AppVersionCode: diagnosis.AppVersionCode, StoredVersionName: diagnosis.StoredVersionName, StoredVersionCode: diagnosis.StoredVersionCode, CompatibilityAction: diagnosis.CompatibilityAction}
		response := doctorOutput{Command: "doctor", OK: diagnosis.Status == "compatible", Data: data}
		if jsonOutput {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(response)
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\nBinary: %s (code %d)\nStored app: %s (code %d)\nWorkspace format: %d\nIndex schema: %d (required %d)\nCompatibility action: %s\nRecommendation: %s\n", data.Status, data.AppVersionName, data.AppVersionCode, data.StoredVersionName, data.StoredVersionCode, data.WorkspaceFormatVersion, data.CurrentIndexSchemaVersion, data.RequiredIndexSchemaVersion, data.CompatibilityAction, data.Recommendation)
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
	AppVersionName             string `json:"app_version_name"`
	AppVersionCode             int    `json:"app_version_code"`
	StoredVersionName          string `json:"stored_version_name"`
	StoredVersionCode          int    `json:"stored_version_code"`
	CompatibilityAction        string `json:"compatibility_action"`
}
type doctorOutput struct {
	Command string     `json:"command"`
	OK      bool       `json:"ok"`
	Data    doctorData `json:"data"`
	Error   any        `json:"error"`
}

var _ = config.WorkspaceFormatVersion
