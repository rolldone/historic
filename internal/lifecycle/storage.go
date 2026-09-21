package lifecycle

import (
	"errors"
	"fmt"
	"io"
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
			if !strings.HasPrefix(entry.Name(), prefix) || strings.HasPrefix(entry.Name(), ".staging-") {
				continue
			}
			if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
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
	return copyOpenTopic(service.Workspace, id)
}

func copyOpenTopic(workspace config.Workspace, id domain.ID) (Change, error) {
	if !id.Valid() {
		return Change{}, fmt.Errorf("%w: %q", domain.ErrInvalidID, id)
	}
	location, err := topicLocationForOpen(workspace, id)
	if err != nil {
		return Change{}, err
	}
	if location.storage != domain.StorageClosed {
		return Change{}, fmt.Errorf("%w: topic %s is already open", domain.ErrConflict, id)
	}
	if err := validateTopicPath(workspace, location.path); err != nil {
		return Change{}, err
	}
	meta, err := validateCopiedTopic(location.path, id)
	if err != nil {
		return Change{}, fmt.Errorf("validate closed topic: %w", err)
	}

	folder := filepath.Base(location.path)
	destination := filepath.Join(workspace.Histories, folder)
	if _, err := os.Lstat(destination); err == nil {
		return Change{}, fmt.Errorf("%w: destination %s already exists", domain.ErrConflict, workspace.RelativePath(destination))
	} else if !errors.Is(err, os.ErrNotExist) {
		return Change{}, fmt.Errorf("inspect destination: %w", err)
	}
	staging := filepath.Join(workspace.Histories, ".staging-open-"+id.String())
	if _, err := os.Lstat(staging); err == nil {
		return Change{}, fmt.Errorf("%w: staging path already exists", domain.ErrConflict)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Change{}, fmt.Errorf("inspect staging path: %w", err)
	}

	if err := copyTopicTree(location.path, staging); err != nil {
		_ = os.RemoveAll(staging)
		return Change{}, fmt.Errorf("stage topic copy: %w", err)
	}
	if _, err := validateCopiedTopic(staging, id); err != nil {
		_ = os.RemoveAll(staging)
		return Change{}, fmt.Errorf("validate staged topic: %w", err)
	}
	if err := os.Rename(staging, destination); err != nil {
		_ = os.RemoveAll(staging)
		return Change{}, fmt.Errorf("install open topic: %w", err)
	}
	if _, err := indexer.Rebuild(workspace); err != nil {
		_ = os.RemoveAll(destination)
		return Change{}, fmt.Errorf("rebuild storage index: %w", err)
	}
	return Change{
		ID: id, Title: meta.Frontmatter.Title, Previous: meta.Frontmatter.Status, Current: meta.Frontmatter.Status,
		Path: workspace.RelativePath(destination), Storage: domain.StorageOpen, PreviousStorage: domain.StorageClosed,
		Archived: false,
	}, nil
}

func topicLocationForOpen(workspace config.Workspace, id domain.ID) (topicLocation, error) {
	path, storage, err := TopicLocation(workspace, id)
	if err != nil {
		return topicLocation{}, err
	}
	return topicLocation{path: path, storage: storage}, nil
}

func copyTopicTree(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: symlink %s", domain.ErrConflict, filepath.ToSlash(path))
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := destination
		if relative != "." {
			target = filepath.Join(destination, relative)
		}
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%w: unsupported topic entry %s", domain.ErrConflict, filepath.ToSlash(path))
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
		if err != nil {
			_ = input.Close()
			return err
		}
		_, copyErr := io.Copy(output, input)
		inputCloseErr := input.Close()
		outputCloseErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		if inputCloseErr != nil {
			return inputCloseErr
		}
		return outputCloseErr
	})
}

func validateCopiedTopic(path string, id domain.ID) (markdown.Document, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return markdown.Document{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return markdown.Document{}, fmt.Errorf("%w: topic root is not a directory", domain.ErrConflict)
	}
	var metadata markdown.Document
	metaFound := false
	err = filepath.WalkDir(path, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		entryInfo, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if entryInfo.Mode()&os.ModeSymlink != 0 || entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: symlink %s", domain.ErrConflict, current)
		}
		if entryInfo.IsDir() {
			return nil
		}
		if !entryInfo.Mode().IsRegular() {
			return fmt.Errorf("%w: unsupported topic entry %s", domain.ErrConflict, current)
		}
		if filepath.Base(current) != "_meta.md" || current != filepath.Join(path, "_meta.md") {
			return nil
		}
		metadata, err = markdown.ParseFile(current)
		if err != nil {
			return err
		}
		if metadata.Frontmatter.ID != id {
			return fmt.Errorf("%w: metadata ID %s does not match path ID %s", domain.ErrConflict, metadata.Frontmatter.ID, id)
		}
		metaFound = true
		return nil
	})
	if err != nil {
		return markdown.Document{}, err
	}
	if !metaFound {
		return markdown.Document{}, fmt.Errorf("%w: topic metadata is missing", domain.ErrConflict)
	}
	return metadata, nil
}
