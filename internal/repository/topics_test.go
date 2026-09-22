package repository

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"
)

func newTestStore(t *testing.T) TopicStore {
	t.Helper()
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return NewTopicStore(workspace)
}

func TestCreateTopicAutoIDUsesModernTimestampAllocation(t *testing.T) {
	store := newTestStore(t)
	for _, folder := range []string{"00001-first", "00003-third"} {
		if err := os.Mkdir(filepath.Join(store.Workspace.Histories, folder), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(store.Workspace.Database, "00002-archived"), 0o755); err != nil {
		t.Fatal(err)
	}
	topic, err := store.CreateTopic("New Topic", "")
	if err != nil {
		t.Fatalf("CreateTopic: %v", err)
	}
	id, parseErr := domain.ParseTopicIdentity(topic.ID.String())
	if parseErr != nil || !id.Valid() {
		t.Fatalf("auto ID %q is not a valid topic identity: %v", topic.ID, parseErr)
	}
	if len(topic.ID.String()) != 13 {
		t.Fatalf("auto ID = %s, want 13-digit modern topic ID", topic.ID)
	}
	if _, err := os.Stat(filepath.Join(topic.Path, markdown.MetaFilename)); err != nil {
		t.Fatalf("metadata: %v", err)
	}
	meta, err := markdown.ParseTopicMetadataFile(filepath.Join(topic.Path, markdown.MetaFilename))
	if err != nil {
		t.Fatalf("parse metadata: %v", err)
	}
	if meta.Description != "" || topic.Description != "" {
		t.Fatalf("description = %q, topic description = %q, want empty", meta.Description, topic.Description)
	}
}

func TestCreateTopicManualIDAndCollision(t *testing.T) {
	store := newTestStore(t)
	created, err := store.CreateTopic("Manual Topic", "00015")
	if err != nil {
		t.Fatalf("manual CreateTopic: %v", err)
	}
	if created.ID != "00015" {
		t.Fatalf("ID = %s, want 00015", created.ID)
	}
	_, err = store.CreateTopic("Other Topic", "00015")
	if !errors.Is(err, domain.ErrDuplicateID) {
		t.Fatalf("duplicate error = %v, want ErrDuplicateID", err)
	}
}

func TestCreateTopicRejectsInvalidAndEmptyInput(t *testing.T) {
	store := newTestStore(t)
	for title, id := range map[string]string{"": "", "No ID": "15"} {
		if _, err := store.CreateTopic(title, id); err == nil {
			t.Errorf("CreateTopic(%q, %q) accepted invalid input", title, id)
		}
	}
}

func TestCreateTopicDoesNotOverwriteExistingFolder(t *testing.T) {
	store := newTestStore(t)
	path := filepath.Join(store.Workspace.Histories, "00001-existing-topic")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "keep.md"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := store.CreateTopic("Existing Topic", "00001")
	if !errors.Is(err, domain.ErrDuplicateID) {
		t.Fatalf("error = %v, want ErrDuplicateID", err)
	}
	content, err := os.ReadFile(filepath.Join(path, "keep.md"))
	if err != nil || string(content) != "keep" {
		t.Fatalf("existing content changed: %q, %v", content, err)
	}
}
