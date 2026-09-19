package lifecycle

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"historic/internal/config"
	"historic/internal/domain"
)

// BatchItem is the result for one active topic in a batch synchronization.
type BatchItem struct {
	ID      string `json:"id"`
	Path    string `json:"path"`
	Files   int    `json:"files"`
	Assets  int    `json:"assets"`
	Updated bool   `json:"updated"`
	Error   string `json:"error,omitempty"`
}

// BatchChange summarizes batch synchronization across active topics.
type BatchChange struct {
	Topics  []BatchItem
	Total   int
	Updated int
	Errors  int
}

// SyncMetaBatch synchronizes all active topics in deterministic path order and
// continues after individual topic errors.
func SyncMetaBatch(workspace config.Workspace) BatchChange {
	paths, err := activeTopicPaths(workspace)
	if err != nil {
		return BatchChange{Errors: 1, Topics: []BatchItem{{Error: err.Error()}}}
	}
	change := BatchChange{Topics: make([]BatchItem, 0, len(paths)), Total: len(paths)}
	for _, path := range paths {
		id := topicIDFromFolder(path)
		item := BatchItem{ID: id.String(), Path: workspace.RelativePath(path)}
		result, syncErr := SyncMeta(workspace, item.Path)
		if syncErr != nil {
			item.Error = syncErr.Error()
			change.Errors++
		} else {
			item.Files, item.Assets, item.Updated = result.Files, result.Assets, result.Updated
			if item.Updated {
				change.Updated++
			}
		}
		change.Topics = append(change.Topics, item)
	}
	return change
}

func activeTopicPaths(workspace config.Workspace) ([]string, error) {
	entries, err := os.ReadDir(workspace.Histories)
	if errors.Is(err, os.ErrNotExist) {
		return []string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("enumerate active topics: %w", err)
	}
	paths := make([]string, 0)
	for _, entry := range entries {
		if entry.Name() == config.DatabaseDirName || entry.Name() == config.IndexFileName || !entry.IsDir() {
			continue
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("%w: active topic symlink %s", domain.ErrConflict, entry.Name())
		}
		if syncTopicFolderPattern.MatchString(entry.Name()) {
			paths = append(paths, filepath.Join(workspace.Histories, entry.Name()))
		}
	}
	sort.Slice(paths, func(i, j int) bool {
		left, right := filepath.Base(paths[i]), filepath.Base(paths[j])
		return left < right
	})
	return paths, nil
}

func topicIDFromFolder(path string) domain.ID {
	base := filepath.Base(path)
	if len(base) >= 6 && strings.HasPrefix(base[5:], "-") {
		if id, err := domain.ParseID(base[:5]); err == nil {
			return id
		}
	}
	return ""
}
