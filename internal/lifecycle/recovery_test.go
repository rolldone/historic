package lifecycle

import (
	"os"
	"path/filepath"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"
)

func TestArchiveFailureLeavesSourceAndDetectsConflict(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(workspace.Histories, "00001-topic")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteTopicMetadata(filepath.Join(source, markdown.MetaFilename), markdown.TopicMetadata{ID: "00001", Title: "Topic", Created: "2026-09-18"}); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(workspace.Database, "00001-topic"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := NewService(workspace).ArchiveStatus("00001", domain.StatusComplete); err == nil {
		t.Fatal("archive conflict unexpectedly succeeded")
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("source was removed: %v", err)
	}
}
