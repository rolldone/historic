package lifecycle

import (
	"crypto/sha256"
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
	path, storage, err := TopicLocation(workspace, id)
	if err != nil {
		return topicLocation{}, err
	}
	return topicLocation{path: path, storage: storage}, nil
}

func (service Service) Close(id domain.ID) (Change, error) {
	return closeTopic(service.Workspace, id)
}

func closeTopic(workspace config.Workspace, id domain.ID) (Change, error) {
	if !id.Valid() {
		return Change{}, fmt.Errorf("%w: %q", domain.ErrInvalidID, id)
	}
	path, storage, err := TopicLocation(workspace, id)
	if err != nil {
		return Change{}, err
	}
	location := topicLocation{path: path, storage: storage}
	if location.storage != domain.StorageOpen {
		return Change{}, fmt.Errorf("%w: topic %s is already closed", domain.ErrConflict, id)
	}
	if err := validateTopicPath(workspace, location.path); err != nil {
		return Change{}, err
	}
	meta, err := validateCopiedTopic(location.path, id)
	if err != nil {
		return Change{}, fmt.Errorf("validate open topic: %w", err)
	}
	sourceManifest, err := buildTopicManifest(location.path)
	if err != nil {
		return Change{}, fmt.Errorf("build close manifest: %w", err)
	}
	archive, err := findClosedTopic(workspace, id)
	if err != nil {
		return Change{}, err
	}
	var archiveManifest topicManifest
	if archive.path != "" {
		if err := validateTopicPath(workspace, archive.path); err != nil {
			return Change{}, err
		}
		if _, err := validateCopiedTopic(archive.path, id); err != nil {
			return Change{}, fmt.Errorf("validate existing archive: %w", err)
		}
		archiveManifest, err = buildTopicManifest(archive.path)
		if err != nil {
			return Change{}, fmt.Errorf("build archive manifest: %w", err)
		}
	}

	folder := filepath.Base(location.path)
	sameSnapshot := archive.path != "" && filepath.Base(archive.path) == folder && manifestsEqual(sourceManifest, archiveManifest)
	if !sameSnapshot {
		if err := stageDifferentialArchive(workspace, location.path, archive.path, folder, sourceManifest, archiveManifest); err != nil {
			return Change{}, err
		}
	}
	if err := finalizeClosedTopic(workspace, location.path, archive.path, folder, sameSnapshot); err != nil {
		return Change{}, err
	}
	return Change{ID: id, Title: meta.Frontmatter.Title, Previous: meta.Frontmatter.Status, Current: meta.Frontmatter.Status, Path: workspace.RelativePath(filepath.Join(workspace.Database, folder)), Storage: domain.StorageClosed, PreviousStorage: domain.StorageOpen, Archived: true}, nil
}

func findClosedTopic(workspace config.Workspace, id domain.ID) (topicLocation, error) {
	entries, err := os.ReadDir(workspace.Database)
	if errors.Is(err, os.ErrNotExist) {
		return topicLocation{}, nil
	}
	if err != nil {
		return topicLocation{}, fmt.Errorf("scan closed topics: %w", err)
	}
	prefix := id.String() + "-"
	var found topicLocation
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), prefix) || strings.HasPrefix(entry.Name(), ".staging-") || entry.Name() == ".git" {
			continue
		}
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			if found.path != "" {
				return topicLocation{}, fmt.Errorf("%w: duplicate closed topic ID %s", domain.ErrConflict, id)
			}
			found = topicLocation{path: filepath.Join(workspace.Database, entry.Name()), storage: domain.StorageClosed}
		}
	}
	return found, nil
}

// topicManifest contains content hashes keyed by logical POSIX paths.
type topicManifest map[string][sha256.Size]byte

func buildTopicManifest(root string) (topicManifest, error) {
	manifest := make(topicManifest)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relative == ".git" || strings.HasPrefix(relative, ".git"+string(filepath.Separator)) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: symlink %s", domain.ErrConflict, filepath.ToSlash(path))
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%w: unsupported topic entry %s", domain.ErrConflict, filepath.ToSlash(path))
		}
		hash, err := hashTopicFile(path)
		if err != nil {
			return err
		}
		manifest[filepath.ToSlash(relative)] = hash
		return nil
	})
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

func hashTopicFile(path string) ([sha256.Size]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return [sha256.Size]byte{}, err
	}
	var result [sha256.Size]byte
	copy(result[:], hash.Sum(nil))
	return result, nil
}

func manifestsEqual(left, right topicManifest) bool {
	if len(left) != len(right) {
		return false
	}
	for path, hash := range left {
		if other, ok := right[path]; !ok || other != hash {
			return false
		}
	}
	return true
}

