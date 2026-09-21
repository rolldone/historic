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

// TopicLocation resolves one topic in either storage root and rejects duplicates.
func TopicLocation(workspace config.Workspace, id domain.ID) (string, domain.StorageState, error) {
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
		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(entry.Name(), id.String()+"-") && !strings.HasPrefix(entry.Name(), ".staging-") {
				locations = append(locations, struct {
					path    string
					storage domain.StorageState
				}{filepath.Join(candidate.root, entry.Name()), candidate.storage})
			}
		}
	}
	if len(locations) == 0 {
		return "", "", fmt.Errorf("%w: %s", domain.ErrTopicMissing, id)
	}
	if len(locations) > 1 {
		return "", "", fmt.Errorf("%w: topic %s exists in open and closed storage", domain.ErrConflict, id)
	}
	return locations[0].path, locations[0].storage, nil
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
