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
		file, _ := markdown.NewDocument(domain.Frontmatter{ID: item.id, Title: "Current", Status: domain.StatusProgress, Created: "2026-09-21"}, "current body")
		if err := markdown.WriteFile(filepath.Join(topic, "current.md"), file); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(topic, "image.png"), []byte("asset"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	change, err := RebuildMetadata(workspace)
	if err != nil || change.Topics != 2 || change.Records != 4 {
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
		if !contains(meta.Body, "current.md") || !contains(meta.Body, "image.png") || contains(meta.Body, "deleted.md") || contains(meta.Body, "old.png") {
			t.Fatalf("metadata body=%q", meta.Body)
		}
	}
}

func TestRebuildMetadataDoesNotWriteComputedTopicStatus(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00001-topic")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	meta, _ := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Topic", Status: domain.StatusComplete, Created: "2026-09-21"}, "manual notes")
	if err := markdown.WriteFile(filepath.Join(topic, "_meta.md"), meta); err != nil {
		t.Fatal(err)
	}
	child, _ := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Child", Status: domain.StatusProgress, Created: "2026-09-21"}, "active")
	if err := markdown.WriteFile(filepath.Join(topic, "child.md"), child); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(topic, "_meta.md"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RebuildMetadata(workspace); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(topic, "_meta.md"))
	if err != nil {
		t.Fatal(err)
	}
	beforeDocument, err := markdown.Parse(filepath.Join(topic, "_meta.md"), before)
	if err != nil {
		t.Fatal(err)
	}
	afterDocument, err := markdown.Parse(filepath.Join(topic, "_meta.md"), after)
	if err != nil {
		t.Fatal(err)
	}
	if beforeDocument.Frontmatter.Status != afterDocument.Frontmatter.Status {
		t.Fatalf("topic status changed from %s to %s", beforeDocument.Frontmatter.Status, afterDocument.Frontmatter.Status)
	}
}

func contains(value, part string) bool {
	return len(value) >= len(part) && filepath.Base(part) != "" && strings.Contains(value, part)
}
