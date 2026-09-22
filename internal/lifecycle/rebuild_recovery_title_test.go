package lifecycle

import (
	"os"
	"path/filepath"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"
)

func TestRecoverTopicMetadataUsesFolderSlugTitle(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00005-search-read-model-v2")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "WO 01 Two-Table Search Read Model", Status: domain.StatusComplete, Created: "2026-09-21"}, "body")
	if err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(filepath.Join(topic, "wos", "01-wo.md"), file); err != nil {
		t.Fatal(err)
	}
	if _, err := RebuildMetadata(workspace); err != nil {
		t.Fatal(err)
	}
	metadata, err := markdown.ReadTopicMetadata(topic)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Title != "search read model v2" {
		t.Fatalf("recovered title = %q, want folder slug title", metadata.Title)
	}
}
