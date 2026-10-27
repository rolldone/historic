package lifecycle

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"
)

func withFrozenStatusNow(t *testing.T, value time.Time) {
	t.Helper()
	previous := fileStatusNow
	fileStatusNow = func() time.Time { return value }
	t.Cleanup(func() { fileStatusNow = previous })
}

// writeUTCStatusTopic creates one open topic with a single managed task file.
func writeUTCStatusTopic(t *testing.T, workspace config.Workspace, topicID, created string) (string, string) {
	t.Helper()
	topic := filepath.Join(workspace.Histories, topicID+"-utc")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename),
		markdown.TopicMetadata{ID: domain.ID(topicID), Title: "UTC", Created: "2026-10-01"}); err != nil {
		t.Fatal(err)
	}
	document, err := markdown.NewDocument(
		domain.Frontmatter{ID: domain.ID(topicID), Title: "Task", Status: domain.StatusProgress, Created: created}, "body")
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(topic, "task.md")
	if err := markdown.WriteFile(target, document); err != nil {
		t.Fatal(err)
	}
	return topic, target
}

func TestChangeFileStatusWritesUTCDate(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// 2026-10-28 03:00 in UTC+7 is still 2026-10-27 in UTC.
	withFrozenStatusNow(t, time.Date(2026, 10, 28, 3, 0, 0, 0, time.FixedZone("WIB", 7*60*60)))
	_, target := writeUTCStatusTopic(t, workspace, "00021", "2026-10-20")

	change, err := ChangeFileStatus(workspace, "task.md", domain.StatusReview)
	if err != nil {
		t.Fatal(err)
	}
	if change.Updated != "2026-10-27" {
		t.Fatalf("change.Updated = %q, want 2026-10-27", change.Updated)
	}
	document, err := markdown.ParseFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if document.Frontmatter.Updated != "2026-10-27" {
		t.Fatalf("frontmatter updated = %q, want 2026-10-27", document.Frontmatter.Updated)
	}
	if document.Frontmatter.Updated < document.Frontmatter.Created {
		t.Fatalf("updated %q precedes created %q", document.Frontmatter.Updated, document.Frontmatter.Created)
	}
}

func TestChangeFileStatusNeverMovesUpdatedBeforeCreated(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// A legacy document created with a local-time date ahead of the UTC date.
	withFrozenStatusNow(t, time.Date(2026, 10, 27, 20, 0, 0, 0, time.UTC))
	_, target := writeUTCStatusTopic(t, workspace, "00022", "2026-10-28")

	change, err := ChangeFileStatus(workspace, "task.md", domain.StatusProgress)
	if err != nil {
		t.Fatal(err)
	}
	if change.Updated != "2026-10-28" {
		t.Fatalf("change.Updated = %q, want 2026-10-28 (created floor)", change.Updated)
	}
	document, err := markdown.ParseFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if document.Frontmatter.Updated < document.Frontmatter.Created {
		t.Fatalf("updated %q precedes created %q", document.Frontmatter.Updated, document.Frontmatter.Created)
	}
}

func TestRecoveredTopicMetadataKeepsUpdatedAfterCreated(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00023-recovered")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	// No _meta.yaml: recovery derives metadata from the managed files.
	document, err := markdown.NewDocument(domain.Frontmatter{
		ID: "00023", Title: "Recovered", Status: domain.StatusProgress,
		Created: "2026-10-28", Updated: "2026-10-27",
	}, "body")
	if err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(filepath.Join(topic, "note.md"), document); err != nil {
		t.Fatal(err)
	}
	change, err := syncMetaTopic(workspace, workspace.RelativePath(topic), false)
	if err != nil {
		t.Fatalf("sync metadata: %v", err)
	}
	if !change.Updated {
		t.Fatalf("expected metadata write, got %#v", change)
	}
	meta, err := markdown.ReadTopicMetadata(topic)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Updated < meta.Created {
		t.Fatalf("recovered updated %q precedes created %q", meta.Updated, meta.Created)
	}
}
