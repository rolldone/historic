package lifecycle

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/indexer"
	"historic/internal/markdown"
)

// DeleteChange describes a destructive filesystem operation that completed
// after its source and read model were validated.
type DeleteChange struct {
	ID        domain.ID
	Title     string
	Path      string
	Storage   domain.StorageState
	Target    string
	Permanent bool
}

// DeleteFile removes one file from the open, current topic state. Metadata and
// the rebuildable index are reconciled only after the file is staged in trash.
func (service Service) DeleteFile(input string) (DeleteChange, error) {
	path, topic, err := resolveDeleteFile(service.Workspace, input)
	if err != nil {
		return DeleteChange{}, err
	}
	return service.deleteFile(path, topic)
}

func (service Service) deleteFile(path, topic string) (DeleteChange, error) {
	if err := validateTopicPath(service.Workspace, topic); err != nil {
		return DeleteChange{}, err
	}
	id := topicIDFromPath(service.Workspace, topic)
	if !id.Valid() {
		return DeleteChange{}, fmt.Errorf("%w: unable to determine topic ID", domain.ErrConflict)
	}
	meta, err := validateCopiedTopic(topic, id)
	if err != nil {
		return DeleteChange{}, fmt.Errorf("validate current topic: %w", err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return DeleteChange{}, fmt.Errorf("inspect delete target: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return DeleteChange{}, fmt.Errorf("%w: delete target must be a regular file", domain.ErrConflict)
	}
	if filepath.Base(path) == markdown.MetaFilename || filepath.Base(path) == markdown.LegacyMetaFilename {
		return DeleteChange{}, fmt.Errorf("%w: topic metadata is protected", domain.ErrConflict)
	}
	metaPath := filepath.Join(topic, markdown.MetaFilename)
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return DeleteChange{}, fmt.Errorf("backup topic metadata: %w", err)
	}
	metaInfo, err := os.Lstat(metaPath)
	if err != nil || metaInfo.Mode()&os.ModeSymlink != 0 {
		return DeleteChange{}, fmt.Errorf("%w: topic metadata is not a regular file", domain.ErrConflict)
	}

	trash, err := uniqueTrashPath(filepath.Dir(path), ".trash-delete-file-")
	if err != nil {
		return DeleteChange{}, err
	}
	if err := os.Rename(path, trash); err != nil {
		return DeleteChange{}, fmt.Errorf("stage file in trash: %w", err)
	}
	rollback := func() error {
		var rollbackErr error
		if err := os.Rename(trash, path); err != nil {
			rollbackErr = fmt.Errorf("restore deleted file: %w", err)
		}
		if err := os.WriteFile(metaPath, metaBytes, metaInfo.Mode().Perm()); err != nil && rollbackErr == nil {
			rollbackErr = fmt.Errorf("restore topic metadata: %w", err)
		}
		return rollbackErr
	}
	if _, err := syncMetaTopic(service.Workspace, service.Workspace.RelativePath(topic), false); err != nil {
		rollbackErr := rollback()
		return DeleteChange{}, combineDeleteErrors(fmt.Errorf("reconcile topic metadata: %w", err), rollbackErr)
	}
	if _, err := indexer.Rebuild(service.Workspace); err != nil {
		rollbackErr := rollback()
		return DeleteChange{}, combineDeleteErrors(fmt.Errorf("rebuild delete index: %w", err), rollbackErr)
	}
	if err := os.Remove(trash); err != nil {
		rollbackErr := rollback()
		return DeleteChange{}, combineDeleteErrors(fmt.Errorf("remove delete trash: %w", err), rollbackErr)
	}
	return DeleteChange{ID: id, Title: meta.Frontmatter.Title, Path: service.Workspace.RelativePath(path), Storage: domain.StorageOpen, Target: "file", Permanent: true}, nil
}

// DeleteTopic removes one explicitly selected open or closed topic. If both
// copies exist, scope must be supplied so a closed snapshot is never changed
// by an operation intended for the current workdir.
func (service Service) DeleteTopic(id domain.ID, scope domain.StorageState) (DeleteChange, error) {
	return service.deleteTopic(id, scope, false)
}

// PurgeTopic permanently removes one explicitly selected topic after the CLI
// has performed its additional confirmation contract.
func (service Service) PurgeTopic(id domain.ID, scope domain.StorageState) (DeleteChange, error) {
	return service.deleteTopic(id, scope, true)
}

func (service Service) deleteTopic(id domain.ID, scope domain.StorageState, permanent bool) (DeleteChange, error) {
	if !id.Valid() {
		return DeleteChange{}, fmt.Errorf("%w: %q", domain.ErrInvalidID, id)
	}
	if scope != "" && !scope.IsValid() {
		return DeleteChange{}, fmt.Errorf("%w: invalid storage scope %q", domain.ErrConflict, scope)
	}
	location, err := resolveDeleteTopic(service.Workspace, id, scope)
	if err != nil {
		return DeleteChange{}, err
	}
	if err := validateTopicPath(service.Workspace, location.path); err != nil {
		return DeleteChange{}, err
	}
	meta, err := validateCopiedTopic(location.path, id)
	if err != nil {
		return DeleteChange{}, fmt.Errorf("validate delete topic: %w", err)
	}
	trash, err := uniqueTrashPath(filepath.Dir(location.path), ".trash-delete-topic-")
	if err != nil {
		return DeleteChange{}, err
	}
	if err := os.Rename(location.path, trash); err != nil {
		return DeleteChange{}, fmt.Errorf("stage topic in trash: %w", err)
	}
	if _, err := indexer.Rebuild(service.Workspace); err != nil {
		rollbackErr := os.Rename(trash, location.path)
		return DeleteChange{}, combineDeleteErrors(fmt.Errorf("rebuild delete index: %w", err), rollbackErr)
	}
	if permanent {
		if err := os.RemoveAll(trash); err != nil {
			return DeleteChange{}, fmt.Errorf("remove topic trash: %w", err)
		}
	}
	return DeleteChange{ID: id, Title: meta.Frontmatter.Title, Path: service.Workspace.RelativePath(location.path), Storage: location.storage, Target: "topic", Permanent: permanent}, nil
}

func resolveDeleteTopic(workspace config.Workspace, id domain.ID, scope domain.StorageState) (topicLocation, error) {
	locations := make([]topicLocation, 0, 2)
	for _, candidate := range []struct {
		root    string
		storage domain.StorageState
	}{{workspace.Histories, domain.StorageOpen}, {workspace.Database, domain.StorageClosed}} {
		entries, err := os.ReadDir(candidate.root)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return topicLocation{}, fmt.Errorf("scan %s topics: %w", candidate.storage, err)
		}
		prefix := id.String() + "-"
		for _, entry := range entries {
			if entry.Name() == ".git" || strings.HasPrefix(entry.Name(), ".staging-") || strings.HasPrefix(entry.Name(), ".trash-") || !strings.HasPrefix(entry.Name(), prefix) {
				continue
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return topicLocation{}, fmt.Errorf("%w: topic symlink %s", domain.ErrConflict, workspace.RelativePath(filepath.Join(candidate.root, entry.Name())))
			}
			if entry.IsDir() {
				locations = append(locations, topicLocation{path: filepath.Join(candidate.root, entry.Name()), storage: candidate.storage})
			}
		}
	}
	if scope != "" {
		var scoped []topicLocation
		for _, location := range locations {
			if location.storage == scope {
				scoped = append(scoped, location)
			}
		}
		if len(scoped) == 1 {
			return scoped[0], nil
		}
		if len(scoped) > 1 {
			return topicLocation{}, fmt.Errorf("%w: duplicate %s topic ID %s", domain.ErrConflict, scope, id)
		}
		return topicLocation{}, fmt.Errorf("%w: topic %s is not in %s storage", domain.ErrTopicMissing, id, scope)
	}
	if len(locations) == 0 {
		return topicLocation{}, fmt.Errorf("%w: %s", domain.ErrTopicMissing, id)
	}
	if len(locations) > 1 {
		return topicLocation{}, fmt.Errorf("%w: topic %s exists in open and closed storage; specify --open or --closed", domain.ErrConflict, id)
	}
	return locations[0], nil
}

