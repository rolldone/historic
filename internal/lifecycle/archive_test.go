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

func TestArchiveStatusChangesWorkStatusWithoutMovingTopic(t *testing.T) {
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
	if change.Archived || change.Current != domain.StatusComplete || change.Storage != domain.StorageOpen || change.Path != ".historic/00001-topic" {
		t.Fatalf("change = %#v", change)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(workspace.Database, "00001-topic")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("closed topic exists: %v", err)
	}
}

func TestCloseAndOpenPreserveStatusAndFiles(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(workspace.Histories, "00001-topic")
	if err := os.MkdirAll(filepath.Join(source, "wos"), 0o755); err != nil {
		t.Fatal(err)
	}
	document, _ := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Topic", Status: domain.StatusBlocked, Created: "2026-09-18"}, "meta")
	if err := markdown.WriteFile(filepath.Join(source, "_meta.md"), document); err != nil {
		t.Fatal(err)
	}
	content := []byte("entry bytes\n")
	if err := os.WriteFile(filepath.Join(source, "wos", "01-task.md"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	service := NewService(workspace)
	closed, err := service.Close("00001")
	if err != nil || closed.Storage != domain.StorageClosed {
		t.Fatalf("close = %#v err=%v", closed, err)
	}
	opened, err := service.Open("00001")
	if err != nil || opened.Storage != domain.StorageOpen || opened.Current != domain.StatusBlocked {
		t.Fatalf("open = %#v err=%v", opened, err)
	}
	got, err := os.ReadFile(filepath.Join(source, "wos", "01-task.md"))
	if err != nil || string(got) != string(content) {
		t.Fatalf("content = %q err=%v", got, err)
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
