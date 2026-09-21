package markdown

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"historic/internal/domain"
)

func TestTopicMetadataRequiresCanonicalFile(t *testing.T) {
	topic := t.TempDir()
	legacy := filepath.Join(topic, "_meta.md")
	legacyContent := []byte("---\nid: \"00011\"\ntitle: Legacy Topic\nstatus: progress\ncreated: 2026-09-21\n---\n# Legacy Topic\n")
	if err := os.WriteFile(legacy, legacyContent, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadTopicMetadata(topic); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy metadata read error = %v, want missing canonical metadata", err)
	}
	if _, err := os.Stat(legacy); err != nil {
		t.Fatalf("legacy asset changed or removed: %v", err)
	}
}

func TestTopicMetadataInvalidCanonicalDoesNotReadLegacy(t *testing.T) {
	topic := t.TempDir()
	legacy := filepath.Join(topic, "_meta.md")
	if err := os.WriteFile(legacy, []byte("legacy asset"), 0o644); err != nil {
		t.Fatal(err)
	}
	canonical := filepath.Join(topic, MetaFilename)
	if err := os.WriteFile(canonical, []byte("id: not-an-id\ntitle: Broken\ncreated: 2026-09-21\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadTopicMetadata(topic); err == nil {
		t.Fatal("invalid canonical metadata was accepted")
	}
	got, err := os.ReadFile(legacy)
	if err != nil || string(got) != "legacy asset" {
		t.Fatalf("legacy asset = %q err=%v", got, err)
	}
}

func TestWriteTopicMetadataManifestRejectsInvalidManifestAtomically(t *testing.T) {
	topic := t.TempDir()
	path := filepath.Join(topic, MetaFilename)
	metadata := TopicMetadata{ID: "00005", Title: "Topic", Status: domain.StatusProgress, Created: "2026-09-21"}
	if err := WriteTopicMetadata(path, metadata); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := WriteTopicMetadataManifest(path, metadata, []ManifestFile{{Path: "../escape", Type: "task", Status: domain.StatusProgress}}, nil)
	if err == nil || updated {
		t.Fatalf("invalid manifest result = updated:%v err:%v", updated, err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("invalid manifest replaced canonical metadata")
	}
}
