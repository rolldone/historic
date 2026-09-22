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

func TestRebuildTreatsAnyHistoricMarkdownAsTopicMember(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00005-topic")
	if err := os.MkdirAll(filepath.Join(topic, "wos"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename), markdown.TopicMetadata{
		ID: "00005", Title: "Topic", Created: "2026-09-22",
	}); err != nil {
		t.Fatal(err)
	}
	for path, metadata := range map[string]domain.Frontmatter{
		"wos/spec.md": {ID: "00001", Title: "Spec", Status: domain.StatusDraft, Created: "2026-09-22", Related: []domain.ID{"../spec.md"}},
		"wos/wo.md":   {ID: "00002", Title: "Work Order", Status: domain.StatusComplete, Created: "2026-09-22", Related: []domain.ID{"./spec.md"}},
	} {
		document, docErr := markdown.NewDocument(metadata, "historic body")
		if docErr != nil {
			t.Fatal(docErr)
		}
		if err := markdown.WriteFile(filepath.Join(topic, filepath.FromSlash(path)), document); err != nil {
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
	var members int
	if err := database.QueryRow("SELECT COUNT(*) FROM files WHERE topic_id = '00005' AND type = 'historic_file'").Scan(&members); err != nil {
		t.Fatal(err)
	}
	if members != 2 {
		t.Fatalf("Historic members = %d, want 2", members)
	}
	var draft string
	if err := database.QueryRow("SELECT status FROM files WHERE topic_id = '00005' AND path = 'wos/spec.md'").Scan(&draft); err != nil {
		t.Fatal(err)
	}
	if draft != domain.StatusDraft.String() {
		t.Fatalf("draft member status = %q", draft)
	}
}
