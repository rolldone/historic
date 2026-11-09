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

// seedSortIDWorkspace builds topics 00002 (closed) and 00001 (open), each with
// one matching file, so the legacy ID ascending order can be asserted.
func seedSortIDWorkspace(t *testing.T) config.Workspace {
	t.Helper()
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	write := func(root, id, title string) {
		t.Helper()
		topic := filepath.Join(root, id+"-"+title)
		if err := os.MkdirAll(topic, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename),
			markdown.TopicMetadata{ID: domain.ID(id), Title: title, Created: "2026-09-18"}); err != nil {
			t.Fatal(err)
		}
		status := domain.StatusProgress
		if root == workspace.Database {
			status = domain.StatusComplete
		}
		document, err := markdown.NewDocument(domain.Frontmatter{
			ID: domain.ID(id), Title: "Target " + id, Status: status, Created: "2026-09-18",
		}, "target body keyword")
		if err != nil {
			t.Fatal(err)
		}
		if err := markdown.WriteFile(filepath.Join(topic, "note.md"), document); err != nil {
			t.Fatal(err)
		}
	}
	write(workspace.Histories, "00001", "one")
	write(workspace.Database, "00002", "two")
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	return workspace
}

func TestFindSortIDOrdersLegacyIDAscending(t *testing.T) {
	workspace := seedSortIDWorkspace(t)

	results, err := Find(workspace, Options{Keyword: "keyword", Sort: "id"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("no results")
	}
	for index := 1; index < len(results); index++ {
		previous, current := results[index-1].TopicID, results[index].TopicID
		if previous > current {
			t.Fatalf("results not ordered by topic ID ascending: %s before %s (%+v)", previous, current, results)
		}
	}
	if results[0].TopicID != "00001" || results[len(results)-1].TopicID != "00002" {
		t.Fatalf("sort id edges = %s..%s, want 00001..00002", results[0].TopicID, results[len(results)-1].TopicID)
	}
}

func TestRecentTopicsSortIDOrdersLegacyIDAscending(t *testing.T) {
	workspace := seedSortIDWorkspace(t)

	recent, err := RecentTopics(workspace, Options{Sort: "id"})
	if err != nil {
		t.Fatal(err)
	}
	if len(recent) != 2 {
		t.Fatalf("recent = %+v", recent)
	}
	if recent[0].TopicID != "00001" || recent[1].TopicID != "00002" {
		t.Fatalf("recent sort id = %s, %s; want 00001, 00002", recent[0].TopicID, recent[1].TopicID)
	}
}

func TestFindSortRejectsUnknownValueNamingID(t *testing.T) {
	workspace := seedSortIDWorkspace(t)

	_, err := Find(workspace, Options{Keyword: "keyword", Sort: "bogus"})
	if err == nil {
		t.Fatal("unknown sort value accepted")
	}
	if !strings.Contains(err.Error(), "id") || !strings.Contains(err.Error(), "relevance") {
		t.Fatalf("error = %q, want it to name every accepted value", err)
	}
}
