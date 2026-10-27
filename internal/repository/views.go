package repository

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"historic/internal/domain"
	"historic/internal/markdown"
)

// TopicView is a read model for list and show commands.
type TopicView struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Created     string     `json:"created"`
	Updated     string     `json:"updated,omitempty"`
	Mtime       string     `json:"mtime,omitempty"`
	Path        string     `json:"path"`
	Storage     string     `json:"storage"`
	Extends     string     `json:"extends,omitempty"`
	Files       []FileView `json:"files"`
	Body        string     `json:"body,omitempty"`
	Active      bool       `json:"active"`
}

// FileView describes a Markdown file inside a topic.
type FileView struct {
	Path        string `json:"path"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	ForwardedTo string `json:"forwarded_to,omitempty"`
}

// ListTopics returns deterministic topic views for one storage scope. Both the
// open workdir and its closed mirror are listed independently, matching find.
func (store TopicStore) ListTopics(closedOnly bool) ([]TopicView, error) {
	views := make([]TopicView, 0)
	for _, root := range []struct {
		path   string
		active bool
	}{{store.Workspace.Histories, true}, {store.Workspace.Database, false}} {
		if closedOnly == root.active {
			continue
		}
		entries, err := os.ReadDir(root.path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("list topics: %w", err)
		}
		seen := make(map[string]struct{}, len(entries))
		for _, entry := range entries {
			match := topicFolderPattern.FindStringSubmatch(entry.Name())
			if len(match) != 3 {
				continue
			}
			path := filepath.Join(root.path, entry.Name())
			info, err := os.Lstat(path)
			if err != nil {
				return nil, err
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return nil, fmt.Errorf("%w: topic symlink %s", domain.ErrConflict, store.Workspace.RelativePath(path))
			}
			if !info.IsDir() {
				continue
			}
			if _, exists := seen[match[1]]; exists {
				return nil, fmt.Errorf("%w: duplicate %s topic ID %s", domain.ErrConflict, map[bool]string{true: "open", false: "closed"}[root.active], match[1])
			}
			seen[match[1]] = struct{}{}
			view, err := store.readTopicView(path, root.active, false)
			if err != nil {
				return nil, err
			}
			views = append(views, view)
		}
	}
	sort.Slice(views, func(i, j int) bool {
		if views[i].ID != views[j].ID {
			return views[i].ID < views[j].ID
		}
		return views[i].Storage == "open" && views[j].Storage != "open"
	})
	return views, nil
}

// ShowTopics returns all versions of a topic, open first; includeClosed filters to closed copies only.
func (store TopicStore) ShowTopics(id domain.ID, includeClosed bool) ([]TopicView, error) {
	if !id.Valid() {
		return nil, fmt.Errorf("%w: %q", domain.ErrInvalidID, id)
	}
	views := make([]TopicView, 0, 2)
	prefix := id.String() + "-"
	for _, root := range []string{store.Workspace.Histories, store.Workspace.Database} {
		entries, readErr := os.ReadDir(root)
		if errors.Is(readErr, os.ErrNotExist) {
			continue
		}
		if readErr != nil {
			return nil, fmt.Errorf("find topic: %w", readErr)
		}
		for _, entry := range entries {
			if !strings.HasPrefix(entry.Name(), prefix) || strings.HasPrefix(entry.Name(), ".staging-") {
				continue
			}
			path := filepath.Join(root, entry.Name())
			info, statErr := os.Lstat(path)
			if statErr != nil {
				return nil, statErr
			}
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				continue
			}
			active := root == store.Workspace.Histories
			if includeClosed && active {
				continue
			}
			view, viewErr := store.readTopicView(path, active, true)
			if viewErr != nil {
				return nil, viewErr
			}
			views = append(views, view)
		}
	}
	sort.SliceStable(views, func(i, j int) bool {
		return views[i].Storage == "open" && views[j].Storage != "open"
	})
	if len(views) == 0 {
		return nil, fmt.Errorf("%w: %s", domain.ErrTopicMissing, id)
	}
	return views, nil
}

// ShowTopic preserves the single-view API for existing callers.
func (store TopicStore) ShowTopic(id domain.ID, includeClosed bool) (TopicView, error) {
	views, err := store.ShowTopics(id, includeClosed)
	if err != nil {
		return TopicView{}, err
	}
	return views[0], nil
}

func (store TopicStore) readTopicView(path string, active, includeBody bool) (TopicView, error) {
	meta, err := markdown.ReadTopicMetadata(path)
	if err != nil {
		return TopicView{}, fmt.Errorf("read metadata %s: %w", store.Workspace.RelativePath(filepath.Join(path, markdown.MetaFilename)), err)
	}
	metaPath := filepath.Join(path, markdown.MetaFilename)
	info, err := os.Stat(metaPath)
	if err != nil {
		return TopicView{}, fmt.Errorf("stat topic metadata: %w", err)
	}
	view := TopicView{
		ID: meta.ID.String(), Title: meta.Title, Description: meta.Description,
		Created: meta.Created, Updated: meta.Updated, Mtime: info.ModTime().UTC().Format(time.RFC3339Nano),
		Path: store.Workspace.RelativePath(path), Body: "", Active: active,
		Storage: map[bool]string{true: "open", false: "closed"}[active],
		Extends: meta.Extends.String(),
		Files:   []FileView{},
	}
	if includeBody {
		view.Files, err = readFiles(path, store.Workspace.Root)
		if err != nil {
			return TopicView{}, err
		}
	}
	return view, nil
}

func readFiles(topicPath, root string) ([]FileView, error) {
	files := make([]FileView, 0)
	err := filepath.WalkDir(topicPath, func(path string, entry os.DirEntry, walkErr error) error {
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
		if entry.IsDir() || filepath.Base(path) == markdown.MetaFilename || strings.ToLower(filepath.Ext(path)) != ".md" {
			return nil
		}
		document, err := markdown.ParseFile(path)
		if err != nil {
			return fmt.Errorf("read file %s: %w", filepath.ToSlash(path), err)
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, FileView{Path: filepath.ToSlash(relative), Title: document.Frontmatter.Title, Description: document.Frontmatter.Description, Status: document.Frontmatter.Status.String(), ForwardedTo: document.Frontmatter.ForwardedTo.String()})
		return nil
	})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}
