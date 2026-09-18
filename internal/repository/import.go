package repository

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"
)

// ImportTopic copies an archived topic back to the active workspace. The
// archive remains the source of truth and is never removed by this operation.
func (store TopicStore) ImportTopic(id domain.ID, force bool) (domain.Topic, error) {
	if !id.Valid() {
		return domain.Topic{}, fmt.Errorf("%w: %q", domain.ErrInvalidID, id)
	}
	source, err := archivedTopicPath(store.Workspace, id)
	if err != nil {
		return domain.Topic{}, err
	}
	if err := validateImportTree(source); err != nil {
		return domain.Topic{}, err
	}
	metaPath := filepath.Join(source, "_meta.md")
	meta, err := markdown.ParseFile(metaPath)
	if err != nil {
		return domain.Topic{}, fmt.Errorf("read archived metadata: %w", err)
	}
	if meta.Frontmatter.ID != id {
		return domain.Topic{}, fmt.Errorf("%w: metadata ID %s does not match requested ID %s", domain.ErrConflict, meta.Frontmatter.ID, id)
	}
	destination := filepath.Join(store.Workspace.Histories, filepath.Base(source))
	if _, err := os.Lstat(destination); err == nil && !force {
		return domain.Topic{}, fmt.Errorf("%w: import destination already exists: %s", domain.ErrConflict, store.Workspace.RelativePath(destination))
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return domain.Topic{}, fmt.Errorf("inspect import destination: %w", err)
	}
	staging := filepath.Join(store.Workspace.Histories, ".staging-import-"+filepath.Base(source))
	if _, err := os.Lstat(staging); err == nil {
		return domain.Topic{}, fmt.Errorf("%w: import staging path already exists", domain.ErrConflict)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return domain.Topic{}, fmt.Errorf("inspect import staging path: %w", err)
	}
	if err := copyTree(source, staging); err != nil {
		_ = os.RemoveAll(staging)
		return domain.Topic{}, fmt.Errorf("copy archived topic: %w", err)
	}
	if meta.Frontmatter.Status.IsClose() {
		stagedMetaPath := filepath.Join(staging, "_meta.md")
		stagedMeta, err := markdown.ParseFile(stagedMetaPath)
		if err != nil {
			_ = os.RemoveAll(staging)
			return domain.Topic{}, fmt.Errorf("read staged metadata: %w", err)
		}
		stagedMeta.Frontmatter.Status = domain.StatusProgress
		stagedMeta.Frontmatter.Updated = time.Now().UTC().Format("2006-01-02")
		if err := markdown.WriteFile(stagedMetaPath, stagedMeta); err != nil {
			_ = os.RemoveAll(staging)
			return domain.Topic{}, fmt.Errorf("update imported status: %w", err)
		}
	}
	if force {
		if err := os.RemoveAll(destination); err != nil {
			_ = os.RemoveAll(staging)
			return domain.Topic{}, fmt.Errorf("replace import destination: %w", err)
		}
	}
	if err := os.Rename(staging, destination); err != nil {
		_ = os.RemoveAll(staging)
		return domain.Topic{}, fmt.Errorf("finalize import: %w", err)
	}
	return domain.Topic{ID: id, Title: meta.Frontmatter.Title, Status: domain.StatusProgress, Path: destination}, nil
}

func archivedTopicPath(workspace config.Workspace, id domain.ID) (string, error) {
	entries, err := os.ReadDir(workspace.Database)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w: %s", domain.ErrTopicMissing, id)
		}
		return "", fmt.Errorf("scan archived topics: %w", err)
	}
	prefix := id.String() + "-"
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) && !strings.HasPrefix(entry.Name(), ".staging-") {
			return filepath.Join(workspace.Database, entry.Name()), nil
		}
	}
	return "", fmt.Errorf("%w: archived topic %s", domain.ErrTopicMissing, id)
}

func validateImportTree(root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: symlink is not allowed in archive: %s", domain.ErrConflict, path)
		}
		return nil
	})
}

func copyTree(source, destination string) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(destination, info.Mode().Perm()); err != nil {
		return err
	}
	return filepath.Walk(source, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		target := filepath.Join(destination, relative)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%w: unsupported archive file %s", domain.ErrConflict, path)
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		defer input.Close()
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		output, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_EXCL, info.Mode().Perm())
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(output, input)
		closeErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}
