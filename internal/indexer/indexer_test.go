package indexer

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"
)

func TestRebuildIndexesMarkdownAndIsRepeatable(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00001-topic")
	if err := os.MkdirAll(filepath.Join(topic, "wos"), 0o755); err != nil {
		t.Fatal(err)
	}
	metadata := domain.Frontmatter{ID: "00001", Title: "Topic", Status: domain.StatusProgress, Created: "2026-09-18", Tags: []string{"one"}}
	for name := range map[string]string{"_meta.md": "meta body", "prd.md": "prd body", "wos/01-task.md": "task body"} {
		document, err := markdown.NewDocument(metadata, name+" body")
		if err != nil {
			t.Fatal(err)
		}
		if err := markdown.WriteFile(filepath.Join(topic, name), document); err != nil {
			t.Fatal(err)
		}
	}
	count, err := Rebuild(workspace)
	if err != nil || count != 3 {
		t.Fatalf("Rebuild = %d, %v", count, err)
	}
	indexed, err := Count(workspace)
	if err != nil || indexed != 3 {
		t.Fatalf("Count = %d, %v", indexed, err)
	}
	if _, err := Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	indexed, err = Count(workspace)
	if err != nil || indexed != 3 {
		t.Fatalf("second Count = %d, %v", indexed, err)
	}
}

func TestRebuildPreservesOldIndexWhenScanFails(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00001-topic")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	valid, _ := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Valid", Status: domain.StatusProgress, Created: "2026-09-18"}, "valid")
	if err := markdown.WriteFile(filepath.Join(topic, "note.md"), valid); err != nil {
		t.Fatal(err)
	}
	if _, err := Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(topic, "broken.md"), []byte("no frontmatter"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Rebuild(workspace); err == nil {
		t.Fatal("invalid Markdown rebuild succeeded")
	}
	count, err := Count(workspace)
	if err != nil || count != 1 {
		t.Fatalf("old index changed: count=%d err=%v", count, err)
	}
}

func TestRebuildCanRecreateDeletedIndex(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(workspace.Index); err != nil {
		t.Fatal(err)
	}
	if _, err := Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(workspace.Index); err != nil {
		t.Fatal(err)
	}
}

func TestIndexDatabaseHasExpectedRecords(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	database, err := sql.Open("sqlite", workspace.Index)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var table string
	if err := database.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='index_records'").Scan(&table); err != nil {
		t.Fatal(err)
	}
	if table != "index_records" {
		t.Fatal(errors.New("index table missing"))
	}
}
