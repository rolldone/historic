package lifecycle

import (
	"fmt"
	"path/filepath"

	"historic/internal/domain"
)

// MirrorAllResult reports the outcome of mirroring every open topic.
type MirrorAllResult struct {
	Succeeded   []MirrorSuccess `json:"succeeded"`
	Failed      []MirrorFailure `json:"failed"`
	Total       int             `json:"total"`
	Saved       int             `json:"saved"`
	FailedCount int             `json:"failed_count"`
}

type MirrorSuccess struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Path  string `json:"path"`
}

type MirrorFailure struct {
	ID    string `json:"id"`
	Path  string `json:"path"`
	Error string `json:"error"`
}

// MirrorAllOpen snapshots the topics discovered in one initial filesystem snapshot.
// Each target is validated against its discovered path, and failures do not stop later topics.
func (service Service) MirrorAllOpen() (MirrorAllResult, error) {
	result := MirrorAllResult{Succeeded: make([]MirrorSuccess, 0), Failed: make([]MirrorFailure, 0)}
	targets, discoveryFailures, err := openTopicTargets(service.Workspace)
	if err != nil {
		return result, err
	}
	result.Failed = make([]MirrorFailure, 0, len(discoveryFailures))
	for _, failure := range discoveryFailures {
		result.Failed = append(result.Failed, MirrorFailure{ID: failure.ID, Path: failure.Path, Error: failure.Error})
	}
	result.Total = len(targets) + len(discoveryFailures)
	for _, target := range targets {
		path, storage, locationErr := TopicLocation(service.Workspace, target.id)
		if locationErr != nil {
			result.Failed = append(result.Failed, MirrorFailure{ID: target.id.String(), Path: service.Workspace.RelativePath(target.path), Error: locationErr.Error()})
			continue
		}
		if storage != domain.StorageOpen || filepath.Clean(path) != filepath.Clean(target.path) {
			conflict := fmt.Errorf("%w: topic location changed after save discovery", domain.ErrConflict)
			result.Failed = append(result.Failed, MirrorFailure{ID: target.id.String(), Path: service.Workspace.RelativePath(target.path), Error: conflict.Error()})
			continue
		}
		change, mirrorErr := mirrorTopicAt(service.Workspace, target.id, target.path)
		if mirrorErr != nil {
			result.Failed = append(result.Failed, MirrorFailure{ID: target.id.String(), Path: service.Workspace.RelativePath(target.path), Error: mirrorErr.Error()})
			continue
		}
		result.Succeeded = append(result.Succeeded, MirrorSuccess{ID: change.ID.String(), Title: change.Title, Path: change.Path})
	}
	result.Saved = len(result.Succeeded)
	result.FailedCount = len(result.Failed)
	if result.FailedCount > 0 {
		return result, fmt.Errorf("%d topic gagal disimpan", result.FailedCount)
	}
	return result, nil
}
