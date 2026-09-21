package lifecycle

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"historic/internal/config"
	"historic/internal/domain"
)

// TopicLocation resolves one logical topic in either storage root. When an
// open workdir and closed snapshot coexist, the open workdir is authoritative.
func TopicLocation(workspace config.Workspace, id domain.ID) (string, domain.StorageState, error) {
	if !id.Valid() {
		return "", "", fmt.Errorf("%w: %q", domain.ErrInvalidID, id)
	}
	locations := make([]struct {
		path    string
		storage domain.StorageState
	}, 0, 2)
	for _, candidate := range []struct {
		root    string
		storage domain.StorageState
	}{{workspace.Histories, domain.StorageOpen}, {workspace.Database, domain.StorageClosed}} {
		entries, err := os.ReadDir(candidate.root)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", "", err
		}
		prefix := id.String() + "-"
		for _, entry := range entries {
			if !strings.HasPrefix(entry.Name(), prefix) || strings.HasPrefix(entry.Name(), ".staging-") {
				continue
			}
			path := filepath.Join(candidate.root, entry.Name())
			info, err := os.Lstat(path)
			if err != nil {
				return "", "", err
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return "", "", fmt.Errorf("%w: topic symlink %s", domain.ErrConflict, filepath.ToSlash(path))
			}
			if !info.IsDir() {
				continue
			}
			locations = append(locations, struct {
				path    string
				storage domain.StorageState
			}{path, candidate.storage})
		}
	}
	if len(locations) == 0 {
		return "", "", fmt.Errorf("%w: %s", domain.ErrTopicMissing, id)
	}
	var openPath, closedPath string
	for _, location := range locations {
		switch location.storage {
		case domain.StorageOpen:
			if openPath != "" {
				return "", "", fmt.Errorf("%w: duplicate open topic ID %s", domain.ErrConflict, id)
			}
			openPath = location.path
		case domain.StorageClosed:
			if closedPath != "" {
				return "", "", fmt.Errorf("%w: duplicate closed topic ID %s", domain.ErrConflict, id)
			}
			closedPath = location.path
		}
	}
	if openPath != "" {
		return openPath, domain.StorageOpen, nil
	}
	return closedPath, domain.StorageClosed, nil
}

// ActiveTopicPath is the compatibility resolver for operations that require open storage.
func ActiveTopicPath(workspace config.Workspace, id domain.ID) (string, error) {
	path, storage, err := TopicLocation(workspace, id)
	if err != nil {
		return "", err
	}
	if storage != domain.StorageOpen {
		return "", fmt.Errorf("%w: topic %s is closed", domain.ErrTopicMissing, id)
	}
	return path, nil
}
