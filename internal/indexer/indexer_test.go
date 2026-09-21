package indexer

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"
)

func writeIndexerDocument(t *testing.T, path string, metadata domain.Frontmatter, body string) {
	t.Helper()
	document, err := markdown.NewDocument(metadata, body)
	if err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(path, document); err != nil {
		t.Fatal(err)
	}
}

func writeIndexerTopicMetadata(t *testing.T, path string, metadata domain.Frontmatter) {
	t.Helper()
	if err := markdown.WriteTopicMetadata(path, markdown.TopicMetadataFromFrontmatter(metadata, "")); err != nil {
		t.Fatal(err)
	}
}

func TestRebuildUsesTopicIdentityAndLogicalPaths(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	openTopic := filepath.Join(workspace.Histories, "00001-renamed-topic")
	closedTopic := filepath.Join(workspace.Database, "00001-old-topic")
	if err := os.MkdirAll(filepath.Join(openTopic, "wos"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(closedTopic, "wos"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeIndexerTopicMetadata(t, filepath.Join(openTopic, markdown.MetaFilename), domain.Frontmatter{ID: "00001", Title: "Renamed Topic", Status: domain.StatusProgress, Created: "2026-09-21"})
	writeIndexerDocument(t, filepath.Join(openTopic, "wos", "task.md"), domain.Frontmatter{ID: "00001", Title: "Current Task", Status: domain.StatusProgress, Created: "2026-09-21"}, "current task body")
	writeIndexerTopicMetadata(t, filepath.Join(closedTopic, markdown.MetaFilename), domain.Frontmatter{ID: "00001", Title: "Old Snapshot", Status: domain.StatusComplete, Created: "2026-09-21"})
	writeIndexerDocument(t, filepath.Join(closedTopic, "wos", "task.md"), domain.Frontmatter{ID: "00001", Title: "Old Task", Status: domain.StatusComplete, Created: "2026-09-21"}, "old task body")

	if _, err := Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	database, err := sql.Open("sqlite", workspace.Index)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var topicCount int
	if err := database.QueryRow("SELECT COUNT(*) FROM topics WHERE id = '00001'").Scan(&topicCount); err != nil {
		t.Fatal(err)
	}
	if topicCount != 1 {
		t.Fatalf("logical topic count = %d, want 1", topicCount)
	}
	var slug, physicalPath, storage string
	if err := database.QueryRow("SELECT slug, path, storage FROM topics WHERE id = '00001'").Scan(&slug, &physicalPath, &storage); err != nil {
		t.Fatal(err)
	}
	if slug != "renamed-topic" || physicalPath != ".historic/00001-renamed-topic" || storage != "open" {
		t.Fatalf("topic identity = slug %q path %q storage %q", slug, physicalPath, storage)
	}
	var logicalPath string
	if err := database.QueryRow("SELECT path FROM files WHERE topic_id = '00001' AND path = 'wos/task.md'").Scan(&logicalPath); err != nil {
		t.Fatal(err)
	}
	if logicalPath != "wos/task.md" {
		t.Fatalf("logical file path = %q", logicalPath)
	}
	var fileCount int
	if err := database.QueryRow("SELECT COUNT(*) FROM files WHERE topic_id = '00001'").Scan(&fileCount); err != nil {
		t.Fatal(err)
	}
	if fileCount != 1 {
		t.Fatalf("logical child file count = %d, want 1", fileCount)
	}
	var oldSnapshotCount int
	if err := database.QueryRow("SELECT COUNT(*) FROM files WHERE content LIKE '%old snapshot%'").Scan(&oldSnapshotCount); err != nil {
		t.Fatal(err)
	}
	if oldSnapshotCount != 0 {
		t.Fatalf("closed snapshot was indexed as a duplicate: %d", oldSnapshotCount)
	}
}

func TestRebuildRejectsDuplicateTopicIDsInSameRoot(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, folder := range []string{"00001-first", "00001-second"} {
		topic := filepath.Join(workspace.Histories, folder)
		if err := os.MkdirAll(topic, 0o755); err != nil {
			t.Fatal(err)
		}
		writeIndexerTopicMetadata(t, filepath.Join(topic, markdown.MetaFilename), domain.Frontmatter{ID: "00001", Title: folder, Status: domain.StatusProgress, Created: "2026-09-21"})
	}
	if _, err := Rebuild(workspace); err == nil || !errors.Is(err, domain.ErrConflict) || !strings.Contains(err.Error(), "duplicate topic ID 00001") {
		t.Fatalf("duplicate rebuild error = %v", err)
	}
}

func TestRebuildKeepsNestedLogicalPathWithinTopic(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00001-topic")
	if err := os.MkdirAll(filepath.Join(topic, "wos"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeIndexerTopicMetadata(t, filepath.Join(topic, markdown.MetaFilename), domain.Frontmatter{ID: "00001", Title: "Topic", Status: domain.StatusProgress, Created: "2026-09-21"})
	writeIndexerDocument(t, filepath.Join(topic, "wos", "task.md"), domain.Frontmatter{ID: "00001", Title: "Task", Status: domain.StatusProgress, Created: "2026-09-21"}, "task")
	if _, err := Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	database, err := sql.Open("sqlite", workspace.Index)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var path string
	if err := database.QueryRow("SELECT path FROM files WHERE topic_id = '00001'").Scan(&path); err != nil {
		t.Fatal(err)
	}
	if path != "wos/task.md" {
		t.Fatalf("path = %q, want wos/task.md", path)
	}
	if strings.Contains(path, ".historic") || strings.Contains(path, "..") || filepath.IsAbs(path) {
		t.Fatalf("path escaped topic boundary: %q", path)
	}
}

func TestRebuildIndexesDescriptionAndFTS(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00001-topic")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, description := range map[string]string{
		"_meta.yaml": "Topic overview",
		"note.md":    "File details",
	} {
		if name == markdown.MetaFilename {
			if err := markdown.WriteTopicMetadata(filepath.Join(topic, name), markdown.TopicMetadata{ID: "00001", Title: name, Description: description, Status: domain.StatusProgress, Created: "2026-09-21"}); err != nil {
				t.Fatal(err)
			}
			continue
		}
		document, err := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: name, Description: description, Status: domain.StatusProgress, Created: "2026-09-21"}, "body")
		if err != nil {
			t.Fatal(err)
		}
		if err := markdown.WriteFile(filepath.Join(topic, name), document); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	database, err := sql.Open("sqlite", workspace.Index)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	rows, err := database.Query("SELECT filename, description FROM index_records ORDER BY filename")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	got := map[string]string{}
	for rows.Next() {
		var filename string
		var description sql.NullString
		if err := rows.Scan(&filename, &description); err != nil {
			t.Fatal(err)
		}
		got[filename] = description.String
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if got["_meta.yaml"] != "Topic overview" || got["note.md"] != "File details" {
		t.Fatalf("indexed descriptions = %#v", got)
	}
	for _, keyword := range []string{"Topic overview", "File details"} {
		var count int
		if err := database.QueryRow("SELECT COUNT(*) FROM historic_fts WHERE historic_fts MATCH ?", keyword).Scan(&count); err != nil {
			t.Fatalf("FTS %q: %v", keyword, err)
		}
		if count != 1 {
			t.Fatalf("FTS %q count = %d, want 1", keyword, count)
		}
	}
}

func TestRebuildIndexesMarkdownAndIsRepeatable(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00001-topic")
	if err := os.MkdirAll(filepath.Join(topic, "wos"), 0o755); err != nil {
		t.Fatal(err)
	}
	metadata := domain.Frontmatter{ID: "00001", Title: "Topic", Status: domain.StatusProgress, Created: "2026-09-18", Tags: []string{"one"}}
	for name := range map[string]string{markdown.MetaFilename: "meta body", "prd.md": "prd body", "wos/01-task.md": "task body"} {
		if name == markdown.MetaFilename {
			if err := markdown.WriteTopicMetadata(filepath.Join(topic, name), markdown.TopicMetadataFromFrontmatter(metadata, "")); err != nil {
				t.Fatal(err)
			}
			continue
		}
		document, err := markdown.NewDocument(metadata, name+" body")
		if err != nil {
			t.Fatal(err)
		}
		if err := markdown.WriteFile(filepath.Join(topic, name), document); err != nil {
			t.Fatal(err)
		}
	}
	count, err := Rebuild(workspace)
	if err != nil || count != 3 {
		t.Fatalf("Rebuild = %d, %v", count, err)
	}
	indexed, err := Count(workspace)
	if err != nil || indexed != 3 {
		t.Fatalf("Count = %d, %v", indexed, err)
	}
	if _, err := Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	indexed, err = Count(workspace)
	if err != nil || indexed != 3 {
		t.Fatalf("second Count = %d, %v", indexed, err)
	}
}

func TestRebuildPreservesOldIndexWhenScanFails(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00001-topic")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	valid, _ := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Valid", Status: domain.StatusProgress, Created: "2026-09-18"}, "valid")
	if err := markdown.WriteFile(filepath.Join(topic, "note.md"), valid); err != nil {
		t.Fatal(err)
	}
	if _, err := Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(topic, "broken.md"), []byte("no frontmatter"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Rebuild(workspace); err != nil {
		t.Fatalf("rebuild with Markdown asset failed: %v", err)
	}
	count, err := Count(workspace)
	if err != nil || count != 1 {
		t.Fatalf("old index changed: count=%d err=%v", count, err)
	}
}

func TestRebuildCanRecreateDeletedIndex(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(workspace.Index); err != nil {
		t.Fatal(err)
	}
	if _, err := Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(workspace.Index); err != nil {
		t.Fatal(err)
	}
}

func TestRebuildReadModelClassifiesAssetsAndAggregatesStatus(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00001-topic")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	writeIndexerTopicMetadata(t, filepath.Join(topic, markdown.MetaFilename), domain.Frontmatter{ID: "00001", Title: "Topic", Status: domain.StatusComplete, Created: "2026-09-21"})
	writeIndexerDocument(t, filepath.Join(topic, "active.md"), domain.Frontmatter{ID: "00001", Title: "Active", Status: domain.StatusProgress, Created: "2026-09-21"}, "active")
	writeIndexerDocument(t, filepath.Join(topic, "done.md"), domain.Frontmatter{ID: "00001", Title: "Done", Status: domain.StatusComplete, Created: "2026-09-21"}, "done")
	if err := os.WriteFile(filepath.Join(topic, "brief.md"), []byte("plain Markdown asset"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	database, err := sql.Open("sqlite", workspace.Index)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var computed string
	if err := database.QueryRow("SELECT computed_status FROM topics WHERE id = '00001'").Scan(&computed); err != nil {
		t.Fatal(err)
	}
	if computed != domain.StatusProgress.String() {
		t.Fatalf("computed status = %q, want progress", computed)
	}
	var managed, assets, nullAssets int
	if err := database.QueryRow("SELECT COUNT(*) FROM files WHERE topic_id = '00001' AND type = 'historic_file'").Scan(&managed); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM files WHERE topic_id = '00001' AND type = 'asset'").Scan(&assets); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM files WHERE topic_id = '00001' AND type = 'asset' AND status IS NULL").Scan(&nullAssets); err != nil {
		t.Fatal(err)
	}
	if managed != 2 || assets != 1 || nullAssets != 1 {
		t.Fatalf("read model counts managed=%d assets=%d nullAssets=%d", managed, assets, nullAssets)
	}
}

func TestRebuildRejectsInvalidTopicMetadataAndPreservesIndex(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00001-topic")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	document, _ := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Valid", Status: domain.StatusProgress, Created: "2026-09-18"}, "valid")
	if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename), markdown.TopicMetadataFromFrontmatter(document.Frontmatter, "")); err != nil {
		t.Fatal(err)
	}
	if _, err := Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(topic, markdown.MetaFilename))

	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(topic, markdown.MetaFilename), []byte("invalid metadata"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := RebuildWithPreparation(workspace, func() error { return nil }); err == nil {
		t.Fatal("invalid topic metadata rebuild succeeded")
	}
	count, err := Count(workspace)
	if err != nil || count != 1 {
		t.Fatalf("old index changed: count=%d err=%v", count, err)
	}
	if err := os.WriteFile(filepath.Join(topic, markdown.MetaFilename), before, 0o644); err != nil {

		t.Fatal(err)
	}
}

