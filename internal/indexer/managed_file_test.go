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

func TestRebuildClassifiesValidWorkOrderWithDifferentIDAsManagedFile(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00005-topic")
	if err := os.MkdirAll(filepath.Join(topic, "wos"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename), markdown.TopicMetadata{
		ID: "00005", Title: "Topic", Status: domain.StatusProgress, Created: "2026-09-22",
	}); err != nil {
		t.Fatal(err)
	}
	workOrder, err := markdown.NewDocument(domain.Frontmatter{
		ID: "00001", Title: "WO", Status: domain.StatusComplete, Created: "2026-09-22",
	}, "work order")
	if err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(filepath.Join(topic, "wos", "01-work-order.md"), workOrder); err != nil {
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
	var fileType, status string
	if err := database.QueryRow("SELECT type, status FROM files WHERE topic_id = '00005' AND path = 'wos/01-work-order.md'").Scan(&fileType, &status); err != nil {
		t.Fatal(err)
	}
	if fileType != "historic_file" || status != domain.StatusComplete.String() {
		t.Fatalf("WO read model = type:%q status:%q", fileType, status)
	}
	var computed string
	if err := database.QueryRow("SELECT computed_status FROM topics WHERE id = '00005'").Scan(&computed); err != nil {
		t.Fatal(err)
	}
	if computed != domain.StatusComplete.String() {
		t.Fatalf("computed status = %q, want complete", computed)
	}
}
