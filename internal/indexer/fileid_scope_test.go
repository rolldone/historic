package indexer

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"
)

const sharedFileID = "01a1a8a1-6258-757e-95f0-fd5228110e62"

// seedTopicWithSharedFileID writes one topic folder whose manifest records the
// same FileID for shared.md. Legacy topic-ID migration preserves FileIDs, so
// two topics ending up with an identical FileID is a valid state.
func seedTopicWithSharedFileID(t *testing.T, root, folder string, topicID domain.ID, title string) {
	t.Helper()
	path := filepath.Join(root, folder)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	document, err := markdown.NewDocument(domain.Frontmatter{ID: topicID, Title: "Shared File", Status: domain.StatusProgress, Created: "2026-10-01"}, "shared body\n")
	if err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(filepath.Join(path, "shared.md"), document); err != nil {
		t.Fatal(err)
	}
	metadata := markdown.TopicMetadata{
		ID:      topicID,
		Title:   title,
		Created: "2026-10-01",
		Files:   []markdown.ManifestFile{{ID: domain.FileID(sharedFileID), Path: "shared.md", Type: "file", Status: domain.StatusProgress}},
		Assets:  []markdown.ManifestAsset{},
	}
	if err := markdown.WriteTopicMetadata(filepath.Join(path, markdown.MetaFilename), metadata); err != nil {
		t.Fatal(err)
	}
}

func countRows(t *testing.T, index, query string, args ...any) int {
	t.Helper()
	database, err := sql.Open("sqlite", index)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var count int
	if err := database.QueryRow(query, args...).Scan(&count); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return count
}

// TestRebuildAllowsSameFileIDAcrossTopics covers issue #1: two topics sharing a
// FileID must both be indexed instead of aborting the whole rebuild with
// "UNIQUE constraint failed: files.id, files.storage (1555)".
func TestRebuildAllowsSameFileIDAcrossTopics(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	seedTopicWithSharedFileID(t, workspace.Histories, "00001-alpha", "00001", "Alpha")
	seedTopicWithSharedFileID(t, workspace.Histories, "00002-beta", "00002", "Beta")

	if _, err := Rebuild(workspace); err != nil {
		t.Fatalf("rebuild with duplicate FileID: %v", err)
	}
	if rows := countRows(t, workspace.Index, "SELECT COUNT(*) FROM files WHERE id = ?", sharedFileID); rows != 2 {
		t.Fatalf("rows for shared FileID = %d, want 2 (one per topic)", rows)
	}
	if rows := countRows(t, workspace.Index, "SELECT COUNT(*) FROM files WHERE id = ? AND topic_id = '00001'", sharedFileID); rows != 1 {
		t.Fatalf("topic 00001 rows = %d, want 1", rows)
	}
	if rows := countRows(t, workspace.Index, "SELECT COUNT(*) FROM files WHERE id = ? AND topic_id = '00002'", sharedFileID); rows != 1 {
		t.Fatalf("topic 00002 rows = %d, want 1", rows)
	}
}

// TestRebuildIsIdempotentForSameFileID ensures the per-topic upsert makes a
// repeated rebuild a no-op instead of a constraint violation.
func TestRebuildIsIdempotentForSameFileID(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	seedTopicWithSharedFileID(t, workspace.Histories, "00001-alpha", "00001", "Alpha")
	seedTopicWithSharedFileID(t, workspace.Histories, "00002-beta", "00002", "Beta")

	first, err := Rebuild(workspace)
	if err != nil {
		t.Fatalf("first rebuild: %v", err)
	}
	second, err := Rebuild(workspace)
	if err != nil {
		t.Fatalf("second rebuild: %v", err)
	}
	if first != second {
		t.Fatalf("record counts differ: first=%d second=%d", first, second)
	}
	if rows := countRows(t, workspace.Index, "SELECT COUNT(*) FROM files WHERE id = ?", sharedFileID); rows != 2 {
		t.Fatalf("rows for shared FileID = %d, want 2", rows)
	}
}

// TestSchemaScopesFileIDPerTopic freezes the schema contract: the files primary
// key must be (topic_id, id, storage).
func TestSchemaScopesFileIDPerTopic(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	seedTopicWithSharedFileID(t, workspace.Histories, "00001-alpha", "00001", "Alpha")
	seedTopicWithSharedFileID(t, workspace.Histories, "00002-beta", "00002", "Beta")
	if _, err := Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	database, err := sql.Open("sqlite", workspace.Index)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var schema string
	if err := database.QueryRow("SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'files'").Scan(&schema); err != nil {
		t.Fatal(err)
	}
	if !containsNormalized(schema, "PRIMARY KEY (topic_id, id, storage)") {
		t.Fatalf("files primary key is not scoped per topic:\n%s", schema)
	}
	if containsNormalized(schema, "PRIMARY KEY (id, storage)") {
		t.Fatalf("files still uses the global FileID primary key:\n%s", schema)
	}
}

// TestRebuildDropsLegacyIndexRows verifies that a full rebuild does not retain
// stale rows from an earlier index (legacy path rows), because index is a
// rebuildable read model.
func TestRebuildDropsLegacyIndexRows(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	seedTopicWithSharedFileID(t, workspace.Histories, "00001-alpha", "00001", "Alpha")

	database, err := sql.Open("sqlite", workspace.Index)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO topics (id, num_padded, title, slug, path, storage, created_at)
		VALUES ('00001', '00001', 'Alpha', 'alpha', '.historic/00001-alpha', 'open', '2026-10-01')`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO files (id, storage, topic_id, type, path, filename, title, mtime, hash)
		VALUES (?, 'open', '00001', 'asset', 'legacy-file.md', 'legacy-file.md', 'Legacy', 'now', 'hash')`, sharedFileID); err != nil {
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	if rows := countRows(t, workspace.Index, "SELECT COUNT(*) FROM files WHERE path = 'legacy-file.md'"); rows != 0 {
		t.Fatalf("legacy index rows survived rebuild: %d", rows)
	}
}

func containsNormalized(value, needle string) bool {
	return len(needle) == 0 || len(value) > 0 && indexOfNormalized(value, needle) >= 0
}

func indexOfNormalized(value, needle string) int {
	normalize := func(input string) string {
		result := make([]byte, 0, len(input))
		space := false
		for i := 0; i < len(input); i++ {
			character := input[i]
			if character == ' ' || character == '\n' || character == '\t' || character == '\r' {
				if !space {
					result = append(result, ' ')
				}
				space = true
				continue
			}
			space = false
			result = append(result, character)
		}
		return string(result)
	}
	haystack, target := normalize(value), normalize(needle)
	for i := 0; i+len(target) <= len(haystack); i++ {
		if haystack[i:i+len(target)] == target {
			return i
		}
	}
	return -1
}
