package search

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/indexer"
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
	if _, err := indexer.Rebuild(workspace); err != nil {
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
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	results, err := Find(workspace, Options{Keyword: "target", ArchivedOnly: true, Status: domain.StatusArchived})
	if err != nil || len(results) != 1 || results[0].ID != "00002" {
		t.Fatalf("filtered results = %#v, %v", results, err)
	}
	results, err = Find(workspace, Options{Keyword: "not-present"})
	if err != nil || len(results) != 0 {
		t.Fatalf("empty results = %#v, %v", results, err)
	}
}

func TestFindSupportsPhase4FiltersAndUnicode(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00014-phase-four")
	if err := os.MkdirAll(filepath.Join(topic, "wos"), 0o755); err != nil {
		t.Fatal(err)
	}
	doc, _ := markdown.NewDocument(domain.Frontmatter{ID: "00014", Title: "Phase 4 Search", Status: domain.StatusProgress, Created: "2026-09-19"}, "Unicode café punctuation: FTS ranking and filter.")
	if err := markdown.WriteFile(filepath.Join(topic, "wos", "01-fts-search.md"), doc); err != nil {
		t.Fatal(err)
	}
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	results, err := Find(workspace, Options{Keyword: "café", ActiveOnly: true, Type: "task", ID: "00014", Folder: ".historic/00014-phase-four"})
	if err != nil || len(results) != 1 || results[0].ID != "00014" || !results[0].Active {
		t.Fatalf("phase 4 filtered results = %#v, %v", results, err)
	}
	if results[0].Snippet == "" || strings.Contains(results[0].Path, string(filepath.Separator)+"home"+string(filepath.Separator)) {
		t.Fatalf("unexpected snippet/path = %#v", results[0])
	}
	if _, err := Find(workspace, Options{Keyword: "café", ActiveOnly: true, ArchivedOnly: true}); err == nil {
		t.Fatal("conflicting active/archive filters succeeded")
	}
	if _, err := Find(workspace, Options{Keyword: ""}); err == nil {
		t.Fatal("empty query succeeded")
	}
}

func TestFindSuggestsRebuildWhenIndexMissing(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Find(workspace, Options{Keyword: "missing"}); err == nil || !strings.Contains(err.Error(), "historic rebuild") {
		t.Fatalf("missing index error = %v", err)
	}
}

func TestHighlightHumanOnlyAddsTerminalEmphasis(t *testing.T) {
	const text = "Search keyword appears twice: search."
	highlighted := HighlightHuman(text, "search")
	if highlighted == text || !strings.Contains(highlighted, "\x1b[1;33mSearch\x1b[0m") || !strings.Contains(highlighted, "\x1b[1;33msearch\x1b[0m") {
		t.Fatalf("highlighted = %q", highlighted)
	}
	if plain := HighlightHuman(text, ""); plain != text {
		t.Fatalf("empty keyword changed text: %q", plain)
	}
}
