package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"historic/internal/config"
	"historic/internal/indexer"

	"github.com/spf13/cobra"
)

func newDoctorCommand() *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{Use: "doctor", Short: "Diagnose Historic workspace compatibility", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		workspace, err := commandWorkspaceForRecovery()
		if err != nil {
			return writeCommandError(cmd, "doctor", jsonOutput, err)
		}
		diagnosis := indexer.Diagnose(workspace, Version)
		duplicates := diagnosis.DuplicateFileIDs
		if duplicates == nil {
			duplicates = make([]indexer.DuplicateFileID, 0)
		}
		data := doctorData{BinaryVersion: diagnosis.BinaryVersion, ExecutablePath: diagnosis.ExecutablePath, WorkspaceFormatVersion: diagnosis.WorkspaceFormat, CurrentIndexSchemaVersion: diagnosis.CurrentIndexSchema, RequiredIndexSchemaVersion: diagnosis.RequiredIndexSchema, Status: diagnosis.Status, MarkdownValid: diagnosis.MarkdownValid, IndexExists: diagnosis.IndexExists, Recommendation: diagnosis.Recommendation, AppVersionName: diagnosis.AppVersionName, AppVersionCode: diagnosis.AppVersionCode, StoredVersionName: diagnosis.StoredVersionName, StoredVersionCode: diagnosis.StoredVersionCode, CompatibilityAction: diagnosis.CompatibilityAction, DuplicateFileIDs: duplicates}
		response := doctorOutput{Command: "doctor", OK: diagnosis.Status == "compatible", Data: data}
		if jsonOutput {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(response)
		}
		recommendation := diagnosis.Recommendation
		if diagnosis.CompatibilityAction == string(config.ActionMigrateWorkspace) {
			recommendation = recommendation + "; run historic upgrade"
		}
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\nBinary: %s (code %d)\nStored app: %s (code %d)\nWorkspace format: %d\nIndex schema: %d (required %d)\nCompatibility action: %s\nRecommendation: %s\n", data.Status, data.AppVersionName, data.AppVersionCode, data.StoredVersionName, data.StoredVersionCode, data.WorkspaceFormatVersion, data.CurrentIndexSchemaVersion, data.RequiredIndexSchemaVersion, data.CompatibilityAction, recommendation); err != nil {
			return err
		}
		if len(data.DuplicateFileIDs) == 0 {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "Duplicate FileIDs: none")
			return err
		}
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Duplicate FileIDs: %d (scoped per topic; not an error)\n", len(data.DuplicateFileIDs)); err != nil {
			return err
		}
		for _, duplicate := range data.DuplicateFileIDs {
			topicIDs := make([]string, 0, len(duplicate.Topics))
			for _, topic := range duplicate.Topics {
				topicIDs = append(topicIDs, topic.TopicID)
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  file_id=%s storage=%s topics=%s\n", duplicate.FileID, duplicate.Storage, strings.Join(topicIDs, ",")); err != nil {
				return err
			}
		}
		return nil
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
	// DuplicateFileIDs reports the same FileID recorded in several topics.
	// Legal since index schema 8 (FileID is scoped per topic).
	DuplicateFileIDs []indexer.DuplicateFileID `json:"duplicate_file_ids"`
}
type doctorOutput struct {
	Command string     `json:"command"`
	OK      bool       `json:"ok"`
	Data    doctorData `json:"data"`
	Error   any        `json:"error"`
}

var _ = config.WorkspaceFormatVersion
