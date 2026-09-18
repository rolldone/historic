package repository

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"historic/internal/domain"
	"historic/internal/markdown"
)

func TestImportTopicCopiesArchiveAndReopensStatus(t *testing.T) {
	store := newTestStore(t)
	source := filepath.Join(store.Workspace.Database, "00001-topic")
	if err := os.MkdirAll(filepath.Join(source, "wos"), 0o755); err != nil {
		t.Fatal(err)
	}
	meta, _ := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Topic", Status: domain.StatusComplete, Created: "2026-09-18"}, "meta")
	if err := markdown.WriteFile(filepath.Join(source, "_meta.md"), meta); err != nil {
		t.Fatal(err)
	}
	entry, _ := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Task", Status: domain.StatusComplete, Created: "2026-09-18"}, "entry")
	if err := markdown.WriteFile(filepath.Join(source, "wos", "01-task.md"), entry); err != nil {
		t.Fatal(err)
	}
	topic, err := store.ImportTopic("00001", false)
	if err != nil {
		t.Fatal(err)
	}
	if topic.Status != domain.StatusProgress || topic.Path != filepath.Join(store.Workspace.Histories, "00001-topic") {
		t.Fatalf("topic = %#v", topic)
	}
	if _, err := os.Stat(filepath.Join(source, "wos", "01-task.md")); err != nil {
		t.Fatal(err)
	}
	imported, err := markdown.ParseFile(filepath.Join(topic.Path, "_meta.md"))
	if err != nil || imported.Frontmatter.Status != domain.StatusProgress {
		t.Fatalf("imported metadata = %#v err=%v", imported.Frontmatter, err)
	}
}

func TestImportTopicRejectsConflictWithoutPartialDestination(t *testing.T) {
	store := newTestStore(t)
	source := filepath.Join(store.Workspace.Database, "00001-topic")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	meta, _ := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Topic", Status: domain.StatusComplete, Created: "2026-09-18"}, "meta")
	if err := markdown.WriteFile(filepath.Join(source, "_meta.md"), meta); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(store.Workspace.Histories, "00001-topic"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ImportTopic("00001", false); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("error = %v, want conflict", err)
	}
}
