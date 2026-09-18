package lifecycle

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"
)

func TestChangeStatusUpdatesMetadataAndIndex(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00001-topic")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	document, _ := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Topic", Status: domain.StatusCreate, Created: "2026-09-18"}, "body")
	if err := markdown.WriteFile(filepath.Join(topic, "_meta.md"), document); err != nil {
		t.Fatal(err)
	}
	change, err := NewService(workspace).ChangeStatus("00001", domain.StatusProgress)
	if err != nil {
		t.Fatal(err)
	}
	if change.Previous != domain.StatusCreate || change.Current != domain.StatusProgress || change.Archived {
		t.Fatalf("change = %#v", change)
	}
	updated, err := markdown.ParseFile(filepath.Join(topic, "_meta.md"))
	if err != nil || updated.Frontmatter.Status != domain.StatusProgress || updated.Frontmatter.Updated == "" {
		t.Fatalf("metadata = %#v err=%v", updated.Frontmatter, err)
	}
}

func TestChangeStatusRejectsInvalidAndClosedTransitions(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00001-topic")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	document, _ := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Topic", Status: domain.StatusComplete, Created: "2026-09-18"}, "body")
	if err := markdown.WriteFile(filepath.Join(topic, "_meta.md"), document); err != nil {
		t.Fatal(err)
	}
	if _, err := NewService(workspace).ChangeStatus("00001", domain.StatusProgress); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("closed transition error = %v", err)
	}
	if _, err := NewService(workspace).ChangeStatus("00099", domain.StatusProgress); !errors.Is(err, domain.ErrTopicMissing) {
		t.Fatalf("missing topic error = %v", err)
	}
}
