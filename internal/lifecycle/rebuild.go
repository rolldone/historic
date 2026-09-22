package lifecycle

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"historic/internal/config"
	"historic/internal/indexer"
	"historic/internal/markdown"
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
	count, err := indexer.RebuildWithRollbackPreparation(workspace, func() (func(), error) {
		snapshots := make(map[string][]byte, len(paths))
		for _, path := range paths {
			metaPath := filepath.Join(path, markdown.MetaFilename)
			content, readErr := os.ReadFile(metaPath)
			if readErr == nil {
				snapshots[metaPath] = content
				continue
			}
			if !os.IsNotExist(readErr) {
				return nil, readErr
			}
			snapshots[metaPath] = nil
		}
		rollback := func() {
			for path, content := range snapshots {
				if content == nil {
					_ = os.Remove(path)
					continue
				}
				_ = markdown.RestoreTopicMetadata(path, content)
			}
		}
		for _, path := range paths {
			change, syncErr := syncMetaTopic(workspace, workspace.RelativePath(path), false)
			if syncErr != nil {
				rollback()
				result.Errors++
				return nil, fmt.Errorf("reconcile metadata failed for %d topic(s): %w", result.Errors, syncErr)
			}
			if change.Updated {
				result.Updated++
			}
		}
		return rollback, nil
	})
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
