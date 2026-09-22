package repository

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"historic/internal/domain"
	"historic/internal/markdown"
)

func TestImportTopicOpensWithoutChangingStatusOrContent(t *testing.T) {
	store := newTestStore(t)
	source := filepath.Join(store.Workspace.Database, "00001-topic")
	if err := os.MkdirAll(filepath.Join(source, "wos"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteTopicMetadata(filepath.Join(source, markdown.MetaFilename), markdown.TopicMetadataFromFrontmatter(domain.Frontmatter{ID: "00001", Title: "Topic", Status: domain.StatusComplete, Created: "2026-09-18"}, "")); err != nil {
		t.Fatal(err)
	}
	entryContent := []byte("entry content\n")
	if err := os.WriteFile(filepath.Join(source, "wos", "01-task.md"), entryContent, 0o644); err != nil {
		t.Fatal(err)
	}
	topic, err := store.ImportTopic("00001", false)
	if err != nil {
		t.Fatal(err)
	}
	if topic.Path != filepath.Join(store.Workspace.Histories, "00001-topic") {
		t.Fatalf("topic = %#v", topic)
	}
	got, err := os.ReadFile(filepath.Join(topic.Path, "wos", "01-task.md"))
	if err != nil || string(got) != string(entryContent) {
		t.Fatalf("content = %q err=%v", got, err)
	}
	opened, err := markdown.ParseTopicMetadataFile(filepath.Join(topic.Path, markdown.MetaFilename))
	if err != nil {
		t.Fatalf("metadata = %#v err=%v", opened, err)
	}
}

func TestImportTopicRejectsConflictWithoutPartialDestination(t *testing.T) {
	store := newTestStore(t)
	source := filepath.Join(store.Workspace.Database, "00001-topic")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteTopicMetadata(filepath.Join(source, markdown.MetaFilename), markdown.TopicMetadataFromFrontmatter(domain.Frontmatter{ID: "00001", Title: "Topic", Status: domain.StatusComplete, Created: "2026-09-18"}, "")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(store.Workspace.Histories, "00001-topic"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ImportTopic("00001", false); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("error = %v, want conflict", err)
	}
}
