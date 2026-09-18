package lifecycle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/gitproxy"
	"historic/internal/indexer"
)

// Restore restores one archived topic path from a snapshot into the active
// workspace. It refuses to overwrite an existing active topic unless force is
// explicitly requested.
func Restore(workspace config.Workspace, id domain.ID, snapshot string, force bool) error {
	if !id.Valid() {
		return fmt.Errorf("%w: %q", domain.ErrInvalidID, id)
	}
	if strings.TrimSpace(snapshot) == "" {
		return fmt.Errorf("%w: snapshot is empty", domain.ErrConflict)
	}
	repository, err := gitproxy.Open(workspace.Database)
	if err != nil {
		return err
	}
	if _, err := repository.Head(); err != nil {
		return fmt.Errorf("validate snapshot repository: %w", err)
	}
	active, err := activeTopicPath(workspace, id)
	if err == nil && !force {
		return fmt.Errorf("%w: active topic exists at %s; use --force", domain.ErrConflict, workspace.RelativePath(active))
	}
	if err != nil && !os.IsNotExist(err) && !strings.Contains(err.Error(), domain.ErrTopicMissing.Error()) {
		return err
	}
	folder := id.String() + "-"
	if err := validateRestorePath(folder); err != nil {
		return err
	}
	if force && active != "" {
		if err := os.RemoveAll(active); err != nil {
			return fmt.Errorf("remove active topic for restore: %w", err)
		}
	}
	if err := os.MkdirAll(filepath.Join(workspace.Histories, folder), 0o755); err != nil {
		return fmt.Errorf("create restore target: %w", err)
	}
	if err := repository.Restore(snapshot, folder); err != nil {
		return fmt.Errorf("restore snapshot: %w", err)
	}
	if _, err := indexer.Rebuild(workspace); err != nil {
		return fmt.Errorf("rebuild restore index: %w", err)
	}
	return nil
}

func validateRestorePath(path string) error {
	if strings.TrimSpace(path) == "" || filepath.IsAbs(path) || path == "." || strings.Contains(path, "..") || strings.ContainsAny(path, `/\\`) {
		return fmt.Errorf("%w: invalid restore path", domain.ErrConflict)
	}
	return nil
}
