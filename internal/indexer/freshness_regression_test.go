package indexer_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/lifecycle"
	"historic/internal/markdown"
)

func TestRebuildRemovesDeletedFileFromManifestSQLiteAndFTS(t *testing.T) {
	workspace := config.NewWorkspace(t.TempDir())
	if _, err := config.Initialize(workspace.Root); err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00001-calibration")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	meta := markdown.TopicMetadata{ID: domain.ID("00001"), Title: "Calibration", Created: "2026-09-22", Files: []markdown.ManifestFile{}, Assets: []markdown.ManifestAsset{}}
	if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename), meta); err != nil {
		t.Fatal(err)
	}
	doc, err := markdown.NewDocument(domain.Frontmatter{ID: domain.ID("00001"), Title: "Keep", Status: domain.StatusProgress, Created: "2026-09-22"}, "keep-token")
	if err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(topic, "keep.md")
	remove := filepath.Join(topic, "remove.md")
	if err := markdown.WriteFile(keep, doc); err != nil {
		t.Fatal(err)
	}
	removeDoc, _ := markdown.NewDocument(domain.Frontmatter{ID: domain.ID("00001"), Title: "Remove", Status: domain.StatusProgress, Created: "2026-09-22"}, "remove-token")
	if err := markdown.WriteFile(remove, removeDoc); err != nil {
		t.Fatal(err)
	}
	if _, err := lifecycle.RebuildMetadata(workspace); err != nil {
		t.Fatal(err)
	}
	before, err := markdown.ReadTopicMetadata(topic)
	if err != nil || len(before.Files) != 2 {
		t.Fatalf("before metadata = %+v, %v", before, err)
	}
	var keepID domain.FileID
	for _, file := range before.Files {
		if file.Path == "keep.md" {
			keepID = file.ID
		}
	}
	if err := os.Remove(remove); err != nil {
		t.Fatal(err)
	}
	if _, err := lifecycle.RebuildMetadata(workspace); err != nil {
		t.Fatal(err)
	}
	after, err := markdown.ReadTopicMetadata(topic)
	if err != nil || len(after.Files) != 1 || after.Files[0].Path != "keep.md" {
		t.Fatalf("after metadata = %+v, %v", after, err)
	}
	if after.Files[0].ID != keepID {
		t.Fatalf("keep FileID changed: %s -> %s", keepID, after.Files[0].ID)
	}
	database, err := sql.Open("sqlite", workspace.Index)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var files, fts int
	if err := database.QueryRow("SELECT COUNT(*) FROM files WHERE path = 'remove.md'").Scan(&files); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM historic_fts WHERE path = 'remove.md'").Scan(&fts); err != nil {
		t.Fatal(err)
	}
	if files != 0 || fts != 0 {
		t.Fatalf("deleted file remained: files=%d fts=%d", files, fts)
	}
	if _, err := lifecycle.RebuildMetadata(workspace); err != nil {
		t.Fatal(err)
	}
	afterSecond, err := markdown.ReadTopicMetadata(topic)
	if err != nil || len(afterSecond.Files) != 1 || afterSecond.Files[0].ID != keepID {
		t.Fatalf("second rebuild changed metadata: %+v, %v", afterSecond, err)
	}
}
