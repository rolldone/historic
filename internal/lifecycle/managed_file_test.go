package lifecycle

import (
	"os"
	"path/filepath"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"
)

func TestSyncMetaClassifiesWorkOrderWithDifferentIDAndPreservesStatus(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00005-topic")
	if err := os.MkdirAll(filepath.Join(topic, "wos"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename), markdown.TopicMetadata{
		ID: "00005", Title: "Topic", Created: "2026-09-22",
	}); err != nil {
		t.Fatal(err)
	}
	workOrder, err := markdown.NewDocument(domain.Frontmatter{
		ID: "00001", Title: "WO", Status: domain.StatusComplete, Created: "2026-09-22",
	}, "work order")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(topic, "wos", "01-work-order.md")
	if err := markdown.WriteFile(path, workOrder); err != nil {
		t.Fatal(err)
	}
	if _, err := RebuildMetadata(workspace); err != nil {
		t.Fatal(err)
	}
	metadata, err := markdown.ReadTopicMetadata(topic)
	if err != nil {
		t.Fatal(err)
	}
	if len(metadata.Files) != 1 || metadata.Files[0].Path != "wos/01-work-order.md" || metadata.Files[0].Type != "task" || metadata.Files[0].Status != domain.StatusComplete {
		t.Fatalf("manifest = %#v", metadata.Files)
	}
	if len(metadata.Assets) != 0 {
		t.Fatalf("assets = %#v, want empty", metadata.Assets)
	}
}
