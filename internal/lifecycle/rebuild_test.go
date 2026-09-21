package lifecycle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"
)

func TestRebuildMetadataProcessesOpenAndClosedTopics(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		root   string
		id     domain.ID
		status domain.Status
	}{{workspace.Histories, "00001", domain.StatusProgress}, {workspace.Database, "00002", domain.StatusComplete}} {
		topic := filepath.Join(item.root, item.id.String()+"-topic")
		if err := os.MkdirAll(topic, 0o755); err != nil {
			t.Fatal(err)
		}
		document, _ := markdown.NewDocument(domain.Frontmatter{ID: item.id, Title: "Topic", Status: item.status, Created: "2026-09-21"}, "## Deskripsi\n\n## Files\n\n- [stale](./deleted.md)\n\n## Assets\n\n- [old](./old.png)\n\n## Progress\n")
		if err := markdown.WriteFile(filepath.Join(topic, "_meta.md"), document); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(topic, "current.md"), []byte("not-managed"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	change, err := RebuildMetadata(workspace)
	if err != nil || change.Topics != 2 || change.Records != 2 {
		t.Fatalf("change=%#v err=%v", change, err)
	}
	for _, item := range []struct {
		root   string
		id     domain.ID
		status domain.Status
	}{{workspace.Histories, "00001", domain.StatusProgress}, {workspace.Database, "00002", domain.StatusComplete}} {
		meta, err := markdown.ParseFile(filepath.Join(item.root, item.id.String()+"-topic", "_meta.md"))
		if err != nil {
			t.Fatal(err)
		}
		if meta.Frontmatter.Status != item.status {
			t.Fatalf("status=%s want %s", meta.Frontmatter.Status, item.status)
		}
		if !contains(meta.Body, "current.md") || contains(meta.Body, "deleted.md") {
			t.Fatalf("metadata body=%q", meta.Body)
		}
	}
}

func contains(value, part string) bool {
	return len(value) >= len(part) && filepath.Base(part) != "" && strings.Contains(value, part)
}
