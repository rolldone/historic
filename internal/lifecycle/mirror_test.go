package lifecycle

import (
	"os"
	"path/filepath"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"
)

func TestMirrorTopicPreservesOpenAndMirrorsIdempotently(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	openPath := filepath.Join(workspace.Histories, "00001-topic")
	if err := os.MkdirAll(openPath, 0o755); err != nil {
		t.Fatal(err)
	}
	meta, err := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Topic", Status: domain.StatusProgress, Created: "2026-10-01", Updated: "2026-10-10"}, "body")
	if err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(filepath.Join(openPath, markdown.MetaFilename), meta); err != nil {
		t.Fatal(err)
	}
	service := NewService(workspace)
	if _, err := service.MirrorTopic("00001"); err != nil {
		t.Fatal(err)
	}
	closedPath := filepath.Join(workspace.Database, "00001-topic")
	if _, err := os.Stat(openPath); err != nil {
		t.Fatalf("open topic removed: %v", err)
	}
	closedMeta, err := markdown.ReadTopicMetadata(closedPath)
	if err != nil || closedMeta.Updated != "2026-10-10" {
		t.Fatalf("mirror timestamp = %q, err=%v", closedMeta.Updated, err)
	}
	before, err := os.ReadFile(filepath.Join(closedPath, markdown.MetaFilename))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.MirrorTopic("00001"); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(closedPath, markdown.MetaFilename))
	if err != nil || string(before) != string(after) {
		t.Fatalf("idempotent mirror changed snapshot: err=%v", err)
	}
}
