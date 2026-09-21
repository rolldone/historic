package lifecycle

import (
	"os"
	"path/filepath"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"
)

func TestSyncMetaBatchProcessesActiveTopicsDeterministically(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, folder := range []string{"00002-second", "00001-first"} {
		topic := filepath.Join(workspace.Histories, folder)
		if err := os.MkdirAll(topic, 0o755); err != nil {
			t.Fatal(err)
		}
		id, _ := domain.ParseID(folder[:5])
		metadata := markdown.TopicMetadata{ID: id, Title: folder, Status: domain.StatusProgress, Created: "2026-09-19"}
		if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename), metadata); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(topic, "plain.txt"), []byte("asset"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	archived := filepath.Join(workspace.Database, "00003-archived")
	if err := os.MkdirAll(archived, 0o755); err != nil {
		t.Fatal(err)
	}
	change := SyncMetaBatch(workspace)
	if change.Total != 2 || change.Errors != 0 || len(change.Topics) != 2 {
		t.Fatalf("batch = %#v", change)
	}
	if change.Topics[0].ID != "00001" || change.Topics[1].ID != "00002" {
		t.Fatalf("ordering = %#v", change.Topics)
	}
	for _, topic := range change.Topics {
		if topic.Assets != 1 || topic.Updated {
			t.Fatalf("topic result = %#v", topic)
		}
	}
	second := SyncMetaBatch(workspace)
	if second.Updated != 0 || second.Errors != 0 {
		t.Fatalf("idempotent batch = %#v", second)
	}
}

func TestSyncMetaBatchContinuesAfterTopicError(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, folder := range []string{"00001-broken", "00002-good"} {
		topic := filepath.Join(workspace.Histories, folder)
		if err := os.MkdirAll(topic, 0o755); err != nil {
			t.Fatal(err)
		}
		id, _ := domain.ParseID(folder[:5])
		if folder == "00002-good" {
			metadata := markdown.TopicMetadata{ID: id, Title: folder, Status: domain.StatusProgress, Created: "2026-09-19"}
			if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename), metadata); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.WriteFile(filepath.Join(topic, markdown.LegacyMetaFilename), []byte("invalid"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	change := SyncMetaBatch(workspace)
	if change.Total != 2 || change.Errors != 2 || len(change.Topics) != 2 {
		t.Fatalf("batch error result = %#v", change)
	}
	if change.Topics[0].Error == "" || change.Topics[1].Error == "" {
		t.Fatalf("continue-on-error results = %#v", change.Topics)
	}
	if _, err := os.Stat(filepath.Join(workspace.Histories, "00001-broken", markdown.LegacyMetaFilename)); err != nil {
		t.Fatalf("invalid legacy metadata was not preserved: %v", err)
	}
}

func TestActiveTopicPathsIgnoreNonTopicsAndEmptyWorkspace(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace.Histories, "README.md"), []byte("not topic"), 0o644); err != nil {
		t.Fatal(err)
	}
	paths, err := activeTopicPaths(workspace)
	if err != nil || len(paths) != 0 {
		t.Fatalf("paths = %v, err=%v", paths, err)
	}
}
