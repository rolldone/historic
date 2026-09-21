package markdown

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
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
