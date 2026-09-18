package lifecycle

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/indexer"
	"historic/internal/markdown"
)

// Change describes one successful lifecycle status change.
type Change struct {
	ID       domain.ID
	Title    string
	Previous domain.Status
	Current  domain.Status
	Path     string
	Archived bool
}

// Service updates topic metadata and rebuilds the source-derived index.
type Service struct {
	Workspace config.Workspace
}

// NewService creates a lifecycle service for a workspace.
func NewService(workspace config.Workspace) Service {
	return Service{Workspace: workspace}
}

// ChangeStatus changes an active topic's status. Close statuses are archived
// in the service but archive path movement is isolated for WO 12.
func (service Service) ChangeStatus(id domain.ID, next domain.Status) (Change, error) {
	if !id.Valid() {
		return Change{}, fmt.Errorf("%w: %q", domain.ErrInvalidID, id)
	}
	if !next.IsValid() {
		return Change{}, fmt.Errorf("%w: %q", domain.ErrInvalidStatus, next)
	}
	path, err := activeTopicPath(service.Workspace, id)
	if err != nil {
		return Change{}, err
	}
	metaPath := filepath.Join(path, "_meta.md")
	document, err := markdown.ParseFile(metaPath)
	if err != nil {
		return Change{}, fmt.Errorf("read topic metadata: %w", err)
	}
	if document.Frontmatter.ID != id {
		return Change{}, fmt.Errorf("%w: metadata ID %s does not match path ID %s", domain.ErrConflict, document.Frontmatter.ID, id)
	}
	if err := domain.ValidateTransition(document.Frontmatter.Status, next); err != nil {
		return Change{}, err
	}
	previous := document.Frontmatter.Status
	document.Frontmatter.Status = next
	document.Frontmatter.Updated = time.Now().UTC().Format("2006-01-02")
	if err := markdown.WriteFile(metaPath, document); err != nil {
		return Change{}, fmt.Errorf("update topic status: %w", err)
	}
	if _, err := indexer.Rebuild(service.Workspace); err != nil {
		return Change{}, fmt.Errorf("rebuild lifecycle index: %w", err)
	}
	return Change{ID: id, Title: document.Frontmatter.Title, Previous: previous, Current: next, Path: service.Workspace.RelativePath(path), Archived: next.IsClose()}, nil
}

func activeTopicPath(workspace config.Workspace, id domain.ID) (string, error) {
	entries, err := os.ReadDir(workspace.Histories)
	if errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("%w: %s", domain.ErrTopicMissing, id)
	}
	if err != nil {
		return "", fmt.Errorf("scan active topics: %w", err)
	}
	prefix := id.String() + "-"
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) {
			return filepath.Join(workspace.Histories, entry.Name()), nil
		}
	}
	return "", fmt.Errorf("%w: %s", domain.ErrTopicMissing, id)
}
