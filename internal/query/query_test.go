package query

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/indexer"
	"historic/internal/markdown"
)

func writeQueryDocument(t *testing.T, path string, metadata domain.Frontmatter, body string) {
	t.Helper()
	document, err := markdown.NewDocument(metadata, body)
	if err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(path, document); err != nil {
		t.Fatal(err)
	}
}

func writeQueryMetadata(t *testing.T, path string, metadata domain.Frontmatter) {
	t.Helper()
	if err := markdown.WriteTopicMetadata(path, markdown.TopicMetadataFromFrontmatter(metadata, "")); err != nil {
		t.Fatal(err)
	}
}

func TestQueryTopicsAggregatesManagedFilesExcludesAssetsAndKeepsEmptyTopics(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	mixed := filepath.Join(workspace.Histories, "00001-mixed")
	empty := filepath.Join(workspace.Histories, "00002-empty")
	complete := filepath.Join(workspace.Database, "00003-complete")
	if err := os.MkdirAll(filepath.Join(mixed, "wos"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(empty, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(complete, 0o755); err != nil {
		t.Fatal(err)
	}

	writeQueryMetadata(t, filepath.Join(mixed, markdown.MetaFilename), domain.Frontmatter{ID: "00001", Title: "Mixed", Status: domain.StatusCreate, Tags: []string{"schema"}, Created: "2026-09-01"})
	writeQueryDocument(t, filepath.Join(mixed, "wos", "active.md"), domain.Frontmatter{ID: "00001", Title: "Active", Status: domain.StatusProgress, Tags: []string{"urgent"}, Created: "2026-09-02", Updated: "2026-09-10"}, "active")
	writeQueryDocument(t, filepath.Join(mixed, "wos", "complete.md"), domain.Frontmatter{ID: "00001", Title: "Complete", Status: domain.StatusComplete, Created: "2026-09-03"}, "complete")
	writeQueryDocument(t, filepath.Join(mixed, "wos", "cancelled.md"), domain.Frontmatter{ID: "00001", Title: "Cancelled", Status: domain.StatusCancelled, Created: "2026-09-04"}, "cancelled")
	if err := os.WriteFile(filepath.Join(mixed, "diagram.png"), []byte("asset"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeQueryMetadata(t, filepath.Join(empty, markdown.MetaFilename), domain.Frontmatter{ID: "00002", Title: "Empty", Status: domain.StatusCreate, Created: "2026-09-01"})
	writeQueryMetadata(t, filepath.Join(complete, markdown.MetaFilename), domain.Frontmatter{ID: "00003", Title: "Complete", Status: domain.StatusCreate, Created: "2026-09-01"})
	writeQueryDocument(t, filepath.Join(complete, "done.md"), domain.Frontmatter{ID: "00003", Title: "Done", Status: domain.StatusComplete, Created: "2026-09-05"}, "done")

	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	results, err := QueryTopics(workspace, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got := []string{results[0].ID, results[1].ID, results[2].ID}; !reflect.DeepEqual(got, []string{"00001", "00002", "00003"}) {
		t.Fatalf("topic order = %v", got)
	}
	mixedResult := results[0]
	if mixedResult.TotalFiles != 3 || mixedResult.ActiveFiles != 1 || mixedResult.ResolvedFiles != 2 || mixedResult.CompleteFiles != 1 || mixedResult.CancelledFiles != 1 || mixedResult.ComputedStatus != "active" {
		t.Fatalf("mixed aggregate = %+v", mixedResult)
	}
	if results[1].TotalFiles != 0 || results[1].ComputedStatus != "" {
		t.Fatalf("empty aggregate = %+v", results[1])
	}
	if results[2].ComputedStatus != "complete" || results[2].Storage != "closed" {
		t.Fatalf("complete aggregate = %+v", results[2])
	}
}

func TestQueryFiltersCombineAndFileResultsCarryTopicContext(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00001-filterable")
	if err := os.MkdirAll(filepath.Join(topic, "wos"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeQueryMetadata(t, filepath.Join(topic, markdown.MetaFilename), domain.Frontmatter{ID: "00001", Title: "Filterable", Status: domain.StatusCreate, Tags: []string{"schema"}, Created: "2026-09-01"})
	writeQueryDocument(t, filepath.Join(topic, "wos", "target.md"), domain.Frontmatter{ID: "00001", Title: "Target", Status: domain.StatusProgress, Tags: []string{"urgent"}, Created: "2026-09-02", Updated: "2026-09-10"}, "target")
	writeQueryDocument(t, filepath.Join(topic, "other.md"), domain.Frontmatter{ID: "00001", Title: "Other", Status: domain.StatusComplete, Created: "2026-09-03"}, "other")
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	options := Options{Status: domain.StatusProgress, Storage: domain.StorageOpen, Type: fileTypeHistoric, TopicID: "00001", Folder: "wos", Tags: []string{"urgent"}, UpdatedAfter: "2026-09-01", UpdatedBefore: "2026-09-11"}
	files, err := QueryFiles(workspace, options)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Path != "wos/target.md" {
		t.Fatalf("filtered files = %+v", files)
	}
	if files[0].Topic.ID != "00001" || files[0].Topic.TotalFiles != 2 || files[0].Topic.ActiveFiles != 1 || files[0].Topic.ComputedStatus != "active" {
		t.Fatalf("file topic context = %+v", files[0].Topic)
	}
	if got, err := QueryTopics(workspace, Options{MinFiles: 2, MaxFiles: 2}); err != nil || len(got) != 1 || got[0].ID != "00001" {
		t.Fatalf("HAVING aggregate filter = %+v, %v", got, err)
	}
}

func TestQueryRejectsInvalidFilters(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, options := range []Options{{Type: "unknown"}, {Storage: "unknown"}, {Status: "unknown"}, {Folder: "../escape"}, {MinFiles: 2, MaxFiles: 1}} {
		if _, err := QueryTopics(workspace, options); err == nil {
			t.Fatalf("invalid options accepted: %+v", options)
		}
	}
}

func TestAggregateQuerySmokeHasStableJSONAndExcludesAssets(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00001-smoke")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	writeQueryMetadata(t, filepath.Join(topic, markdown.MetaFilename), domain.Frontmatter{ID: "00001", Title: "Smoke", Status: domain.StatusCreate, Created: "2026-09-01"})
	writeQueryDocument(t, filepath.Join(topic, "done.md"), domain.Frontmatter{
		ID: "00001", Title: "Done", Status: domain.StatusComplete, Created: "2026-09-02",
	}, "done")
	if err := os.WriteFile(filepath.Join(topic, "diagram.png"), []byte("asset"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}

	results, err := QueryTopics(workspace, Options{})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(results)
	if err != nil {
		t.Fatal(err)
	}
	const expected = `[{"id":"00001","num_padded":"00001","title":"Smoke","slug":"smoke","path":".historic/00001-smoke","storage":"open","created_at":"2026-09-01","tags":[],"related":[],"total_files":1,"active_files":0,"resolved_files":1,"complete_files":1,"cancelled_files":0,"computed_status":"complete","last_file_updated_at":"2026-09-02"}]`
	if string(encoded) != expected {
		t.Fatalf("aggregate JSON = %s, want %s", encoded, expected)
	}
	again, err := json.Marshal(results)
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != string(encoded) {
		t.Fatalf("aggregate JSON changed between encodes: %s != %s", encoded, again)
	}
}
