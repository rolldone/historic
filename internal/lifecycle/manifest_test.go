package lifecycle

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"
)

func TestRebuildWritesManifestPreservesManualMetadataAndClassifiesLegacyAsset(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		id    domain.ID
		title string
	}{
		{id: "00003", title: "Topic Storage Open Close"},
		{id: "00005", title: "Search Read Model v2"},
	} {
		topic := filepath.Join(workspace.Histories, item.id.String()+"-topic")
		if err := os.MkdirAll(filepath.Join(topic, "wos"), 0o755); err != nil {
			t.Fatal(err)
		}
		metadata := markdown.TopicMetadata{
			ID: item.id, Title: item.title, Description: "manual description", Status: domain.StatusProgress,
			Created: "2026-09-21", Updated: "2026-09-22", Tags: []string{"manual", "search"}, Related: []domain.ID{"00003"},
		}
		if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename), metadata); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(topic, "_meta.md"), []byte("legacy metadata is an asset"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(topic, "diagram.pdf"), []byte("pdf"), 0o644); err != nil {
			t.Fatal(err)
		}
		entry, err := markdown.NewDocument(domain.Frontmatter{ID: item.id, Title: "Task", Status: domain.StatusProgress, Created: "2026-09-21"}, "task")
		if err != nil {
			t.Fatal(err)
		}
		if err := markdown.WriteFile(filepath.Join(topic, "wos", "01-task.md"), entry); err != nil {
			t.Fatal(err)
		}
	}

	change, err := RebuildMetadata(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if change.Updated != 2 {
		t.Fatalf("updated topics = %d, want 2", change.Updated)
	}
	database, err := sql.Open("sqlite", workspace.Index)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var legacyAssets, canonicalChildren int
	if err := database.QueryRow("SELECT COUNT(*) FROM files WHERE path = '_meta.md' AND type = 'asset'").Scan(&legacyAssets); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM files WHERE path = '_meta.yaml'").Scan(&canonicalChildren); err != nil {
		t.Fatal(err)
	}
	if legacyAssets != 2 || canonicalChildren != 0 {
		t.Fatalf("metadata read model legacy assets=%d canonical children=%d", legacyAssets, canonicalChildren)
	}
	for _, item := range []struct {
		id     domain.ID
		files  int
		assets int
	}{
		{id: "00003", files: 1, assets: 2},
		{id: "00005", files: 1, assets: 2},
	} {
		topic := filepath.Join(workspace.Histories, item.id.String()+"-topic")
		metadata, err := markdown.ParseTopicMetadataFile(filepath.Join(topic, markdown.MetaFilename))
		if err != nil {
			t.Fatal(err)
		}
		if metadata.Description != "manual description" || metadata.Status != domain.StatusProgress || metadata.Updated != "2026-09-22" || len(metadata.Tags) != 2 || len(metadata.Related) != 1 {
			t.Fatalf("manual metadata was not preserved: %#v", metadata)
		}
		if len(metadata.Files) != item.files || metadata.Files[0].Path != "wos/01-task.md" || metadata.Files[0].Type != "task" || metadata.Files[0].Status != domain.StatusProgress {
			t.Fatalf("files manifest = %#v", metadata.Files)
		}
		if len(metadata.Assets) != item.assets || metadata.Assets[0].Path != "_meta.md" || metadata.Assets[0].Type != "markdown" {
			t.Fatalf("assets manifest = %#v", metadata.Assets)
		}
	}
}

func TestRebuildManifestUpdatesAfterAddDeleteAndClassificationAndIsIdempotent(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00005-topic")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename), markdown.TopicMetadata{ID: "00005", Title: "Topic", Status: domain.StatusProgress, Created: "2026-09-21"}); err != nil {
		t.Fatal(err)
	}
	if _, err := RebuildMetadata(workspace); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(filepath.Join(topic, markdown.MetaFilename))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RebuildMetadata(workspace); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(filepath.Join(topic, markdown.MetaFilename))
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("idempotent rebuild changed an unchanged manifest")
	}

	entry, err := markdown.NewDocument(domain.Frontmatter{ID: "00005", Title: "Added", Status: domain.StatusComplete, Created: "2026-09-21"}, "added")
	if err != nil {
		t.Fatal(err)
	}
	added := filepath.Join(topic, "added.md")
	if err := markdown.WriteFile(added, entry); err != nil {
		t.Fatal(err)
	}
	if _, err := RebuildMetadata(workspace); err != nil {
		t.Fatal(err)
	}
	metadata, err := markdown.ReadTopicMetadata(topic)
	if err != nil {
		t.Fatal(err)
	}
	if len(metadata.Files) != 1 || metadata.Files[0].Path != "added.md" {
		t.Fatalf("after add manifest = %#v", metadata.Files)
	}
	if err := os.WriteFile(added, []byte("now an asset"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := RebuildMetadata(workspace); err != nil {
		t.Fatal(err)
	}
	metadata, err = markdown.ReadTopicMetadata(topic)
	if err != nil {
		t.Fatal(err)
	}
	if len(metadata.Files) != 0 || len(metadata.Assets) != 1 || metadata.Assets[0].Path != "added.md" {
		t.Fatalf("after classification manifest = %#v", metadata)
	}
	if err := os.Remove(added); err != nil {
		t.Fatal(err)
	}
	if _, err := RebuildMetadata(workspace); err != nil {
		t.Fatal(err)
	}
	metadata, err = markdown.ReadTopicMetadata(topic)
	if err != nil {
		t.Fatal(err)
	}
	if len(metadata.Files) != 0 || len(metadata.Assets) != 0 {
		t.Fatalf("after delete manifest = %#v", metadata)
	}
}
