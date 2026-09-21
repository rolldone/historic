package repository

import (
	"fmt"
	"path/filepath"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/lifecycle"
)

// ImportTopic is retained for compatibility and now opens the topic in place.
func (store TopicStore) ImportTopic(id domain.ID, force bool) (domain.Topic, error) {
	if force {
		return domain.Topic{}, fmt.Errorf("%w: force import is no longer supported", domain.ErrConflict)
	}
	_, storage, err := lifecycle.TopicLocation(store.Workspace, id)
	if err != nil {
		return domain.Topic{}, err
	}
	if storage == domain.StorageOpen {
		return domain.Topic{}, fmt.Errorf("%w: topic %s is already open", domain.ErrConflict, id)
	}
	change, err := lifecycle.NewService(store.Workspace).Open(id)
	if err != nil {
		return domain.Topic{}, err
	}
	return domain.Topic{ID: id, Title: change.Title, Status: change.Current, Path: filepath.Join(store.Workspace.Root, filepath.FromSlash(change.Path))}, nil
}

func archivedTopicPath(workspace config.Workspace, id domain.ID) (string, error) {
	path, storage, err := lifecycle.TopicLocation(workspace, id)
	if err != nil || storage != domain.StorageClosed {
		if err != nil {
			return "", err
		}
		return "", fmt.Errorf("%w: archived topic %s", domain.ErrTopicMissing, id)
	}
	return path, nil
}

func validateImportTree(root string) error {
	return nil
}

func copyTree(source, destination string) error {
	return fmt.Errorf("copyTree is no longer supported")
}
