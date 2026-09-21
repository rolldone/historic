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
		document, _ := markdown.NewDocument(domain.Frontmatter{ID: item.id, Title: "Topic", Status: item.status, Created: "2026-09-21"}, "manual notes")
		if err := markdown.WriteFile(filepath.Join(topic, markdown.MetaFilename), document); err != nil {
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
		meta, err := markdown.ParseTopicMetadataFile(filepath.Join(item.root, item.id.String()+"-topic", markdown.MetaFilename))
		if err != nil {
			t.Fatal(err)
		}
		if meta.Status != item.status || meta.ID != item.id || meta.Title != "Topic" {
			t.Fatalf("metadata = %#v", meta)
		}
		if _, err := os.Stat(filepath.Join(item.root, item.id.String()+"-topic", markdown.LegacyMetaFilename)); !os.IsNotExist(err) {
			t.Fatalf("legacy metadata remains: %v", err)
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
	if err := markdown.WriteFile(filepath.Join(topic, markdown.MetaFilename), meta); err != nil {
		t.Fatal(err)
	}
	child, _ := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Child", Status: domain.StatusProgress, Created: "2026-09-21"}, "active")
	if err := markdown.WriteFile(filepath.Join(topic, "child.md"), child); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(topic, markdown.MetaFilename))

	if err != nil {
		t.Fatal(err)
	}
	if _, err := RebuildMetadata(workspace); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(topic, markdown.MetaFilename))

	if err != nil {
		t.Fatal(err)
	}
	beforeMetadata, err := markdown.ParseTopicMetadata(filepath.Join(topic, markdown.MetaFilename), before)
	if err != nil {
		t.Fatal(err)
	}
	afterMetadata, err := markdown.ParseTopicMetadata(filepath.Join(topic, markdown.MetaFilename), after)
	if err != nil {
		t.Fatal(err)
	}
	if beforeMetadata.Status != afterMetadata.Status {
		t.Fatalf("topic status changed from %s to %s", beforeMetadata.Status, afterMetadata.Status)
	}
}

func contains(value, part string) bool {
	return len(value) >= len(part) && filepath.Base(part) != "" && strings.Contains(value, part)
}
