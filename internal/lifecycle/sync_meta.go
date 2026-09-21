package lifecycle

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/indexer"
	"historic/internal/markdown"
)

var syncTopicFolderPattern = regexp.MustCompile(`^([0-9]{5})-(.+)$`)

// SyncChange describes one successful metadata synchronization.
type SyncChange struct {
	ID      domain.ID
	Path    string
	Files   int
	Assets  int
	Updated bool
}

// SyncMeta scans a topic and reconciles the filesystem-derived Files and
// Assets manifest in canonical YAML metadata.
func SyncMeta(workspace config.Workspace, input string) (SyncChange, error) {
	return syncMetaTopic(workspace, input, true)
}

func syncMetaTopic(workspace config.Workspace, input string, rebuildIndex bool) (SyncChange, error) {
	topic, id, err := resolveSyncTopic(workspace, input)
	if err != nil {
		return SyncChange{}, err
	}
	metadata, err := markdown.ReadTopicMetadata(topic)
	if err != nil {
		return SyncChange{}, fmt.Errorf("read topic metadata: %w", err)
	}

	managed, assets, err := scanTopicFiles(topic, id)
	if err != nil {
		return SyncChange{}, err
	}
	updated, err := markdown.WriteTopicMetadataManifest(filepath.Join(topic, markdown.MetaFilename), metadata, managed, assets)
	if err != nil {
		return SyncChange{}, fmt.Errorf("write topic manifest: %w", err)
	}
	if rebuildIndex {
		if _, err := indexer.Rebuild(workspace); err != nil {
			return SyncChange{}, fmt.Errorf("rebuild metadata index: %w", err)
		}
	}
	return SyncChange{ID: id, Path: workspace.RelativePath(topic), Files: len(managed), Assets: len(assets), Updated: updated}, nil
}

func scanTopicFiles(topic string, id domain.ID) ([]markdown.ManifestFile, []markdown.ManifestAsset, error) {
	managed := make([]markdown.ManifestFile, 0)
	assets := make([]markdown.ManifestAsset, 0)
	err := filepath.WalkDir(topic, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: symlink %s", domain.ErrConflict, path)
		}
		if entry.IsDir() || path == filepath.Join(topic, markdown.MetaFilename) {
			return nil
		}
		relative, err := filepath.Rel(topic, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if strings.EqualFold(filepath.Ext(path), ".md") && filepath.Base(path) != "_meta.md" {
			document, parseErr := markdown.ParseFile(path)
			if parseErr == nil {
				managed = append(managed, markdown.ManifestFile{Path: relative, Type: manifestFileType(relative), Status: document.Frontmatter.Status})
				return nil
			}
		}
		assets = append(assets, markdown.ManifestAsset{Path: relative, Type: manifestAssetType(path)})
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("scan topic files: %w", err)
	}
	sort.Slice(managed, func(i, j int) bool { return managed[i].Path < managed[j].Path })
	sort.Slice(assets, func(i, j int) bool { return assets[i].Path < assets[j].Path })
	return managed, assets, nil
}

func manifestFileType(path string) string {
	base := strings.ToLower(filepath.Base(path))
	name := strings.TrimSuffix(base, filepath.Ext(base))
	for _, kind := range []string{"prd", "spec", "issue", "note", "decision", "task"} {
		if name == kind || strings.HasPrefix(name, kind+"-") {
			return kind
		}
	}
	if strings.Contains(filepath.ToSlash(path), "/wos/") || strings.HasPrefix(filepath.ToSlash(path), "wos/") {
		return "task"
	}
	return "file"
}

func manifestAssetType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".md" || ext == ".markdown" {
		return "markdown"
	}
	if ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" || ext == ".webp" {
		return "image"
	}
	if ext == ".pdf" {
		return "pdf"
	}
	if ext == "" {
		return "file"
	}
	return strings.TrimPrefix(ext, ".")
}

