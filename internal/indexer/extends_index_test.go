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

const (
	extendsParentID = "1791851149892"
	extendsChildID  = "1792889892200"
)

func writeExtendsTopic(t *testing.T, path string, metadata markdown.TopicMetadata) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteTopicMetadata(filepath.Join(path, markdown.MetaFilename), metadata); err != nil {
		t.Fatal(err)
	}
}

func extendsValue(t *testing.T, database *sql.DB, id, storage string) sql.NullString {
	t.Helper()
	var extends sql.NullString
	if err := database.QueryRow("SELECT extends_id FROM topics WHERE id = ? AND storage = ?", id, storage).Scan(&extends); err != nil {
		t.Fatalf("query extends_id for %s/%s: %v", id, storage, err)
	}
	return extends
}

func TestRebuildStoresExtendsID(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeExtendsTopic(t, filepath.Join(workspace.Histories, extendsParentID+"-parent"), markdown.TopicMetadata{
		ID: domain.ID(extendsParentID), Title: "Parent", Created: "2026-10-25",
	})
	writeExtendsTopic(t, filepath.Join(workspace.Database, extendsParentID+"-parent"), markdown.TopicMetadata{
		ID: domain.ID(extendsParentID), Title: "Parent", Created: "2026-10-25",
	})
	writeExtendsTopic(t, filepath.Join(workspace.Histories, extendsChildID+"-child"), markdown.TopicMetadata{
		ID: domain.ID(extendsChildID), Title: "Child", Created: "2026-10-25",
		Extends: domain.TopicID(extendsParentID),
	})

	parentMeta := filepath.Join(workspace.Database, extendsParentID+"-parent", markdown.MetaFilename)
	before, err := os.ReadFile(parentMeta)
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

	for _, storage := range []string{"open", "closed"} {
		if extends := extendsValue(t, database, extendsParentID, storage); extends.Valid {
			t.Fatalf("parent %s extends_id = %q, want NULL", storage, extends.String)
		}
	}
	childExtends := extendsValue(t, database, extendsChildID, "open")
	if !childExtends.Valid || childExtends.String != extendsParentID {
		t.Fatalf("child extends_id = %#v, want %s", childExtends, extendsParentID)
	}

	after, err := os.ReadFile(parentMeta)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("rebuild modified parent metadata")
	}
	if _, err := Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	repeated := extendsValue(t, database, extendsChildID, "open")
	if !repeated.Valid || repeated.String != extendsParentID {
		t.Fatalf("repeated rebuild extends_id = %#v", repeated)
	}
}

func TestRebuildStoresForwardedToOnFileReadModel(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, extendsChildID+"-child")
	writeExtendsTopic(t, topic, markdown.TopicMetadata{
		ID: domain.ID(extendsChildID), Title: "Child", Created: "2026-10-25",
		Extends: domain.TopicID(extendsParentID),
	})
	document, err := markdown.NewDocument(domain.Frontmatter{
		ID: domain.ID(extendsChildID), Title: "Task", Status: domain.StatusForwarded,
		Created: "2026-10-25", ForwardedTo: domain.TopicID(extendsParentID),
	}, "body")
	if err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(filepath.Join(topic, "task.md"), document); err != nil {
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
	var forwarded sql.NullString
	var status string
	if err := database.QueryRow("SELECT forwarded_to, status FROM files WHERE topic_id = ? AND path = 'task.md'", extendsChildID).Scan(&forwarded, &status); err != nil {
		t.Fatal(err)
	}
	if !forwarded.Valid || forwarded.String != extendsParentID {
		t.Fatalf("forwarded_to = %#v, want %s", forwarded, extendsParentID)
	}
	if status != domain.StatusForwarded.String() {
		t.Fatalf("status = %q", status)
	}
}

func TestRebuildStoresNullForwardedToForLegacyFiles(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00008-plain")
	writeExtendsTopic(t, topic, markdown.TopicMetadata{ID: "00008", Title: "Plain", Created: "2026-09-01"})
	document, err := markdown.NewDocument(domain.Frontmatter{ID: "00008", Title: "Task", Status: domain.StatusProgress, Created: "2026-09-01"}, "body")
	if err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(filepath.Join(topic, "task.md"), document); err != nil {
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
	var forwarded sql.NullString
	if err := database.QueryRow("SELECT forwarded_to FROM files WHERE topic_id = '00008' AND path = 'task.md'").Scan(&forwarded); err != nil {
		t.Fatal(err)
	}
	if forwarded.Valid {
		t.Fatalf("legacy forwarded_to = %q, want NULL", forwarded.String)
	}
}

func TestRebuildStoresNullExtendsForLegacyMetadata(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00007-legacy")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := "id: \"00007\"\ntitle: Legacy\ncreated: \"2026-09-01\"\nfiles: []\nassets: []\n"
	if err := os.WriteFile(filepath.Join(topic, markdown.MetaFilename), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Rebuild(workspace); err != nil {
		t.Fatalf("rebuild legacy metadata: %v", err)
	}
	database, err := sql.Open("sqlite", workspace.Index)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	extends := extendsValue(t, database, "00007", "open")
	if extends.Valid {
		t.Fatalf("legacy extends_id = %q, want NULL", extends.String)
	}
}
