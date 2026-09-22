package lifecycle

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/identifier"
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

// syncMetaTopic reconciles generated canonical metadata during rebuild and
// destructive file operations. It is intentionally not a user-facing command.
func syncMetaTopic(workspace config.Workspace, input string, rebuildIndex bool) (SyncChange, error) {
	topic, id, err := resolveSyncTopic(workspace, input)
	if err != nil {
		return SyncChange{}, err
	}
	metadata, err := markdown.ReadTopicMetadata(topic)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return SyncChange{}, fmt.Errorf("read topic metadata: %w", err)
		}
		metadata, err = recoverTopicMetadata(topic, id)
		if err != nil {
			return SyncChange{}, fmt.Errorf("recover topic metadata: %w", err)
		}
	}

	managed, assets, err := scanTopicFiles(topic, id, metadata.Files)
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

func recoverTopicMetadata(topic string, id domain.ID) (markdown.TopicMetadata, error) {
	managed, _, err := scanTopicFiles(topic, id, nil)
	if err != nil {
		return markdown.TopicMetadata{}, err
	}
	folder := filepath.Base(topic)
	match := syncTopicFolderPattern.FindStringSubmatch(folder)
	if len(match) != 3 {
		return markdown.TopicMetadata{}, fmt.Errorf("invalid topic folder %q", folder)
	}
	title := strings.ReplaceAll(match[2], "-", " ")
	if strings.TrimSpace(title) == "" {
		title = id.String()
	}
	created := time.Now().UTC().Format("2006-01-02")
	updated := ""
	for _, file := range managed {
		path := filepath.Join(topic, filepath.FromSlash(file.Path))
		document, parseErr := markdown.ParseFile(path)
		if parseErr != nil {
			continue
		}
		if document.Frontmatter.Created < created {
			created = document.Frontmatter.Created
		}
		if document.Frontmatter.Updated > updated {
			updated = document.Frontmatter.Updated
		}
	}
	metadata := markdown.TopicMetadata{ID: id, Title: title, Created: created, Updated: updated}
	if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename), metadata); err != nil {
		return markdown.TopicMetadata{}, err
	}
	return metadata, nil
}

func scanTopicFiles(topic string, id domain.ID, previous []markdown.ManifestFile) ([]markdown.ManifestFile, []markdown.ManifestAsset, error) {
	previousByPath := make(map[string]domain.FileID, len(previous))
	for _, file := range previous {
		previousByPath[file.Path] = file.ID
	}
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
			input, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			document, parseErr := markdown.Parse(path, input)
			if parseErr == nil {
				fileID := previousByPath[relative]
				if fileID == "" {
					var generateErr error
					fileID, generateErr = identifier.Default.New()
					if generateErr != nil {
						return generateErr
					}
				}
				managed = append(managed, markdown.ManifestFile{ID: fileID, Path: relative, Type: manifestFileType(relative), Status: document.Frontmatter.Status})
				return nil
			}
			if markdown.LooksLikeHistoricFile(input) {
				return fmt.Errorf("%w: invalid Historic Markdown %s: %v", domain.ErrConflict, relative, parseErr)
			}
			assets = append(assets, markdown.ManifestAsset{Path: relative, Type: manifestAssetType(path)})
			return nil
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
