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

func TestSearchFiltersForwardedStatus(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "1792889892200-child")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename),
		markdown.TopicMetadata{ID: "1792889892200", Title: "Child", Created: "2026-10-25"}); err != nil {
		t.Fatal(err)
	}
	forwarded, err := markdown.NewDocument(domain.Frontmatter{
		ID: "1792889892200", Title: "Moved Task", Status: domain.StatusForwarded,
		Created: "2026-10-25", ForwardedTo: "1791851149892",
	}, "moved work")
	if err != nil {
		t.Fatal(err)
	}
	active, err := markdown.NewDocument(domain.Frontmatter{
		ID: "1792889892200", Title: "Active Task", Status: domain.StatusProgress, Created: "2026-10-25",
	}, "active work")
	if err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(filepath.Join(topic, "moved.md"), forwarded); err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(filepath.Join(topic, "active.md"), active); err != nil {
		t.Fatal(err)
	}
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}

	results, err := Find(workspace, Options{Keyword: "work", Status: domain.StatusForwarded})
	if err != nil {
		t.Fatalf("find forwarded: %v", err)
	}
	found := false
	for _, result := range results {
		if result.Type == "historic_file" {
			if result.Status != domain.StatusForwarded.String() {
				t.Fatalf("unexpected status %q in forwarded filter", result.Status)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("forwarded filter returned no files")
	}

	excluded, err := Find(workspace, Options{Keyword: "work", StatusNot: []domain.Status{domain.StatusForwarded}})
	if err != nil {
		t.Fatalf("find status-not forwarded: %v", err)
	}
	for _, result := range excluded {
		if result.Type == "historic_file" && result.Status == domain.StatusForwarded.String() {
			t.Fatalf("forwarded file leaked into status-not filter: %+v", result)
		}
	}
	if _, err := Find(workspace, Options{Keyword: "work", Status: domain.Status("unknown")}); err == nil {
		t.Fatal("unknown status accepted")
	}
}
