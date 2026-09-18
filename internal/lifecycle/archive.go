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

// Archive moves a close-status topic from working directory to .database.
func (service Service) Archive(id domain.ID) (Change, error) {
	if !id.Valid() {
		return Change{}, fmt.Errorf("%w: %q", domain.ErrInvalidID, id)
	}
	source, err := activeTopicPath(service.Workspace, id)
	if err != nil {
		return Change{}, err
	}
	metaPath := filepath.Join(source, "_meta.md")
	meta, err := markdown.ParseFile(metaPath)
	if err != nil {
		return Change{}, fmt.Errorf("read topic metadata: %w", err)
	}
	if !meta.Frontmatter.Status.IsClose() {
		return Change{}, fmt.Errorf("%w: topic %s has open status %s", domain.ErrConflict, id, meta.Frontmatter.Status)
	}
	if err := validateTopicPath(service.Workspace, source); err != nil {
		return Change{}, err
	}
	destination := filepath.Join(service.Workspace.Database, filepath.Base(source))
	if _, err := os.Lstat(destination); err == nil {
		return Change{}, fmt.Errorf("%w: archive destination %s already exists", domain.ErrConflict, service.Workspace.RelativePath(destination))
	} else if !errors.Is(err, os.ErrNotExist) {
		return Change{}, fmt.Errorf("inspect archive destination: %w", err)
	}
	if err := os.MkdirAll(service.Workspace.Database, 0o755); err != nil {
		return Change{}, fmt.Errorf("create archive directory: %w", err)
	}
	staging := filepath.Join(service.Workspace.Database, ".staging-"+filepath.Base(source))
	if _, err := os.Lstat(staging); err == nil {
		return Change{}, fmt.Errorf("%w: archive staging path already exists", domain.ErrConflict)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Change{}, fmt.Errorf("inspect archive staging path: %w", err)
	}
	if err := os.Rename(source, staging); err != nil {
		return Change{}, fmt.Errorf("move topic to archive staging: %w", err)
	}
	if err := os.Rename(staging, destination); err != nil {
		_ = os.Rename(staging, source)
		return Change{}, fmt.Errorf("finalize topic archive: %w", err)
	}
	if _, err := indexer.Rebuild(service.Workspace); err != nil {
		_ = os.Rename(destination, source)
		return Change{}, fmt.Errorf("rebuild archive index: %w", err)
	}
	return Change{ID: id, Title: meta.Frontmatter.Title, Previous: meta.Frontmatter.Status, Current: meta.Frontmatter.Status, Path: service.Workspace.RelativePath(destination), Archived: true}, nil
}

func validateTopicPath(workspace config.Workspace, path string) error {
	root, err := filepath.EvalSymlinks(workspace.Histories)
	if err != nil {
		return fmt.Errorf("resolve histories root: %w", err)
	}
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("resolve topic path: %w", err)
	}
	relative, err := filepath.Rel(root, realPath)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%w: topic path escapes histories root", domain.ErrConflict)
	}
	return nil
}
