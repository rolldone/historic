package search

import (
	"os"
	"path/filepath"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"
)

func TestFindMatchesFilenameTitleAndBody(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(workspace.Histories, "00001-topic", "note.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	doc, err := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Meeting Notes", Status: domain.StatusProgress, Created: "2026-09-18"}, "Body contains Needle.")
	if err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(path, doc); err != nil {
		t.Fatal(err)
	}
	for _, keyword := range []string{"NOTE", "meeting", "needle"} {
		results, err := Find(workspace, Options{Keyword: keyword, ActiveOnly: true})
		if err != nil || len(results) != 1 {
			t.Fatalf("Find(%q) = %#v, %v", keyword, results, err)
		}
	}
}

func TestFindFiltersAndEmptyResult(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	active := filepath.Join(workspace.Histories, "00001-active")
	archived := filepath.Join(workspace.Database, "00002-archived")
	for _, root := range []string{active, archived} {
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(path string, id domain.ID, status domain.Status) {
		t.Helper()
		doc, _ := markdown.NewDocument(domain.Frontmatter{ID: id, Title: "Target", Status: status, Created: "2026-09-18"}, "target body")
		if err := markdown.WriteFile(filepath.Join(path, "note.md"), doc); err != nil {
			t.Fatal(err)
		}
	}
	write(active, "00001", domain.StatusProgress)
	write(archived, "00002", domain.StatusArchived)
	results, err := Find(workspace, Options{Keyword: "target", ArchivedOnly: true, Status: domain.StatusArchived})
	if err != nil || len(results) != 1 || results[0].ID != "00002" {
		t.Fatalf("filtered results = %#v, %v", results, err)
	}
	results, err = Find(workspace, Options{Keyword: "not-present"})
	if err != nil || len(results) != 0 {
		t.Fatalf("empty results = %#v, %v", results, err)
	}
}