func TestIndexDatabaseHasExpectedRecords(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	database, err := sql.Open("sqlite", workspace.Index)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var table string
	if err := database.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='index_records'").Scan(&table); err != nil {
		t.Fatal(err)
	}
	if table != "index_records" {
		t.Fatal(errors.New("index table missing"))
	}
	for _, name := range []string{"topics", "files"} {
		if err := database.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", name).Scan(&table); err != nil {
			t.Fatalf("read model table %q missing: %v", name, err)
		}
		if table != name {
			t.Fatalf("read model table = %q, want %q", table, name)
		}
	}
	var foreignKeys int
	if err := database.QueryRow("SELECT COUNT(*) FROM pragma_foreign_key_list('files') WHERE \"table\"='topics' AND \"from\"='topic_id' AND \"to\"='id'").Scan(&foreignKeys); err != nil {
		t.Fatal(err)
	}
	if foreignKeys != 1 {
		t.Fatalf("files.topic_id foreign key count = %d, want 1", foreignKeys)
	}
	var fts string
	if err := database.QueryRow("SELECT sql FROM sqlite_master WHERE name='historic_fts'").Scan(&fts); err != nil {
		t.Fatalf("FTS5 table missing: %v", err)
	}
	if !strings.Contains(strings.ToLower(fts), "using fts5") {
		t.Fatalf("historic_fts is not FTS5: %q", fts)
	}
	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM historic_fts").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("empty FTS5 table count = %d", count)
	}
}

func TestRebuildPreservesFTSIndexWhenScanFails(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00001-topic")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	document, _ := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Valid", Status: domain.StatusProgress, Created: "2026-09-18"}, "searchable body")
	if err := markdown.WriteFile(filepath.Join(topic, "note.md"), document); err != nil {
		t.Fatal(err)
	}
	if _, err := Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(topic, "broken.md"), []byte("invalid"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Rebuild(workspace); err != nil {
		t.Fatal("invalid Markdown asset should not fail rebuild")
	}
	database, err := sql.Open("sqlite", workspace.Index)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM historic_fts WHERE content MATCH 'searchable'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("preserved FTS result count = %d", count)
	}
}
