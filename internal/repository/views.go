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
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	Created     string     `json:"created"`
	Updated     string     `json:"updated,omitempty"`
	Path        string     `json:"path"`
	Storage     string     `json:"storage"`
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
}

// ListTopics returns deterministic topic views, filtered to open by default.
func (store TopicStore) ListTopics(includeClosed bool) ([]TopicView, error) {
	paths, err := store.ListTopicDirectories()
	if err != nil {
		return nil, fmt.Errorf("list topics: %w", err)
	}
	views := make([]TopicView, 0, len(paths))
	for _, path := range paths {
		active := filepath.Dir(path) == store.Workspace.Histories
		view, err := store.readTopicView(path, active, false)
		if err != nil {
			return nil, err
		}
		if !includeClosed && view.Storage == "closed" {
			continue
		}
		views = append(views, view)
	}
	sort.Slice(views, func(i, j int) bool { return views[i].ID < views[j].ID })
	return views, nil
}

// ShowTopic returns one topic view, including its metadata and Markdown files.
func (store TopicStore) ShowTopic(id domain.ID, includeClosed bool) (TopicView, error) {
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
		if !includeClosed && !active {
			return TopicView{}, fmt.Errorf("%w: topic %s is closed", domain.ErrTopicMissing, id)
		}
		return store.readTopicView(path, active, true)
	}
	return TopicView{}, fmt.Errorf("%w: %s", domain.ErrTopicMissing, id)
}

func (store TopicStore) readTopicView(path string, active, includeBody bool) (TopicView, error) {
	meta, err := markdown.ReadTopicMetadata(path)
	if err != nil {
		return TopicView{}, fmt.Errorf("read metadata %s: %w", store.Workspace.RelativePath(filepath.Join(path, markdown.MetaFilename)), err)
	}
	view := TopicView{
		ID: meta.ID.String(), Title: meta.Title, Description: meta.Description, Status: meta.Status.String(),
		Created: meta.Created, Updated: meta.Updated,
		Path: store.Workspace.RelativePath(path), Body: "", Active: active,
		Storage: map[bool]string{true: "open", false: "closed"}[active],
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
		files = append(files, FileView{Path: filepath.ToSlash(relative), Title: document.Frontmatter.Title, Description: document.Frontmatter.Description, Status: document.Frontmatter.Status.String()})
		return nil
	})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}
