package lifecycle

import (
	"fmt"
	"path/filepath"
	"time"

	"historic/internal/domain"
	"historic/internal/markdown"
)

// ArchiveStatus validates a close transition, persists it, then moves the
// topic into the internal database without exposing a partial archive.
func (service Service) ArchiveStatus(id domain.ID, next domain.Status) (Change, error) {
	if !next.IsClose() {
		return Change{}, fmt.Errorf("%w: archive requires close status %s", domain.ErrInvalidStatus, next)
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
	if err := domain.ValidateTransition(document.Frontmatter.Status, next); err != nil {
		return Change{}, err
	}
	previous := document.Frontmatter.Status
	document.Frontmatter.Status = next
	document.Frontmatter.Updated = time.Now().UTC().Format("2006-01-02")
	if err := markdown.WriteFile(metaPath, document); err != nil {
		return Change{}, fmt.Errorf("update topic status: %w", err)
	}
	change, err := service.Archive(id)
	if err != nil {
		return Change{}, err
	}
	change.Previous = previous
	change.Current = next
	return change, nil
}
