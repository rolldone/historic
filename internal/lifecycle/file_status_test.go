package lifecycle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/indexer"
	"historic/internal/markdown"
)

func TestChangeFileStatusUpdatesOnlyTargetAndRebuilds(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00001-topic")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	metadata := domain.Frontmatter{ID: "00001", Title: "Topic", Status: domain.StatusProgress, Created: "2026-09-19"}
	if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename), markdown.TopicMetadataFromFrontmatter(metadata, "")); err != nil {
		t.Fatal(err)
	}
	target, err := markdown.NewDocument(metadata, "target body")
	if err != nil {
		t.Fatal(err)
	}
	other, err := markdown.NewDocument(metadata, "other body")
	if err != nil {
		t.Fatal(err)
	}
	targetPath := filepath.Join(topic, "19-fts5-index.md")
	otherPath := filepath.Join(topic, "20-query.md")
	if err := markdown.WriteFile(targetPath, target); err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(otherPath, other); err != nil {
		t.Fatal(err)
	}
	beforeOther, err := os.ReadFile(otherPath)
	if err != nil {
		t.Fatal(err)
	}

	change, err := ChangeFileStatus(workspace, "19-fts5-index.md", domain.StatusComplete)
	if err != nil {
		t.Fatal(err)
	}
	if change.Previous != domain.StatusProgress || change.Current != domain.StatusComplete || change.Path != ".historic/00001-topic/19-fts5-index.md" {
		t.Fatalf("change = %#v", change)
	}
	if change.Updated == "" {
		t.Fatal("updated date is empty")
	}
	updated, err := markdown.ParseFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Frontmatter.Status != domain.StatusComplete || updated.Frontmatter.Updated != change.Updated {
		t.Fatalf("updated frontmatter = %#v", updated.Frontmatter)
	}
	afterOther, err := os.ReadFile(otherPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(afterOther) != string(beforeOther) {
		t.Fatal("non-target file changed")
	}
	if indexed, err := indexer.Count(workspace); err != nil || indexed != 3 {
		t.Fatalf("index count = %d, %v", indexed, err)
	}
}

func TestChangeFileStatusRejectsUnsafeAndInvalidTargets(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"../escape.md", "/tmp/absolute.md", "missing.txt"} {
		if _, err := ChangeFileStatus(workspace, path, domain.StatusComplete); err == nil {
			t.Fatalf("unsafe target %q accepted", path)
		}
	}
	topic := filepath.Join(workspace.Histories, "00001-topic")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(topic, "broken.md"), []byte("not frontmatter"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ChangeFileStatus(workspace, "broken.md", domain.StatusComplete); err == nil || !strings.Contains(err.Error(), "frontmatter") {
		t.Fatalf("invalid Markdown error = %v", err)
	}
}
