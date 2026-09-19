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

// FileChange describes one successful file-level status change.
type FileChange struct {
	ID       domain.ID
	Title    string
	Previous domain.Status
	Current  domain.Status
	Path     string
	Updated  string
}

// ChangeFileStatus changes only the frontmatter of one Markdown file and
// rebuilds the source-derived index. It never archives or moves a topic.
func ChangeFileStatus(workspace config.Workspace, inputPath string, next domain.Status) (FileChange, error) {
	if !next.IsValid() {
		return FileChange{}, fmt.Errorf("%w: %q", domain.ErrInvalidStatus, next)
	}
	path, err := resolveMarkdownPath(workspace, inputPath)
	if err != nil {
		return FileChange{}, err
	}
	document, err := markdown.ParseFile(path)
	if err != nil {
		if strings.HasSuffix(strings.ToLower(path), ".md") {
			return FileChange{}, fmt.Errorf("file status unsupported: target is a regular asset without Historic frontmatter: %w", err)
		}
		return FileChange{}, fmt.Errorf("file status unsupported: target must be a Markdown file")
	}
	if document.Frontmatter.ID != topicIDFromPath(workspace, path) {
		return FileChange{}, fmt.Errorf("file status unsupported: target frontmatter ID does not match topic")
	}
	previous := document.Frontmatter.Status
	updated := time.Now().UTC().Format("2006-01-02")
	document.Frontmatter.Status = next
	document.Frontmatter.Updated = updated
	if err := markdown.WriteFile(path, document); err != nil {
		return FileChange{}, fmt.Errorf("update file status: %w", err)
	}
	if _, err := indexer.Rebuild(workspace); err != nil {
		return FileChange{}, fmt.Errorf("rebuild file status index: %w", err)
	}
	return FileChange{
		ID: document.Frontmatter.ID, Title: document.Frontmatter.Title,
		Previous: previous, Current: next, Path: workspace.RelativePath(path), Updated: updated,
	}, nil
}

func resolveMarkdownPath(workspace config.Workspace, inputPath string) (string, error) {
	value := strings.TrimSpace(inputPath)
	if value == "" {
		return "", fmt.Errorf("%w: file path is empty", domain.ErrConflict)
	}
	normalized := filepath.ToSlash(value)
	if filepath.IsAbs(value) || strings.HasPrefix(normalized, "/") {
		return "", fmt.Errorf("%w: file path must be workspace-relative", domain.ErrConflict)
	}
	for _, part := range strings.Split(normalized, "/") {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("%w: invalid file path %q", domain.ErrConflict, inputPath)
		}
	}

	candidates := make([]string, 0, 1)
	for _, root := range []string{workspace.Histories, workspace.Database} {
		candidate := filepath.Join(workspace.Root, filepath.FromSlash(normalized))
		if isInside(root, candidate) {
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				candidates = append(candidates, candidate)
			}
		}
	}
	if len(candidates) == 0 {
		for _, root := range []string{workspace.Histories, workspace.Database} {
			err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || filepath.Ext(path) != ".md" {
					return nil
				}
				relative := filepath.ToSlash(strings.TrimPrefix(workspace.RelativePath(path), ".historic/"))
				if relative == normalized || filepath.Base(relative) == normalized || strings.HasSuffix(relative, "/"+normalized) {
					candidates = append(candidates, path)
				}
				return nil
			})
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return "", fmt.Errorf("scan Markdown files: %w", err)
			}
		}
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("%w: Markdown file %q", domain.ErrTopicMissing, inputPath)
	}
	if len(candidates) > 1 {
		return "", fmt.Errorf("%w: file path %q matches multiple Markdown files", domain.ErrConflict, inputPath)
	}
	if filepath.Ext(candidates[0]) != ".md" {
		return "", fmt.Errorf("file status unsupported: target is a regular asset without Historic frontmatter")
	}
	return candidates[0], nil
}

func isInside(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func topicIDFromPath(workspace config.Workspace, path string) domain.ID {
	relative := filepath.ToSlash(workspace.RelativePath(path))
	parts := strings.Split(relative, "/")
	for _, part := range parts {
		if len(part) > 6 && part[5] == '-' {
			if id, err := domain.ParseID(part[:5]); err == nil {
				return id
			}
		}
	}
	return ""
}