func resolveDeleteFile(workspace config.Workspace, input string) (string, string, error) {
	value := strings.TrimSpace(input)
	if value == "" || filepath.IsAbs(value) {
		return "", "", fmt.Errorf("%w: file path must be workspace-relative", domain.ErrConflict)
	}
	normalized := filepath.ToSlash(value)
	for _, part := range strings.Split(normalized, "/") {
		if part == "" || part == "." || part == ".." {
			return "", "", fmt.Errorf("%w: invalid file path %q", domain.ErrConflict, input)
		}
		if part == ".database" || part == ".git" || strings.HasPrefix(part, ".staging-") || strings.HasPrefix(part, ".trash-") {
			return "", "", fmt.Errorf("%w: internal path is protected", domain.ErrConflict)
		}
	}
	if strings.HasPrefix(normalized, ".historic/.database/") || strings.HasPrefix(normalized, ".historic/.git/") {
		return "", "", fmt.Errorf("%w: closed and internal paths are protected by delete; open the topic first", domain.ErrConflict)
	}

	var openMatches, closedMatches []string
	for _, root := range []struct {
		path    string
		storage domain.StorageState
	}{
		{workspace.Histories, domain.StorageOpen}, {workspace.Database, domain.StorageClosed},
	} {
		matches, err := findFileMatches(workspace, root.path, root.storage, normalized)
		if err != nil {
			return "", "", err
		}
		if root.storage == domain.StorageOpen {
			openMatches = append(openMatches, matches...)
		} else {
			closedMatches = append(closedMatches, matches...)
		}
	}
	if len(openMatches) == 0 {
		if len(closedMatches) > 0 {
			return "", "", fmt.Errorf("%w: target is in closed storage; open/copy the topic before deleting", domain.ErrConflict)
		}
		return "", "", fmt.Errorf("%w: file %q", domain.ErrTopicMissing, input)
	}
	if len(openMatches) > 1 {
		return "", "", fmt.Errorf("%w: file path %q matches multiple open files", domain.ErrConflict, input)
	}
	path := openMatches[0]
	if filepath.Base(path) == markdown.MetaFilename || filepath.Base(path) == markdown.LegacyMetaFilename {
		return "", "", fmt.Errorf("%w: topic metadata is protected", domain.ErrConflict)
	}
	topic := filepath.Dir(path)
	for topic != workspace.Histories && isInside(workspace.Histories, topic) {
		if filepath.Base(topic) != "" && len(filepath.Base(topic)) >= 6 && filepath.Base(topic)[5] == '-' {
			return path, topic, nil
		}
		topic = filepath.Dir(topic)
	}
	return "", "", fmt.Errorf("%w: delete target is not inside a topic", domain.ErrConflict)
}

