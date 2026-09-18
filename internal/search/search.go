package search

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"
)

// Options controls a filesystem search.
type Options struct {
	Keyword         string
	Status          domain.Status
	Folder          string
	ActiveOnly      bool
	ArchivedOnly    bool
	IncludeArchived bool
}

// Result is one matching Markdown entry or topic metadata file.
type Result struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	Path    string `json:"path"`
	Snippet string `json:"snippet"`
	Active  bool   `json:"active"`
}

// Find scans Markdown source files without invoking a shell or external command.
func Find(workspace config.Workspace, options Options) ([]Result, error) {
	keyword := strings.TrimSpace(options.Keyword)
	if keyword == "" {
		return nil, fmt.Errorf("keyword must not be empty")
	}
	if options.ActiveOnly && options.ArchivedOnly {
		return nil, fmt.Errorf("active and archived filters cannot be combined")
	}
	roots := []struct {
		path   string
		active bool
	}{
		{workspace.Histories, true},
	}
	if !options.ActiveOnly {
		roots = append(roots, struct {
			path   string
			active bool
		}{workspace.Database, false})
	}
	if options.ArchivedOnly {
		roots = roots[1:]
	}
	results := make([]Result, 0)
	for _, root := range roots {
		err := walkRoot(workspace, root.path, root.active, options, keyword, &results)
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Path < results[j].Path })
	return results, nil
}

func walkRoot(workspace config.Workspace, root string, active bool, options Options, keyword string, results *[]Result) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		document, err := markdown.ParseFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", workspace.RelativePath(path), err)
		}
		if options.Status != "" && document.Frontmatter.Status != options.Status {
			return nil
		}
		relative := workspace.RelativePath(path)
		if options.Folder != "" && !folderMatches(relative, options.Folder) {
			return nil
		}
		if !containsMatch(keyword, relative, document.Frontmatter.Title, document.Body) {
			return nil
		}
		*results = append(*results, Result{
			ID: document.Frontmatter.ID.String(), Title: document.Frontmatter.Title,
			Status: document.Frontmatter.Status.String(), Path: relative,
			Snippet: snippet(document.Body, keyword), Active: active,
		})
		return nil
	})
}

func folderMatches(path, folder string) bool {
	folder = filepath.ToSlash(strings.Trim(strings.TrimSpace(folder), "/"))
	path = filepath.ToSlash(path)
	return folder != "" && (path == folder || strings.HasPrefix(path, folder+"/"))
}

func containsMatch(keyword string, values ...string) bool {
	keyword = strings.ToLower(keyword)
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), keyword) {
			return true
		}
	}
	return false
}

func snippet(body, keyword string) string {
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), strings.ToLower(keyword)) {
			return strings.TrimSpace(line)
		}
	}
	scanner := bufio.NewScanner(strings.NewReader(body))
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}
