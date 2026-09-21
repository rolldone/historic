package repository

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"historic/internal/config"
	"historic/internal/domain"
)

type topicLocation struct {
	path    string
	storage domain.StorageState
}

func findTopicLocation(workspace config.Workspace, id domain.ID) (topicLocation, error) {
	if !id.Valid() {
		return topicLocation{}, fmt.Errorf("%w: %q", domain.ErrInvalidID, id)
	}
	locations := make([]topicLocation, 0, 2)
	for _, item := range []struct {
		root    string
		storage domain.StorageState
	}{{workspace.Histories, domain.StorageOpen}, {workspace.Database, domain.StorageClosed}} {
		entries, err := os.ReadDir(item.root)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return topicLocation{}, err
		}
		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(entry.Name(), id.String()+"-") && !strings.HasPrefix(entry.Name(), ".staging-") {
				locations = append(locations, topicLocation{filepath.Join(item.root, entry.Name()), item.storage})
			}
		}
	}
	if len(locations) == 0 {
		return topicLocation{}, fmt.Errorf("%w: %s", domain.ErrTopicMissing, id)
	}
	if len(locations) > 1 {
		return topicLocation{}, fmt.Errorf("%w: topic %s exists in open and closed storage", domain.ErrConflict, id)
	}
	return locations[0], nil
}
