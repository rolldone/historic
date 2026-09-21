package config

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestInitializeMigratesLegacyIndexBeforeStorageIndex(t *testing.T) {
	root := t.TempDir()
	workspace := NewWorkspace(root)
	if err := os.MkdirAll(workspace.Histories, 0o755); err != nil {
		t.Fatal(err)
	}
	database, err := sql.Open("sqlite", workspace.Index)
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(`CREATE TABLE index_records (id INTEGER PRIMARY KEY, num INTEGER NOT NULL, num_padded TEXT NOT NULL, type TEXT NOT NULL DEFAULT '', title TEXT NOT NULL, status TEXT NOT NULL, tags TEXT NOT NULL DEFAULT '[]', related TEXT NOT NULL DEFAULT '[]', created_at TEXT NOT NULL, updated_at TEXT, path TEXT NOT NULL UNIQUE, folder_id TEXT NOT NULL, folder_slug TEXT NOT NULL DEFAULT '', subfolder TEXT NOT NULL DEFAULT '', filename TEXT NOT NULL, file_order INTEGER NOT NULL DEFAULT 0, content TEXT NOT NULL DEFAULT '', word_count INTEGER NOT NULL DEFAULT 0, mtime TEXT NOT NULL, hash TEXT NOT NULL DEFAULT '')`)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Initialize(root); err != nil {
		t.Fatalf("Initialize legacy index: %v", err)
	}
	check, err := sql.Open("sqlite", filepath.Clean(workspace.Index))
	if err != nil {
		t.Fatal(err)
	}
	defer check.Close()
	var count int
	if err := check.QueryRow("SELECT COUNT(*) FROM pragma_table_info('index_records') WHERE name='storage'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("storage column count = %d", count)
	}
}
