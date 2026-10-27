package repository

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"historic/internal/domain"
	"historic/internal/markdown"
)

const (
	parentTopicID = "1791851149892"
	childTopicID  = "1792889892200"
)

func TestCreateTopicExtendsClosedParent(t *testing.T) {
	store := newTestStore(t)
	parent, err := store.CreateTopic("Parent", parentTopicID, "")
	if err != nil {
		t.Fatal(err)
	}
	closedParent := filepath.Join(store.Workspace.Database, filepath.Base(parent.Path))
	if err := os.Rename(parent.Path, closedParent); err != nil {
		t.Fatal(err)
	}
	parentMeta := filepath.Join(closedParent, markdown.MetaFilename)
	before, err := os.ReadFile(parentMeta)
	if err != nil {
		t.Fatal(err)
	}

	child, err := store.CreateTopic("Lanjutan", childTopicID, parentTopicID)
	if err != nil {
		t.Fatalf("CreateTopic extends: %v", err)
	}
	if child.Extends != domain.TopicID(parentTopicID) {
		t.Fatalf("child.Extends = %q, want %q", child.Extends, parentTopicID)
	}
	if child.ID == parent.ID {
		t.Fatalf("child ID %s equals parent ID", child.ID)
	}
	metadata, err := markdown.ParseTopicMetadataFile(filepath.Join(child.Path, markdown.MetaFilename))
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Extends != domain.TopicID(parentTopicID) {
		t.Fatalf("manifest extends = %q, want %q", metadata.Extends, parentTopicID)
	}
	if len(metadata.Files) != 0 {
		t.Fatalf("child inherited files: %#v", metadata.Files)
	}
	after, err := os.ReadFile(parentMeta)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("parent metadata changed by extension create")
	}
}

func TestCreateTopicExtendsOpenParent(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.CreateTopic("Parent", parentTopicID, ""); err != nil {
		t.Fatal(err)
	}
	child, err := store.CreateTopic("Lanjutan", childTopicID, parentTopicID)
	if err != nil {
		t.Fatalf("CreateTopic extends open parent: %v", err)
	}
	if child.Extends != domain.TopicID(parentTopicID) {
		t.Fatalf("child.Extends = %q, want %q", child.Extends, parentTopicID)
	}
}

func TestCreateTopicExtendsRejectsMissingAndInvalidParents(t *testing.T) {
	store := newTestStore(t)
	before := listDirNames(t, store.Workspace.Histories)

	if _, err := store.CreateTopic("Lanjutan", "", "1799999999999"); !errors.Is(err, domain.ErrTopicMissing) {
		t.Fatalf("missing parent error = %v, want ErrTopicMissing", err)
	}
	if _, err := store.CreateTopic("Lanjutan", "", "not-a-topic-id"); !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("invalid parent error = %v, want ErrInvalidID", err)
	}
	after := listDirNames(t, store.Workspace.Histories)
	if len(before) != len(after) {
		t.Fatalf("topic directories changed: before=%v after=%v", before, after)
	}
}

func TestCreateTopicWithoutExtendsOmitsKey(t *testing.T) {
	store := newTestStore(t)
	topic, err := store.CreateTopic("Plain", childTopicID, "")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(topic.Path, markdown.MetaFilename))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "extends") {
		t.Fatalf("metadata unexpectedly contains extends:\n%s", raw)
	}
	if topic.Extends != "" {
		t.Fatalf("topic.Extends = %q, want empty", topic.Extends)
	}
}

func TestLegacyMetadataWithoutExtendsParses(t *testing.T) {
	store := newTestStore(t)
	topic, err := store.CreateTopic("Legacy", childTopicID, "")
	if err != nil {
		t.Fatal(err)
	}
	legacy := "id: \"" + childTopicID + "\"\ntitle: Legacy\ncreated: \"2026-10-25\"\nfiles: []\nassets: []\n"
	if err := os.WriteFile(filepath.Join(topic.Path, markdown.MetaFilename), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	metadata, err := markdown.ParseTopicMetadataFile(filepath.Join(topic.Path, markdown.MetaFilename))
	if err != nil {
		t.Fatalf("legacy metadata rejected: %v", err)
	}
	if metadata.Extends != "" {
		t.Fatalf("legacy extends = %q, want empty", metadata.Extends)
	}
}

func listDirNames(t *testing.T, path string) []string {
	t.Helper()
	entries, err := os.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}
