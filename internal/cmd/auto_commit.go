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
// commit field required for auto-commit reporting plus the additive bump count.
type saveResponseData struct {
	lifecycle.MirrorAllResult
	BumpedFiles int                         `json:"bumped_files"`
	Commit      *lifecycle.AutoCommitResult `json:"commit"`
}

// writeBumpHuman prints the content-change bump summary and any non-fatal
// warnings. It stays silent when nothing was bumped and there were no warnings.
func writeBumpHuman(cmd *cobra.Command, result lifecycle.BumpResult) error {
	if result.Count() > 0 {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "bump updated: %d file\n", result.Count()); err != nil {
			return err
		}
	}
	for _, warning := range result.Warnings {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "bump warning (tidak fatal): %s\n", warning); err != nil {
			return err
		}
	}
	return nil
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
