package lifecycle

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

// SyncMeta scans a topic and updates only its _meta.md Files and Assets sections.
func SyncMeta(workspace config.Workspace, input string) (SyncChange, error) {
	topic, id, err := resolveSyncTopic(workspace, input)
	if err != nil {
		return SyncChange{}, err
	}
	metaPath := filepath.Join(topic, "_meta.md")
	meta, err := markdown.ParseFile(metaPath)
	if err != nil {
		return SyncChange{}, fmt.Errorf("read topic metadata: %w", err)
	}

	managed, assets, err := scanTopicFiles(topic, id)
	if err != nil {
		return SyncChange{}, err
	}
	body := meta.Body
	body = syncMetaSection(body, "## Files", managed)
	body = syncMetaSection(body, "## Assets", assets)
	changed := body != meta.Body
	if changed {
		meta.Body = body
		if err := markdown.WriteFile(metaPath, meta); err != nil {
			return SyncChange{}, fmt.Errorf("write topic metadata: %w", err)
		}
	}
	if _, err := indexer.Rebuild(workspace); err != nil {
		return SyncChange{}, fmt.Errorf("rebuild metadata index: %w", err)
	}
	return SyncChange{ID: id, Path: workspace.RelativePath(topic), Files: len(managed), Assets: len(assets), Updated: changed}, nil
}

func scanTopicFiles(topic string, id domain.ID) ([]string, []string, error) {
	managed := make([]string, 0)
	assets := make([]string, 0)
	err := filepath.WalkDir(topic, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: symlink %s", domain.ErrConflict, path)
		}
		if entry.IsDir() || filepath.Base(path) == "_meta.md" {
			return nil
		}
		relative, err := filepath.Rel(topic, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if strings.EqualFold(filepath.Ext(path), ".md") {
			document, parseErr := markdown.ParseFile(path)
			if parseErr == nil && document.Frontmatter.ID == id {
				managed = append(managed, relative)
				return nil
			}
		}
		assets = append(assets, relative)
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("scan topic files: %w", err)
	}
	return managed, assets, nil
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
	if start < 0 {
		insert := len(lines)
		for index, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "## ") && strings.TrimSpace(line) == "## Progress" {
				insert = index
				break
			}
		}
		block := heading + "\n\n"
		for _, path := range paths {
			block += metaLink(path)
		}
		block += "\n"
		lines = append(lines, "")
		copy(lines[insert+1:], lines[insert:])
		lines[insert] = block
		return strings.Join(lines, "")
	}
	end := start + 1
	for end < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[end]), "## ") {
		end++
	}
	existing := strings.Join(lines[start+1:end], "")
	seen := linkTargets(existing)
	addition := ""
	for _, path := range paths {
		href := "./" + filepath.ToSlash(path)
		if _, ok := seen[href]; ok {
			continue
		}
		addition += metaLink(path)
		seen[href] = struct{}{}
	}
	if addition == "" {
		return body
	}
	if existing != "" && !strings.HasSuffix(existing, "\n") {
		existing += "\n"
	}
	block := existing + addition
	lines = append(lines[:start+1], append([]string{block}, lines[end:]...)...)
	return strings.Join(lines, "")
}

func metaLink(path string) string {
	path = filepath.ToSlash(path)
	return fmt.Sprintf("- [%s](./%s)\n", filepath.Base(path), path)
}

func linkTargets(section string) map[string]struct{} {
	seen := make(map[string]struct{})
	for _, line := range strings.Split(section, "\n") {
		open := strings.Index(line, "](")
		if open < 0 {
			continue
		}
		close := strings.Index(line[open+2:], ")")
		if close < 0 {
			continue
		}
		href := line[open+2 : open+2+close]
		if !strings.HasPrefix(href, "./") {
			continue
		}
		seen[href] = struct{}{}
	}
	return seen
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
