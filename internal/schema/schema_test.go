package schema

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestEnsureReadModelCreatesCalibratedTopicsAndFilesSchema(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, err := database.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatal(err)
	}
	if err := EnsureReadModel(database); err != nil {
		t.Fatal(err)
	}
	if err := EnsureReadModel(database); err != nil {
		t.Fatalf("schema is not idempotent: %v", err)
	}

	var topics, files int
	if err := database.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='topics'").Scan(&topics); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='files'").Scan(&files); err != nil {
		t.Fatal(err)
	}
	if topics != 1 || files != 1 {
		t.Fatalf("topics=%d files=%d, want one table each", topics, files)
	}

	if _, err := database.Exec(`INSERT INTO topics (id, num_padded, title, slug, path, storage, created_at) VALUES ('00001', '00001', 'Topic', 'topic', '.historic/00001-topic', 'open', '2026-09-21')`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO files (topic_id, type, path, filename, title, status, mtime, hash) VALUES ('00001', 'asset', 'docs/readme.txt', 'readme.txt', 'Readme', NULL, '2026-09-21T00:00:00Z', 'hash')`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO files (topic_id, type, path, filename, title, status, mtime, hash) VALUES ('00001', 'historic_file', 'wos/01-task.md', '01-task.md', 'Task', 'progress', '2026-09-21T00:00:00Z', 'hash2')`); err != nil {
		t.Fatal(err)
	}

	checks := []string{
		`INSERT INTO files (topic_id, type, path, filename, title, status, mtime, hash) VALUES ('00001', 'asset', '.historic/escape.txt', 'escape.txt', 'Escape', NULL, 'now', 'x')`,
		`INSERT INTO files (topic_id, type, path, filename, title, status, mtime, hash) VALUES ('00001', 'asset', 'asset.txt', 'asset.txt', 'Asset', 'progress', 'now', 'y')`,
	}
	for _, statement := range checks {
		if _, err := database.Exec(statement); err == nil {
			t.Fatalf("invalid file accepted: %s", statement)
		}
	}
	if _, err := database.Exec(`DELETE FROM topics WHERE id = '00001'`); err != nil {
		t.Fatal(err)
	}
	var remaining int
	if err := database.QueryRow("SELECT COUNT(*) FROM files").Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 0 {
		t.Fatalf("foreign-key cascade left %d files", remaining)
	}
}
