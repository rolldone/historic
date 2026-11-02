package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/lifecycle"

	"github.com/spf13/cobra"
)

// writeAutoCommitHuman prints the single auto-commit status line for a command.
// A nil result means the operation does not trigger auto-commit.
func writeAutoCommitHuman(cmd *cobra.Command, result *lifecycle.AutoCommitResult) error {
	if result == nil {
		return nil
	}
	if result.Committed {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "auto-commit: %s %s\n", result.Hash, result.Message)
		return err
	}
	if result.Reason == "no-changes" {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "auto-commit: skipped (no-changes)\n")
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "auto-commit gagal (tidak fatal): %s — jalankan `historic git add --all` lalu `historic git commit -m \"...\"` secara manual.\n", result.Reason)
	return err
}

// saveResponseData keeps the existing mirror fields flat and adds the nullable
// commit field required for auto-commit reporting.
type saveResponseData struct {
	lifecycle.MirrorAllResult
	Commit *lifecycle.AutoCommitResult `json:"commit"`
}

// closeAllResponseData keeps the existing close-all fields flat and adds the
// nullable commit field.
type closeAllResponseData struct {
	lifecycle.CloseAllResult
	Commit *lifecycle.AutoCommitResult `json:"commit"`
}

// closedTopicID extracts the topic ID when a relative path points at the closed
// mirror (.historic/.database/<topic-id>-<slug>/...).
func closedTopicID(workspace config.Workspace, relativePath string) (domain.ID, bool) {
	closedRoot := filepath.ToSlash(workspace.RelativePath(workspace.Database)) + "/"
	normalized := filepath.ToSlash(relativePath)
	if !strings.HasPrefix(normalized, closedRoot) {
		return "", false
	}
	remainder := strings.TrimPrefix(normalized, closedRoot)
	folder := strings.SplitN(remainder, "/", 2)[0]
	separator := strings.Index(folder, "-")
	if separator <= 0 {
		return "", false
	}
	id, err := domain.ParseTopicIdentity(folder[:separator])
	if err != nil {
		return "", false
	}
	return id, true
}
