package lifecycle

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"
)

func TestArchiveStatusMovesCloseTopicAndPreservesFiles(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(workspace.Histories, "00001-topic")
	if err := os.MkdirAll(filepath.Join(source, "wos"), 0o755); err != nil {
		t.Fatal(err)
	}
	document, _ := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Topic", Status: domain.StatusProgress, Created: "2026-09-18"}, "meta")
	if err := markdown.WriteFile(filepath.Join(source, "_meta.md"), document); err != nil {
		t.Fatal(err)
	}
	entry, _ := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Task", Status: domain.StatusProgress, Created: "2026-09-18"}, "entry")
	if err := markdown.WriteFile(filepath.Join(source, "wos", "01-task.md"), entry); err != nil {
		t.Fatal(err)
	}
	change, err := NewService(workspace).ArchiveStatus("00001", domain.StatusComplete)
	if err != nil {
		t.Fatal(err)
	}
	if !change.Archived || change.Current != domain.StatusComplete || change.Path != ".historic/.database/00001-topic" {
		t.Fatalf("change = %#v", change)
	}
	if _, err := os.Stat(source); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("source still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspace.Database, "00001-topic", "wos", "01-task.md")); err != nil {
		t.Fatal(err)
	}
}

func TestArchiveStatusRejectsDestinationConflictWithoutDeletingSource(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(workspace.Histories, "00001-topic")
	destination := filepath.Join(workspace.Database, "00001-topic")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	document, _ := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Topic", Status: domain.StatusProgress, Created: "2026-09-18"}, "meta")
	if err := markdown.WriteFile(filepath.Join(source, "_meta.md"), document); err != nil {
		t.Fatal(err)
	}
	if _, err := NewService(workspace).ArchiveStatus("00001", domain.StatusComplete); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("error = %v, want conflict", err)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("source removed: %v", err)
	}
}
