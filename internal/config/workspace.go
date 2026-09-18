package config

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"historic/internal/gitproxy"

	_ "modernc.org/sqlite"
)

const (
	HistoriesDirName = ".histories"
	DatabaseDirName  = ".database"
	IndexFileName    = ".index.sqlite"
)

var ErrNotDirectory = errors.New("path exists but is not a directory")

// Workspace describes the filesystem locations used by Historic.
type Workspace struct {
	Root      string
	Histories string
	Database  string
	Index     string
}

// NewWorkspace returns canonical paths for a workspace root.
func NewWorkspace(root string) Workspace {
	root, _ = filepath.Abs(root)
	histories := filepath.Join(root, HistoriesDirName)
	return Workspace{
		Root:      root,
		Histories: histories,
		Database:  filepath.Join(histories, DatabaseDirName),
		Index:     filepath.Join(histories, IndexFileName),
	}
}

// DiscoverRoot finds the nearest initialized Historic workspace, or returns start.
func DiscoverRoot(start string) (string, error) {
	root, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve workspace root: %w", err)
	}
	if info, err := os.Stat(root); err != nil {
		return "", fmt.Errorf("inspect workspace root %s: %w", root, err)
	} else if !info.IsDir() {
		return "", fmt.Errorf("%w: %s", ErrNotDirectory, root)
	}

	for current := root; ; current = filepath.Dir(current) {
		candidate := filepath.Join(current, HistoriesDirName)
		info, statErr := os.Stat(candidate)
		switch {
		case statErr == nil && info.IsDir():
			return current, nil
		case statErr == nil && !info.IsDir():
			return "", fmt.Errorf("%w: %s", ErrNotDirectory, candidate)
		case statErr != nil && !errors.Is(statErr, os.ErrNotExist):
			return "", fmt.Errorf("inspect %s: %w", candidate, statErr)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return root, nil
		}
	}
}

// Initialize creates the workspace structure and an empty rebuildable SQLite index.
// Existing files and topic contents are preserved.
func Initialize(root string) (Workspace, error) {
	workspace := NewWorkspace(root)
	if err := ensureDirectory(workspace.Root); err != nil {
		return Workspace{}, err
	}
	if info, err := os.Stat(workspace.Histories); err == nil && !info.IsDir() {
		return Workspace{}, fmt.Errorf("%w: %s", ErrNotDirectory, workspace.Histories)
	}
	if err := os.MkdirAll(workspace.Histories, 0o755); err != nil {
		return Workspace{}, fmt.Errorf("create %s: %w", workspace.Histories, err)
	}
	if err := ensureDirectory(workspace.Histories); err != nil {
		return Workspace{}, err
	}
	if err := os.MkdirAll(workspace.Database, 0o755); err != nil {
		return Workspace{}, fmt.Errorf("create %s: %w", workspace.Database, err)
	}
	if err := ensureDirectory(workspace.Database); err != nil {
		return Workspace{}, err
	}
	if err := initializeIndex(workspace.Index); err != nil {
		return Workspace{}, err
	}
	if _, err := gitproxy.Open(workspace.Database); err != nil {
		return Workspace{}, fmt.Errorf("initialize internal git repository: %w", err)
	}
	return workspace, nil
}

func ensureDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("inspect %s: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%w: %s", ErrNotDirectory, path)
	}
	return nil
}

func initializeIndex(path string) error {
	database, err := sql.Open("sqlite", path)
	if err != nil {
		return fmt.Errorf("open index %s: %w", path, err)
	}
	defer database.Close()
	if err := database.Ping(); err != nil {
		return fmt.Errorf("initialize index %s: %w", path, err)
	}
	const schema = `
CREATE TABLE IF NOT EXISTS index_records (
    id INTEGER PRIMARY KEY,
    num INTEGER NOT NULL,
    num_padded TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL,
    status TEXT NOT NULL,
    tags TEXT NOT NULL DEFAULT '[]',
    related TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL,
    updated_at TEXT,
    path TEXT NOT NULL UNIQUE,
    folder_id TEXT NOT NULL,
    folder_slug TEXT NOT NULL DEFAULT '',
    subfolder TEXT NOT NULL DEFAULT '',
    filename TEXT NOT NULL,
    file_order INTEGER NOT NULL DEFAULT 0,
    content TEXT NOT NULL DEFAULT '',
    word_count INTEGER NOT NULL DEFAULT 0,
    mtime TEXT NOT NULL,
    hash TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_index_records_num ON index_records(num);
CREATE INDEX IF NOT EXISTS idx_index_records_status ON index_records(status);
CREATE INDEX IF NOT EXISTS idx_index_records_type ON index_records(type);
CREATE INDEX IF NOT EXISTS idx_index_records_folder ON index_records(folder_id);
`
	if _, err := database.Exec(schema); err != nil {
		return fmt.Errorf("create index schema %s: %w", path, err)
	}
	return nil
}

// RelativePath returns a path relative to the workspace root for user output.
func (workspace Workspace) RelativePath(path string) string {
	relative, err := filepath.Rel(workspace.Root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(strings.TrimPrefix(relative, "./"))
}
