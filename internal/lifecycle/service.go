package lifecycle

import (
	"fmt"
	"path/filepath"
	"time"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/indexer"
	"historic/internal/markdown"
)

// Change describes one successful lifecycle status change.
type Change struct {
	ID              domain.ID
	Title           string
	Previous        domain.Status
	Current         domain.Status
	Path            string
	Storage         domain.StorageState
	PreviousStorage domain.StorageState
	Archived        bool
}

// Service updates topic metadata and rebuilds the source-derived index.
type Service struct {
	Workspace config.Workspace
}

// NewService creates a lifecycle service for a workspace.
func NewService(workspace config.Workspace) Service {
	return Service{Workspace: workspace}
}

// ChangeStatus changes a topic's work status without changing its storage location.
func (service Service) ChangeStatus(id domain.ID, next domain.Status) (Change, error) {
	if !id.Valid() {
		return Change{}, fmt.Errorf("%w: %q", domain.ErrInvalidID, id)
	}
	if !next.IsValid() {
		return Change{}, fmt.Errorf("%w: %q", domain.ErrInvalidStatus, next)
	}
	location, err := resolveTopic(service.Workspace, id)
	if err != nil {
		return Change{}, err
	}
	path := location.path
	meta, err := markdown.ReadTopicMetadata(path)
	if err != nil {
		return Change{}, fmt.Errorf("read topic metadata: %w", err)
	}
	if meta.ID != id {
		return Change{}, fmt.Errorf("%w: metadata ID %s does not match path ID %s", domain.ErrConflict, meta.ID, id)
	}
	if err := domain.ValidateTransition(meta.Status, next); err != nil {
		return Change{}, err
	}
	previous := meta.Status
	meta.Status = next
	meta.Updated = time.Now().UTC().Format("2006-01-02")
	if err := markdown.WriteTopicMetadata(filepath.Join(path, markdown.MetaFilename), meta); err != nil {
		return Change{}, fmt.Errorf("update topic status: %w", err)
	}
	if _, err := indexer.Rebuild(service.Workspace); err != nil {
		return Change{}, fmt.Errorf("rebuild lifecycle index: %w", err)
	}
	return Change{ID: id, Title: meta.Title, Previous: previous, Current: next, Path: service.Workspace.RelativePath(path), Storage: location.storage, PreviousStorage: location.storage, Archived: location.storage == domain.StorageClosed}, nil
}

func activeTopicPath(workspace config.Workspace, id domain.ID) (string, error) {
	path, storage, err := TopicLocation(workspace, id)
	if err != nil {
		return "", err
	}
	if storage != domain.StorageOpen {
		return "", fmt.Errorf("%w: topic %s is closed", domain.ErrTopicMissing, id)
	}
	return path, nil
}
