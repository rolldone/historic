package search

import (
	"os"
	"path/filepath"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/indexer"
	"historic/internal/markdown"
)

func TestFindWO09SearchesTopicsAndFilesWithReadModelFilters(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	openTopic := filepath.Join(workspace.Histories, "00001-open-topic")
	closedTopic := filepath.Join(workspace.Database, "00002-closed-topic")
	for _, root := range []string{openTopic, closedTopic} {
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(path string, metadata domain.Frontmatter, body string) {
		t.Helper()
		document, err := markdown.NewDocument(metadata, body)
		if err != nil {
			t.Fatal(err)
		}
		if err := markdown.WriteFile(path, document); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(openTopic, "_meta.md"), domain.Frontmatter{
		ID: "00001", Title: "Needle Topic", Description: "Topic description", Status: domain.StatusProgress,
		Created: "2026-09-01", Tags: []string{"architecture"},
	}, "topic context")
	write(filepath.Join(openTopic, "note.md"), domain.Frontmatter{
		ID: "00001", Title: "Implementation Note", Description: "File description", Status: domain.StatusProgress,
		Created: "2026-09-02", Tags: []string{"needle"},
	}, "content only")
	write(filepath.Join(closedTopic, "_meta.md"), domain.Frontmatter{
		ID: "00002", Title: "Closed Topic", Status: domain.StatusComplete, Created: "2026-09-03",
	}, "needle snapshot")
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}

	results, err := Find(workspace, Options{Keyword: "needle"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Fatalf("default open+closed results = %#v", results)
	}
	seenStorage := map[string]bool{}
	for _, result := range results {
		seenStorage[result.Storage] = true
		if result.TopicID == "" || result.Title == "" || result.Path == "" || result.Snippet == "" {
			t.Fatalf("incomplete result = %+v", result)
		}
	}
	if !seenStorage[domain.StorageOpen.String()] || !seenStorage[domain.StorageClosed.String()] {
		t.Fatalf("default storage coverage = %v", seenStorage)
	}
	filtered, err := Find(workspace, Options{Keyword: "needle", Tags: []string{"needle"}, OpenOnly: true})
	if err != nil || len(filtered) != 1 || filtered[0].Path != ".historic/00001-open-topic/note.md" {
		t.Fatalf("tag/open filter = %#v, %v", filtered, err)
	}
}

func TestFindWO09WeightsMetadataAndReportsMatchedFields(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00001-ranking")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name string, metadata domain.Frontmatter, body string) {
		t.Helper()
		document, err := markdown.NewDocument(metadata, body)
		if err != nil {
			t.Fatal(err)
		}
		if err := markdown.WriteFile(filepath.Join(topic, name), document); err != nil {
			t.Fatal(err)
		}
	}
	write("_meta.md", domain.Frontmatter{ID: "00001", Title: "Metadata Needle", Status: domain.StatusProgress, Created: "2026-09-01"}, "context")
	write("content.md", domain.Frontmatter{ID: "00001", Title: "Content File", Status: domain.StatusProgress, Created: "2026-09-01"}, "content needle")
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	results, err := Find(workspace, Options{Keyword: "needle"})
	if err != nil || len(results) < 2 {
		t.Fatalf("ranking results = %#v, %v", results, err)
	}
	if results[0].Type != "topic" || results[0].Score <= results[1].Score || len(results[0].MatchedIn) == 0 || results[0].MatchedIn[0] != "title" {
		t.Fatalf("metadata ranking = %#v", results[:2])
	}
}
