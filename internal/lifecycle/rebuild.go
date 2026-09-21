package lifecycle

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/indexer"
)

type RebuildChange struct {
	Topics  int
	Updated int
	Errors  int
	Records int
}

// RebuildMetadata reconciles every open and closed topic, then rebuilds the index once.
func RebuildMetadata(workspace config.Workspace) (RebuildChange, error) {
	paths, err := allTopicPaths(workspace)
	if err != nil {
		return RebuildChange{}, err
	}
	result := RebuildChange{Topics: len(paths)}
	for _, path := range paths {
		change, syncErr := syncMetaTopic(workspace, workspace.RelativePath(path), false)
		if syncErr != nil {
			result.Errors++
			continue
		}
		if change.Updated {
			result.Updated++
		}
	}
	if result.Errors > 0 {
		return result, fmt.Errorf("reconcile metadata failed for %d topic(s)", result.Errors)
	}
	count, err := indexer.Rebuild(workspace)
	if err != nil {
		return result, fmt.Errorf("rebuild unified index: %w", err)
	}
	result.Records = count
	return result, nil
}

func allTopicPaths(workspace config.Workspace) ([]string, error) {
	paths := make([]string, 0)
	for _, root := range []string{workspace.Histories, workspace.Database} {
		entries, err := os.ReadDir(root)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() && syncTopicFolderPattern.MatchString(entry.Name()) {
				paths = append(paths, filepath.Join(root, entry.Name()))
			}
		}
	}
	sort.Strings(paths)
	return paths, nil
}

var _ = domain.StorageOpen