func findFileMatches(workspace config.Workspace, root string, storage domain.StorageState, normalized string) ([]string, error) {
	matches := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if errors.Is(walkErr, os.ErrNotExist) {
			return nil
		}
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if pathMatchesInput(workspace, path, normalized) {
				return fmt.Errorf("%w: symlink file is not a valid delete target", domain.ErrConflict)
			}
			return nil
		}
		if entry.IsDir() {
			if filepath.Base(path) == ".git" || (root == workspace.Histories && path == workspace.Database) {
				return filepath.SkipDir
			}
			return nil
		}
		if pathMatchesInput(workspace, path, normalized) {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("scan %s files: %w", storage, err)
	}
	return matches, nil
}

func pathMatchesInput(workspace config.Workspace, path, normalized string) bool {
	relative := filepath.ToSlash(workspace.RelativePath(path))
	if relative == normalized || strings.TrimPrefix(relative, ".historic/") == normalized || filepath.Base(relative) == normalized || strings.HasSuffix(relative, "/"+normalized) {
		return true
	}
	return false
}

func uniqueTrashPath(parent, prefix string) (string, error) {
	for attempt := 0; attempt < 10; attempt++ {
		candidate := filepath.Join(parent, fmt.Sprintf("%s%d-%d", prefix, os.Getpid(), time.Now().UnixNano()+int64(attempt)))
		if _, err := os.Lstat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		} else if err != nil {
			return "", fmt.Errorf("inspect trash path: %w", err)
		}
	}
	return "", fmt.Errorf("%w: unable to allocate a unique trash path", domain.ErrConflict)
}

func combineDeleteErrors(operation error, rollback error) error {
	if rollback == nil {
		return operation
	}
	return fmt.Errorf("%w; rollback failed: %v", operation, rollback)
}
