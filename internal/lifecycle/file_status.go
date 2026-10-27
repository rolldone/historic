package lifecycle

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/indexer"
	"historic/internal/markdown"
)

var topicPathFolderPattern = regexp.MustCompile(`^([0-9]{5}|[0-9]{13})-(.+)$`)

// FileChange describes one successful file-level status change.
type FileChange struct {
	ID          domain.ID
	Title       string
	Previous    domain.Status
	Current     domain.Status
	Path        string
	Updated     string
	ForwardedTo domain.TopicID
}

// ChangeFileStatus changes only the frontmatter of one Markdown file and
// rebuilds the source-derived index. It never archives or moves a topic.
// Switching away from the terminal forwarded status clears forwarded_to.
func ChangeFileStatus(workspace config.Workspace, inputPath string, next domain.Status) (FileChange, error) {
	return changeFileStatus(workspace, inputPath, next, "")
}

// changeFileStatus applies a status change and, when forwardedTo is set,
// records the forward destination on the same document. Callers that pass a
// non-forwarded status always clear any existing forwarded_to.
func changeFileStatus(workspace config.Workspace, inputPath string, next domain.Status, forwardedTo domain.TopicID) (FileChange, error) {
	if !next.IsValid() {
		return FileChange{}, fmt.Errorf("%w: %q", domain.ErrInvalidStatus, next)
	}
	if forwardedTo != "" && !forwardedTo.Valid() {
		return FileChange{}, fmt.Errorf("%w: forwarded_to %q", domain.ErrInvalidID, forwardedTo)
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
	previous := document.Frontmatter.Status
	updated := time.Now().UTC().Format("2006-01-02")
	document.Frontmatter.Status = next
	document.Frontmatter.Updated = updated
	switch {
	case next != domain.StatusForwarded:
		// Leaving the terminal forwarded status drops the destination.
		document.Frontmatter.ForwardedTo = ""
	case forwardedTo != "":
		document.Frontmatter.ForwardedTo = forwardedTo
	}
	if err := markdown.WriteFile(path, document); err != nil {
		return FileChange{}, fmt.Errorf("update file status: %w", err)
	}
	relative := workspace.RelativePath(path)
	manifestErr := syncFileStatusManifest(workspace, path)
	_, rebuildErr := indexer.Rebuild(workspace)
	switch {
	case manifestErr != nil && rebuildErr != nil:
		return FileChange{}, fmt.Errorf("file status partially applied for %s: frontmatter updated but topic manifest sync failed (%v) and index rebuild failed (%v); run \"historic rebuild\" to recover", relative, manifestErr, rebuildErr)
	case manifestErr != nil:
		return FileChange{}, fmt.Errorf("file status partially applied for %s: frontmatter updated but topic manifest sync failed: %w; run \"historic rebuild\" to recover", relative, manifestErr)
	case rebuildErr != nil:
		return FileChange{}, fmt.Errorf("file status partially applied for %s: frontmatter and manifest updated but index rebuild failed: %w; run \"historic rebuild\" to recover", relative, rebuildErr)
	}
	return FileChange{
		ID: document.Frontmatter.ID, Title: document.Frontmatter.Title,
		Previous: previous, Current: next, Path: relative, Updated: updated,
		ForwardedTo: document.Frontmatter.ForwardedTo,
	}, nil
}

// syncFileStatusManifest reconciles the owning topic's _meta.yaml.files entry
// so a status change is reflected without requiring a manual rebuild.
func syncFileStatusManifest(workspace config.Workspace, filePath string) error {
	topic, err := topicRootForFile(workspace, filePath)
	if err != nil {
		return err
	}
	_, err = syncMetaTopic(workspace, workspace.RelativePath(topic), false)
	return err
}

func topicRootForFile(workspace config.Workspace, filePath string) (string, error) {
	dir := filepath.Dir(filePath)
	for {
		parent := filepath.Dir(dir)
		if filepath.Clean(dir) != filepath.Clean(parent) &&
			(filepath.Clean(parent) == filepath.Clean(workspace.Histories) || filepath.Clean(parent) == filepath.Clean(workspace.Database)) {
			return dir, nil
		}
		if filepath.Clean(parent) == filepath.Clean(dir) {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("%w: file %q is not inside a topic", domain.ErrTopicMissing, workspace.RelativePath(filePath))
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
	candidates = uniquePaths(candidates)
	if len(candidates) > 1 {
		return "", fmt.Errorf("%w: file path %q matches multiple Markdown files", domain.ErrConflict, inputPath)
	}
	if filepath.Ext(candidates[0]) != ".md" {
		return "", fmt.Errorf("file status unsupported: target is a regular asset without Historic frontmatter")
	}
	return candidates[0], nil
}

// uniquePaths drops duplicate candidates that resolve to the same file. The
// archive root lives inside the active root, so an explicit .database path is
// discovered by both scans and must not be reported as an ambiguity.
func uniquePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		clean := filepath.Clean(path)
		if _, exists := seen[clean]; exists {
			continue
		}
		seen[clean] = struct{}{}
		result = append(result, path)
	}
	return result
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
		match := topicPathFolderPattern.FindStringSubmatch(part)
		if len(match) == 3 {
			if id, err := domain.ParseTopicIdentity(match[1]); err == nil {
				return id
			}
		}
	}
	return ""
}
