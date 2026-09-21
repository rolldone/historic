package markdown

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLegacyTopicMetadataMigrationSmoke(t *testing.T) {
	topic := t.TempDir()
	legacy := filepath.Join(topic, LegacyMetaFilename)
	legacyContent := []byte("---\nid: \"00011\"\ntitle: Legacy Topic\ndescription: frontmatter description\nstatus: progress\ncreated: 2026-09-21\nupdated: 2026-09-22\ntags: [legacy, yaml]\nrelated: [\"00002\"]\n---\n# Legacy Topic\n\n## Deskripsi\n\nbody description\n\n## Files\n\n- [stale.md](./stale.md)\n\n## Assets\n\n- [old.png](./old.png)\n\n## Progress\n\nkeep this note\n")
	if err := os.WriteFile(legacy, legacyContent, 0o644); err != nil {
		t.Fatal(err)
	}
	metadata, err := ReadTopicMetadata(topic)
	if err != nil {
		t.Fatalf("migrate legacy metadata: %v", err)
	}
	if metadata.ID.String() != "00011" || metadata.Description != "frontmatter description" || len(metadata.Tags) != 2 || len(metadata.Related) != 1 {
		t.Fatalf("metadata = %#v", metadata)
	}
	canonical := filepath.Join(topic, MetaFilename)
	if _, err := ParseTopicMetadataFile(canonical); err != nil {
		t.Fatalf("parse canonical metadata: %v", err)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatalf("legacy metadata still exists: %v", err)
	}
	canonicalBytes, err := os.ReadFile(canonical)
	if err != nil {
		t.Fatal(err)
	}
	if string(canonicalBytes) == string(legacyContent) || string(canonicalBytes) == "" {
		t.Fatalf("canonical metadata unexpectedly copied legacy Markdown: %q", canonicalBytes)
	}
	if _, err := ReadTopicMetadata(topic); err != nil {
		t.Fatalf("second metadata read: %v", err)
	}
}

func TestLegacyTopicMetadataMigrationPreservesLegacyOnFailure(t *testing.T) {
	topic := t.TempDir()
	legacy := filepath.Join(topic, LegacyMetaFilename)
	content := []byte("---\nid: not-an-id\ntitle: Broken\ncreated: 2026-09-21\n---\n")
	if err := os.WriteFile(legacy, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadTopicMetadata(topic); err == nil {
		t.Fatal("invalid legacy metadata migrated successfully")
	}
	got, err := os.ReadFile(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Fatal("invalid legacy metadata was changed")
	}
	if _, err := os.Stat(filepath.Join(topic, MetaFilename)); !os.IsNotExist(err) {
		t.Fatalf("canonical metadata created after failed migration: %v", err)
	}
}