func syncMetaSection(body, heading string, paths []string) string {
	lines := strings.SplitAfter(body, "\n")
	start := -1
	for index, line := range lines {
		if strings.TrimSpace(line) == heading {
			start = index
			break
		}
	}
	generated := heading + "\n\n"
	for _, path := range paths {
		generated += metaLink(path)
	}
	generated += "\n"
	if start < 0 {
		insert := len(lines)
		for index, line := range lines {
			if strings.TrimSpace(line) == "## Progress" {
				insert = index
				break
			}
		}
		lines = append(lines, "")
		copy(lines[insert+1:], lines[insert:])
		lines[insert] = generated
		return strings.Join(lines, "")
	}
	end := start + 1
	for end < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[end]), "## ") {
		end++
	}
	lines = append(lines[:start], append([]string{generated}, lines[end:]...)...)
	return strings.Join(lines, "")
}

func metaLink(path string) string {
	path = filepath.ToSlash(path)
	return fmt.Sprintf("- [%s](./%s)\n", filepath.Base(path), path)
}

func resolveSyncTopic(workspace config.Workspace, input string) (string, domain.ID, error) {
	value := strings.TrimSpace(input)
	if value == "" {
		return "", "", fmt.Errorf("%w: topic is empty", domain.ErrConflict)
	}
	if id, err := domain.ParseID(value); err == nil {
		matches := make([]string, 0, 2)
		for _, root := range []string{workspace.Histories, workspace.Database} {
			entries, readErr := os.ReadDir(root)
			if errors.Is(readErr, os.ErrNotExist) {
				continue
			}
			if readErr != nil {
				return "", "", readErr
			}
			for _, entry := range entries {
				if entry.Type()&os.ModeSymlink != 0 && strings.HasPrefix(entry.Name(), id.String()+"-") {
					return "", "", fmt.Errorf("%w: topic symlink %s", domain.ErrConflict, entry.Name())
				}
				if entry.IsDir() && strings.HasPrefix(entry.Name(), id.String()+"-") {
					matches = append(matches, filepath.Join(root, entry.Name()))
				}
			}
		}
		if len(matches) == 0 {
			return "", "", fmt.Errorf("%w: %s", domain.ErrTopicMissing, id)
		}
		if len(matches) > 1 {
			return "", "", fmt.Errorf("%w: topic %s exists in active and archive roots", domain.ErrConflict, id)
		}
		return matches[0], id, nil
	}
	if filepath.IsAbs(value) || strings.HasPrefix(filepath.ToSlash(value), "/") {
		return "", "", fmt.Errorf("%w: topic path must be workspace-relative", domain.ErrConflict)
	}
	for _, part := range strings.Split(filepath.ToSlash(value), "/") {
		if part == "" || part == "." || part == ".." {
			return "", "", fmt.Errorf("%w: invalid topic path %q", domain.ErrConflict, input)
		}
	}
	candidates := []string{filepath.Join(workspace.Root, filepath.FromSlash(value))}
	if !strings.HasPrefix(filepath.ToSlash(value), ".historic/") {
		candidates = append(candidates,
			filepath.Join(workspace.Histories, filepath.FromSlash(value)),
			filepath.Join(workspace.Database, filepath.FromSlash(value)))
	}
	matches := make([]string, 0, 1)
	for _, candidate := range candidates {
		if !isInside(workspace.Histories, candidate) && !isInside(workspace.Database, candidate) {
			continue
		}
		info, statErr := os.Lstat(candidate)
		if statErr == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return "", "", fmt.Errorf("%w: topic path is a symlink", domain.ErrConflict)
			}
			if info.IsDir() {
				matches = append(matches, candidate)
			}
		}
	}
	if len(matches) != 1 {
		return "", "", fmt.Errorf("%w: topic path %q", domain.ErrTopicMissing, input)
	}
	match := syncTopicFolderPattern.FindStringSubmatch(filepath.Base(matches[0]))
	if len(match) != 3 {
		return "", "", fmt.Errorf("%w: invalid topic path %q", domain.ErrConflict, input)
	}
	id, err := domain.ParseID(match[1])
	if err != nil {
		return "", "", err
	}
	return matches[0], id, nil
}