func stageDifferentialArchive(workspace config.Workspace, source, archive, folder string, sourceManifest, archiveManifest topicManifest) error {
	staging := filepath.Join(workspace.Database, ".staging-close-"+folder)
	if _, err := os.Lstat(staging); err == nil {
		return fmt.Errorf("%w: close staging path already exists", domain.ErrConflict)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect close staging path: %w", err)
	}
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return fmt.Errorf("create close staging: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(staging) }
	for logicalPath, hash := range sourceManifest {
		target := filepath.Join(staging, filepath.FromSlash(logicalPath))
		archiveHash, unchanged := archiveManifest[logicalPath]
		if archive != "" && unchanged && archiveHash == hash {
			oldPath := filepath.Join(archive, filepath.FromSlash(logicalPath))
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				cleanup()
				return fmt.Errorf("create unchanged file directory: %w", err)
			}
			if err := os.Link(oldPath, target); err != nil {
				cleanup()
				return fmt.Errorf("stage unchanged file %s: %w", logicalPath, err)
			}
			continue
		}
		if err := copyTopicFile(filepath.Join(source, filepath.FromSlash(logicalPath)), target); err != nil {
			cleanup()
			return fmt.Errorf("stage changed file %s: %w", logicalPath, err)
		}
	}
	stagedManifest, err := buildTopicManifest(staging)
	if err != nil {
		cleanup()
		return fmt.Errorf("validate close staging manifest: %w", err)
	}
	if !manifestsEqual(sourceManifest, stagedManifest) {
		cleanup()
		return fmt.Errorf("%w: close source changed during staging", domain.ErrConflict)
	}
	return nil
}

func copyTopicFile(source, destination string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("%w: invalid topic file", domain.ErrConflict)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
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
}

func finalizeClosedTopic(workspace config.Workspace, source, archive, folder string, sameSnapshot bool) error {
	staging := filepath.Join(workspace.Database, ".staging-close-"+folder)
	destination := filepath.Join(workspace.Database, folder)
	backup := filepath.Join(workspace.Database, ".staging-close-backup-"+folder)
	sourceBackup := filepath.Join(workspace.Histories, ".staging-close-source-"+folder)
	if _, err := os.Lstat(sourceBackup); err == nil {
		if !sameSnapshot {
			_ = os.RemoveAll(staging)
		}
		return fmt.Errorf("%w: close source staging path already exists", domain.ErrConflict)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect close source staging path: %w", err)
	}
	oldMoved := false
	if !sameSnapshot && archive != "" {
		if err := os.Rename(archive, backup); err != nil {
			_ = os.RemoveAll(staging)
			return fmt.Errorf("stage old archive: %w", err)
		}
		oldMoved = true
	}
	if !sameSnapshot {
		if err := os.Rename(staging, destination); err != nil {
			if oldMoved {
				_ = os.Rename(backup, archive)
			}
			_ = os.RemoveAll(staging)
			return fmt.Errorf("install closed archive: %w", err)
		}
	}
	rollbackArchive := func() {
		if !sameSnapshot {
			_ = os.RemoveAll(destination)
			if oldMoved {
				_ = os.Rename(backup, archive)
			}
		}
	}
	if err := os.Rename(source, sourceBackup); err != nil {
		rollbackArchive()
		return fmt.Errorf("stage open workdir removal: %w", err)
	}
	if _, err := indexer.Rebuild(workspace); err != nil {
		_ = os.Rename(sourceBackup, source)
		rollbackArchive()
		return fmt.Errorf("rebuild storage index: %w", err)
	}
	if err := os.RemoveAll(sourceBackup); err != nil {
		_ = os.Rename(sourceBackup, source)
		rollbackArchive()
		return fmt.Errorf("remove closed workdir: %w", err)
	}
	if oldMoved {
		if err := os.RemoveAll(backup); err != nil {
			return fmt.Errorf("remove replaced archive: %w", err)
		}
	}
	return nil
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
	metaPath := filepath.Join(path, markdown.MetaFilename)
	metaInfo, err := os.Lstat(metaPath)
	if errors.Is(err, os.ErrNotExist) {
		metaPath = filepath.Join(path, markdown.LegacyMetaFilename)
		metaInfo, err = os.Lstat(metaPath)
	}
	if err != nil {
		return markdown.Document{}, err
	}
	if metaInfo.Mode()&os.ModeSymlink != 0 || !metaInfo.Mode().IsRegular() {
		return markdown.Document{}, fmt.Errorf("%w: topic metadata is not a regular file", domain.ErrConflict)
	}
	metadata, err := markdown.ReadTopicMetadata(path)
	if err != nil {
		return markdown.Document{}, err
	}
	if metadata.ID != id {
		return markdown.Document{}, fmt.Errorf("%w: metadata ID %s does not match path ID %s", domain.ErrConflict, metadata.ID, id)
	}
	return markdown.Document{Frontmatter: metadata.Frontmatter()}, nil
}
