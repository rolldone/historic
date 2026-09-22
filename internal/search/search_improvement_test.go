package search

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/indexer"
	"historic/internal/markdown"
)

func TestSearchStatusNotFreshnessSortAndEmptyQuery(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00001-search")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename), markdown.TopicMetadata{ID: "00001", Title: "Search", Created: "2026-09-20"}); err != nil {
		t.Fatal(err)
	}
	complete, _ := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Complete Work", Status: domain.StatusComplete, Created: "2026-09-20"}, "work complete")
	progress, _ := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Active Work", Status: domain.StatusProgress, Created: "2026-09-21", Updated: "2026-09-22"}, "work active")
	if err := markdown.WriteFile(filepath.Join(topic, "complete.md"), complete); err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(filepath.Join(topic, "active.md"), progress); err != nil {
		t.Fatal(err)
	}
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	results, err := Find(workspace, Options{Keyword: "work", StatusNot: []domain.Status{domain.StatusComplete}})
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range results {
		if result.Type == "historic_file" && result.Status == domain.StatusComplete.String() {
			t.Fatalf("excluded complete result: %+v", result)
		}
	}
	if _, err := Find(workspace, Options{Keyword: "work", Status: domain.StatusProgress, StatusNot: []domain.Status{domain.StatusProgress}}); err == nil {
		t.Fatal("status conflict accepted")
	}
	results, err = Find(workspace, Options{Keyword: "work", Sort: "updated"})
	if err != nil || len(results) == 0 {
		t.Fatalf("sorted results = %+v, %v", results, err)
	}
	var foundFresh bool
	for _, result := range results {
		if result.Type == "historic_file" && result.Title == "Active Work" {
			foundFresh = true
			if result.CreatedAt != "2026-09-21" || result.UpdatedAt != "2026-09-22" || result.Mtime == "" || result.Hash == "" || result.Size == 0 {
				t.Fatalf("freshness = %+v", result)
			}
		}
	}
	if !foundFresh {
		t.Fatal("active file freshness result missing")
	}
	recent, err := RecentTopics(workspace, Options{})
	if err != nil || len(recent) != 1 {
		t.Fatalf("recent = %+v, %v", recent, err)
	}
	if recent[0].Topic.ID != "00001" {
		t.Fatalf("recent topic context = %+v", recent[0])
	}
	if _, err := json.Marshal(results); err != nil {
		t.Fatal(err)
	}
}
