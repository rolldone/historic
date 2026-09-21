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
			if !strings.HasPrefix(entry.Name(), id.String()+"-") || strings.HasPrefix(entry.Name(), ".staging-") {
				continue
			}
			if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
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
	if len(locations) == 1 {
		return locations[0].path, locations[0].storage, nil
	}
	var openLocation *struct {
		path    string
		storage domain.StorageState
	}
	for index := range locations {
		if locations[index].storage == domain.StorageOpen {
			if openLocation != nil {
				return "", "", fmt.Errorf("%w: duplicate open topic ID %s", domain.ErrConflict, id)
			}
			openLocation = &locations[index]
		}
	}
	if openLocation != nil {
		return openLocation.path, openLocation.storage, nil
	}
	return "", "", fmt.Errorf("%w: duplicate closed topic ID %s", domain.ErrConflict, id)
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
