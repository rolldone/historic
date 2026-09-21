package lifecycle

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"

	_ "modernc.org/sqlite"
)

func TestWO14UnifiedRebuildSmokeUsesCanonicalMetadataAndReadModel(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	openTopic := filepath.Join(workspace.Histories, "00001-open-topic")
	closedTopic := filepath.Join(workspace.Database, "00002-closed-topic")
	for _, topic := range []string{openTopic, closedTopic} {
		if err := os.MkdirAll(filepath.Join(topic, "wos"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeWO14Metadata(t, openTopic, markdown.TopicMetadata{ID: "00001", Title: "Open Topic", Status: domain.StatusComplete, Created: "2026-09-21", Description: "open description"})
	writeWO14Metadata(t, closedTopic, markdown.TopicMetadata{ID: "00002", Title: "Closed Topic", Status: domain.StatusProgress, Created: "2026-09-21", Description: "closed description"})
	writeWO14File(t, filepath.Join(openTopic, "wos", "01-active.md"), domain.Frontmatter{ID: "00001", Title: "Active task", Description: "active description", Status: domain.StatusProgress, Created: "2026-09-21"}, "active body")
	writeWO14File(t, filepath.Join(openTopic, "wos", "02-done.md"), domain.Frontmatter{ID: "00001", Title: "Done task", Status: domain.StatusComplete, Created: "2026-09-21"}, "done body")
	writeWO14File(t, filepath.Join(closedTopic, "wos", "01-done.md"), domain.Frontmatter{ID: "00002", Title: "Closed task", Status: domain.StatusComplete, Created: "2026-09-21"}, "closed body")
	if err := os.WriteFile(filepath.Join(openTopic, "brief.md"), []byte("plain Markdown asset"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(closedTopic, "diagram.pdf"), []byte("pdf asset"), 0o644); err != nil {
		t.Fatal(err)
	}

	change, err := RebuildMetadata(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if change.Topics != 2 || change.Updated != 0 || change.Errors != 0 || change.Records != 5 {
		t.Fatalf("rebuild change = %#v", change)
	}

	for _, topic := range []string{openTopic, closedTopic} {
		metadata, err := markdown.ParseTopicMetadataFile(filepath.Join(topic, markdown.MetaFilename))
		if err != nil {
			t.Fatal(err)
		}
		if metadata.Description == "" {
			t.Fatalf("metadata description missing for %s", topic)
		}
		content, err := os.ReadFile(filepath.Join(topic, markdown.MetaFilename))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(content), "Files:") || strings.Contains(string(content), "Assets:") {
			t.Fatalf("generated sections written to canonical metadata %s", topic)
		}
	}

	database, err := sql.Open("sqlite", workspace.Index)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var topics, files, managed, assets int
	if err := database.QueryRow("SELECT COUNT(*) FROM topics").Scan(&topics); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM files").Scan(&files); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM files WHERE type = 'historic_file'").Scan(&managed); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM files WHERE type = 'asset' AND status IS NULL").Scan(&assets); err != nil {
		t.Fatal(err)
	}
	if topics != 2 || files != 5 || managed != 3 || assets != 2 {
		t.Fatalf("read model counts topics=%d files=%d managed=%d assets=%d", topics, files, managed, assets)
	}
	var openStatus, closedStatus string
	if err := database.QueryRow("SELECT computed_status FROM topics WHERE id = '00001'").Scan(&openStatus); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRow("SELECT computed_status FROM topics WHERE id = '00002'").Scan(&closedStatus); err != nil {
		t.Fatal(err)
	}
	if openStatus != "progress" || closedStatus != "complete" {
		t.Fatalf("computed statuses open=%q closed=%q", openStatus, closedStatus)
	}
	var description string
	if err := database.QueryRow("SELECT description FROM files WHERE path = 'wos/01-active.md'").Scan(&description); err != nil {
		t.Fatal(err)
	}
	if description != "active description" {
		t.Fatalf("file description = %q", description)
	}
	var ftsMatches int
	if err := database.QueryRow("SELECT COUNT(*) FROM historic_fts WHERE historic_fts MATCH 'closed description'").Scan(&ftsMatches); err != nil {
		t.Fatal(err)
	}
	if ftsMatches != 1 {
		t.Fatalf("FTS matches = %d, want 1", ftsMatches)
	}

	before, err := os.ReadFile(filepath.Join(openTopic, markdown.MetaFilename))
	if err != nil {
		t.Fatal(err)
	}
	second, err := RebuildMetadata(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if second.Topics != change.Topics || second.Records != change.Records || second.Updated != 0 || second.Errors != 0 {
		t.Fatalf("second rebuild change = %#v", second)
	}
	after, err := os.ReadFile(filepath.Join(openTopic, markdown.MetaFilename))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("second rebuild changed canonical metadata")
	}
}

func writeWO14Metadata(t *testing.T, topic string, metadata markdown.TopicMetadata) {
	t.Helper()
	if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename), metadata); err != nil {
		t.Fatal(err)
	}
}

func writeWO14File(t *testing.T, path string, metadata domain.Frontmatter, body string) {
	t.Helper()
	document, err := markdown.NewDocument(metadata, body)
	if err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(path, document); err != nil {
		t.Fatal(err)
	}
}
