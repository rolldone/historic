package lifecycle

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"
)

func writeRecoveryTopic(t *testing.T, root, folder string, files map[string]markdown.Document) string {
	t.Helper()
	topic := filepath.Join(root, folder)
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	for path, document := range files {
		if err := markdown.WriteFile(filepath.Join(topic, filepath.FromSlash(path)), document); err != nil {
			t.Fatal(err)
		}
	}
	return topic
}

func TestRebuildRecoversMissingMetadataForOpenAndClosedTopics(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	workOrder, err := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Recovered WO", Status: domain.StatusComplete, Created: "2026-09-20"}, "body")
	if err != nil {
		t.Fatal(err)
	}
	_ = writeRecoveryTopic(t, workspace.Histories, "00001-open-topic", map[string]markdown.Document{"wos/01-wo.md": workOrder})
	closed := writeRecoveryTopic(t, workspace.Database, "00002-closed-topic", nil)
	if err := os.WriteFile(filepath.Join(closed, "diagram.pdf"), []byte("pdf"), 0o644); err != nil {
		t.Fatal(err)
	}

	change, err := RebuildMetadata(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if change.Topics != 2 || change.Updated != 2 {
		t.Fatalf("change = %#v", change)
	}
	for _, item := range []struct {
		root   string
		path   string
		files  int
		assets int
	}{
		{workspace.Histories, "00001-open-topic", 1, 0},
		{workspace.Database, "00002-closed-topic", 0, 1},
	} {
		metadata, readErr := markdown.ReadTopicMetadata(filepath.Join(item.root, item.path))
		if readErr != nil {
			t.Fatal(readErr)
		}
		if metadata.ID.String() != item.path[:5] || len(metadata.Files) != item.files || len(metadata.Assets) != item.assets {
			t.Fatalf("recovered metadata = %#v", metadata)
		}
	}
	database, err := sql.Open("sqlite", workspace.Index)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var recoveredStatus string
	if err := database.QueryRow("SELECT status FROM files WHERE topic_id = '00001' AND path = 'wos/01-wo.md'").Scan(&recoveredStatus); err != nil {
		t.Fatal(err)
	}
	if recoveredStatus != domain.StatusComplete.String() {
		t.Fatalf("recovered status = %q", recoveredStatus)
	}
}

func TestRebuildRecoveryRollbackPreservesMissingMetadataAndIndex(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	valid, err := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Valid", Status: domain.StatusProgress, Created: "2026-09-20"}, "body")
	if err != nil {
		t.Fatal(err)
	}
	validTopic := writeRecoveryTopic(t, workspace.Histories, "00001-valid", map[string]markdown.Document{"note.md": valid})
	if err := markdown.WriteTopicMetadata(filepath.Join(validTopic, markdown.MetaFilename), markdown.TopicMetadata{ID: "00001", Title: "Valid", Created: "2026-09-20"}); err != nil {
		t.Fatal(err)
	}
	if _, err := RebuildMetadata(workspace); err != nil {
		t.Fatal(err)
	}
	beforeIndex, err := os.ReadFile(workspace.Index)
	if err != nil {
		t.Fatal(err)
	}
	missingTopic := writeRecoveryTopic(t, workspace.Histories, "00002-missing", nil)
	invalidTopic := filepath.Join(workspace.Histories, "00003-invalid")
	if err := os.MkdirAll(invalidTopic, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(invalidTopic, markdown.MetaFilename), []byte("invalid"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := RebuildMetadata(workspace); err == nil {
		t.Fatal("rebuild unexpectedly succeeded with invalid metadata")
	}
	if _, err := os.Stat(filepath.Join(missingTopic, markdown.MetaFilename)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("recovered metadata was not rolled back: %v", err)
	}
	afterIndex, err := os.ReadFile(workspace.Index)
	if err != nil {
		t.Fatal(err)
	}
	if string(beforeIndex) != string(afterIndex) {
		t.Fatal("existing index changed after failed recovery")
	}
	if _, err := os.Stat(filepath.Join(validTopic, "note.md")); err != nil {
		t.Fatal(err)
	}
}

func TestRebuildRecoveryIsIdempotentWithoutManagedMarkdown(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := writeRecoveryTopic(t, workspace.Histories, "00001-assets-only", nil)
	if err := os.WriteFile(filepath.Join(topic, "readme.md"), []byte("plain asset"), 0o644); err != nil {
		t.Fatal(err)
	}
	first, err := RebuildMetadata(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if first.Updated != 1 {
		t.Fatalf("first rebuild = %#v", first)
	}
	before, err := os.ReadFile(filepath.Join(topic, markdown.MetaFilename))
	if err != nil {
		t.Fatal(err)
	}
	second, err := RebuildMetadata(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if second.Updated != 0 {
		t.Fatalf("second rebuild = %#v", second)
	}
	after, err := os.ReadFile(filepath.Join(topic, markdown.MetaFilename))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("idempotent recovery changed metadata")
	}
}
