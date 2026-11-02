package lifecycle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/gitproxy"
)

// AutoCommitResult reports the outcome of an automatic internal commit. When
// Committed is false, Reason explains why (no-changes, no-repository, or a
// failure message that callers surface as a non-fatal warning).
type AutoCommitResult struct {
	Committed    bool   `json:"committed"`
	Hash         string `json:"hash,omitempty"`
	Message      string `json:"message,omitempty"`
	ChangedPaths int    `json:"changed_paths,omitempty"`
	Reason       string `json:"reason,omitempty"`
}

// AutoCommitTopic commits mirror changes produced by a single-topic operation.
func AutoCommitTopic(workspace config.Workspace, command string, id domain.ID) AutoCommitResult {
	return AutoCommit(workspace, command+" "+id.String())
}

// AutoCommitAll commits mirror changes produced by a bulk operation as one
// aggregate commit rather than one commit per topic.
func AutoCommitAll(workspace config.Workspace, command string) AutoCommitResult {
	return AutoCommit(workspace, command+" --all")
}

// AutoCommitRecovery commits mirror changes produced by a workspace-wide
// operation without a single target topic.
func AutoCommitRecovery(workspace config.Workspace, command string) AutoCommitResult {
	return AutoCommit(workspace, command+" ?")
}

// AutoCommit stages and commits every internal repository change under one
// deterministic message of the form "auto: <label> (<n> path berubah)". It never
// pushes, adds remotes, or invents Git identity values.
func AutoCommit(workspace config.Workspace, label string) AutoCommitResult {
	if _, err := os.Stat(filepath.Join(workspace.Database, ".git")); err != nil {
		return AutoCommitResult{Reason: "no-repository"}
	}
	repository := gitproxy.Repository{Root: workspace.Database}
	status, err := repository.StatusPorcelain()
	if err != nil {
		return AutoCommitResult{Reason: err.Error()}
	}
	changed := countPorcelainEntries(status)
	if changed == 0 {
		return AutoCommitResult{Reason: "no-changes"}
	}
	message := fmt.Sprintf("auto: %s (%d path berubah)", label, changed)
	if err := repository.AddAll(); err != nil {
		return AutoCommitResult{Reason: err.Error()}
	}
	hash, err := repository.CommitConfigured(message)
	if err != nil {
		return AutoCommitResult{Reason: err.Error()}
	}
	return AutoCommitResult{Committed: true, Hash: shortHash(hash), Message: message, ChangedPaths: changed}
}

func countPorcelainEntries(status string) int {
	count := 0
	for _, line := range strings.Split(status, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

func shortHash(hash string) string {
	hash = strings.TrimSpace(hash)
	if len(hash) > 7 {
		return hash[:7]
	}
	return hash
}
