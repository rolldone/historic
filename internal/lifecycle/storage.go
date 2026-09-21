package lifecycle

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/indexer"
	"historic/internal/markdown"
)

type topicLocation struct {
	path    string
	storage domain.StorageState
}

func resolveTopic(workspace config.Workspace, id domain.ID) (topicLocation, error) {
	locations := make([]topicLocation, 0, 2)
	for _, candidate := range []struct {
		root    string
		storage domain.StorageState
	}{
		{workspace.Histories, domain.StorageOpen},
		{workspace.Database, domain.StorageClosed},
	} {
		entries, err := os.ReadDir(candidate.root)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return topicLocation{}, fmt.Errorf("scan %s topics: %w", candidate.storage, err)
		}
		prefix := id.String() + "-"
		for _, entry := range entries {
			if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".staging-") && strings.HasPrefix(entry.Name(), prefix) {
				locations = append(locations, topicLocation{path: filepath.Join(candidate.root, entry.Name()), storage: candidate.storage})
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

func moveTopic(workspace config.Workspace, id domain.ID, target domain.StorageState) (Change, error) {
	if !id.Valid() {
		return Change{}, fmt.Errorf("%w: %q", domain.ErrInvalidID, id)
	}
	location, err := resolveTopic(workspace, id)
	if err != nil {
		return Change{}, err
	}
	if location.storage == target {
		return Change{}, fmt.Errorf("%w: topic %s is already %s", domain.ErrConflict, id, target)
	}
	if err := validateTopicPath(workspace, location.path); err != nil {
		return Change{}, err
	}
	info, err := os.Lstat(location.path)
	if err != nil {
		return Change{}, fmt.Errorf("inspect topic source: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return Change{}, fmt.Errorf("%w: topic source is a symlink", domain.ErrConflict)
	}
	meta, err := markdown.ParseFile(filepath.Join(location.path, "_meta.md"))
	if err != nil {
		return Change{}, fmt.Errorf("read topic metadata: %w", err)
	}
	if meta.Frontmatter.ID != id {
		return Change{}, fmt.Errorf("%w: metadata ID %s does not match path ID %s", domain.ErrConflict, meta.Frontmatter.ID, id)
	}
	destinationRoot := workspace.Histories
	if target == domain.StorageClosed {
		destinationRoot = workspace.Database
	}
	destination := filepath.Join(destinationRoot, filepath.Base(location.path))
	if _, err := os.Lstat(destination); err == nil {
		return Change{}, fmt.Errorf("%w: destination %s already exists", domain.ErrConflict, workspace.RelativePath(destination))
	} else if !errors.Is(err, os.ErrNotExist) {
		return Change{}, fmt.Errorf("inspect destination: %w", err)
	}
	if err := os.MkdirAll(destinationRoot, 0o755); err != nil {
		return Change{}, fmt.Errorf("create destination root: %w", err)
	}
	staging := filepath.Join(destinationRoot, ".staging-"+filepath.Base(location.path))
	if _, err := os.Lstat(staging); err == nil {
		return Change{}, fmt.Errorf("%w: staging path already exists", domain.ErrConflict)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Change{}, fmt.Errorf("inspect staging path: %w", err)
	}
	if err := os.Rename(location.path, staging); err != nil {
		return Change{}, fmt.Errorf("stage topic move: %w", err)
	}
	if err := os.Rename(staging, destination); err != nil {
		_ = os.Rename(staging, location.path)
		return Change{}, fmt.Errorf("finalize topic move: %w", err)
	}
	if _, err := indexer.Rebuild(workspace); err != nil {
		_ = os.Rename(destination, location.path)
		return Change{}, fmt.Errorf("rebuild storage index: %w", err)
	}
	return Change{ID: id, Title: meta.Frontmatter.Title, Previous: meta.Frontmatter.Status, Current: meta.Frontmatter.Status, Path: workspace.RelativePath(destination), Storage: target, PreviousStorage: location.storage, Archived: target == domain.StorageClosed}, nil
}

func (service Service) Close(id domain.ID) (Change, error) {
	return moveTopic(service.Workspace, id, domain.StorageClosed)
}

func (service Service) Open(id domain.ID) (Change, error) {
	return moveTopic(service.Workspace, id, domain.StorageOpen)
}
