package lifecycle

import (
	"fmt"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/identifier"
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

// CloseAllResult reports the outcome of each topic in a bulk close operation.
type CloseAllResult struct {
	Succeeded   []CloseAllSuccess `json:"succeeded"`
	Failed      []CloseAllFailure `json:"failed"`
	Total       int               `json:"total"`
	Closed      int               `json:"closed"`
	FailedCount int               `json:"failed_count"`
}

// CloseAllSuccess identifies a topic moved to closed storage.
type CloseAllSuccess struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Path  string `json:"path"`
}

// CloseAllFailure describes one topic that could not be moved to closed storage.
type CloseAllFailure struct {
	ID    string `json:"id"`
	Path  string `json:"path"`
	Error string `json:"error"`
}

// Service updates topic metadata and rebuilds the source-derived index.
type Service struct {
	Workspace       config.Workspace
	FileIDGenerator identifier.Generator
}

// NewService creates a lifecycle service for a workspace.
func NewService(workspace config.Workspace) Service {
	return Service{Workspace: workspace, FileIDGenerator: identifier.Default}
}

// ChangeStatus is retained for API compatibility. Topic work status is not
// authoritative; callers must use ChangeFileStatus for managed members.
func (service Service) ChangeStatus(id domain.ID, next domain.Status) (Change, error) {
	return Change{}, fmt.Errorf("%w: topic has no work status; update a managed file", domain.ErrConflict)
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
