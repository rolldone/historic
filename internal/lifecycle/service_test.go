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

func TestChangeStatusRejectsTopicStatusMutation(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewService(workspace).ChangeStatus("00001", domain.StatusComplete); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("topic status mutation error = %v", err)
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
	if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename), markdown.TopicMetadataFromFrontmatter(document.Frontmatter, "")); err != nil {
		t.Fatal(err)
	}
	if _, err := NewService(workspace).ChangeStatus("00001", domain.StatusProgress); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("closed transition error = %v", err)
	}
	if _, err := NewService(workspace).ChangeStatus("00099", domain.StatusProgress); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("topic status mutation error = %v", err)
	}
}
