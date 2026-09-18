package repository

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"historic/internal/domain"
	"historic/internal/markdown"
)

// TopicView is a read model for list and show commands.
type TopicView struct {
	ID      string     `json:"id"`
	Title   string     `json:"title"`
	Status  string     `json:"status"`
	Created string     `json:"created"`
	Updated string     `json:"updated,omitempty"`
	Path    string     `json:"path"`
	Files   []FileView `json:"files"`
	Body    string     `json:"body,omitempty"`
	Active  bool       `json:"active"`
}

// FileView describes a Markdown file inside a topic.
type FileView struct {
	Path   string `json:"path"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

// ListTopics returns deterministic topic views, filtered to active by default.
func (store TopicStore) ListTopics(includeArchived bool) ([]TopicView, error) {
	paths, err := store.ListTopicDirectories()
	if err != nil {
		return nil, fmt.Errorf("list topics: %w", err)
	}
	views := make([]TopicView, 0, len(paths))
	for _, path := range paths {
		active := filepath.Dir(path) == store.Workspace.Histories
		if !includeArchived && !active {
			continue
		}
		view, err := store.readTopicView(path, active, false)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	sort.Slice(views, func(i, j int) bool { return views[i].ID < views[j].ID })
	return views, nil
}

// ShowTopic returns one topic view, including its metadata and Markdown files.
func (store TopicStore) ShowTopic(id domain.ID, includeArchived bool) (TopicView, error) {
	if !id.Valid() {
		return TopicView{}, fmt.Errorf("%w: %q", domain.ErrInvalidID, id)
	}
	paths, err := store.ListTopicDirectories()
	if err != nil {
		return TopicView{}, fmt.Errorf("find topic: %w", err)
	}
	for _, path := range paths {
		if !strings.HasPrefix(filepath.Base(path), id.String()+"-") {
			continue
		}
		active := filepath.Dir(path) == store.Workspace.Histories
		if !active && !includeArchived {
			return TopicView{}, fmt.Errorf("%w: %s is archived; use --archived", domain.ErrTopicMissing, id)
		}
		return store.readTopicView(path, active, true)
	}
	return TopicView{}, fmt.Errorf("%w: %s", domain.ErrTopicMissing, id)
}

func (store TopicStore) readTopicView(path string, active, includeBody bool) (TopicView, error) {
	metaPath := filepath.Join(path, "_meta.md")
	meta, err := markdown.ParseFile(metaPath)
	if err != nil {
		return TopicView{}, fmt.Errorf("read metadata %s: %w", store.Workspace.RelativePath(metaPath), err)
	}
	view := TopicView{
		ID: meta.Frontmatter.ID.String(), Title: meta.Frontmatter.Title, Status: meta.Frontmatter.Status.String(),
		Created: meta.Frontmatter.Created, Updated: meta.Frontmatter.Updated,
		Path: store.Workspace.RelativePath(path), Body: meta.Body, Active: active,
		Files: []FileView{},
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
		if entry.IsDir() || filepath.Base(path) == "_meta.md" || strings.ToLower(filepath.Ext(path)) != ".md" {
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
		files = append(files, FileView{Path: filepath.ToSlash(relative), Title: document.Frontmatter.Title, Status: document.Frontmatter.Status.String()})
		return nil
	})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}
