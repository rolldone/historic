package search

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/indexer"
	"historic/internal/markdown"
)

// writeDualTopic creates the same topic in open and closed storage, returning
// both directories.
func writeDualTopic(t *testing.T, workspace config.Workspace, id domain.ID, slug, title, created string) (string, string) {
	t.Helper()
	openTopic := filepath.Join(workspace.Histories, id.String()+"-"+slug)
	closedTopic := filepath.Join(workspace.Database, id.String()+"-"+slug)
	for _, topic := range []string{openTopic, closedTopic} {
		if err := os.MkdirAll(topic, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename), markdown.TopicMetadata{ID: id, Title: title, Created: created}); err != nil {
			t.Fatal(err)
		}
		document, err := markdown.NewDocument(domain.Frontmatter{ID: id, Title: title, Status: domain.StatusProgress, Created: created}, "dedupe-token body")
		if err != nil {
			t.Fatal(err)
		}
		if err := markdown.WriteFile(filepath.Join(topic, "note.md"), document); err != nil {
			t.Fatal(err)
		}
	}
	return openTopic, closedTopic
}

func TestFindDedupesByTopicAndRelativePathPreferOpen(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, closedTopic := writeDualTopic(t, workspace, "00001", "dual", "Dual Topic", "2026-10-01")
	// Make the mirror NEWER than the open copy. Dedupe must still prefer open
	// because preference is region-based, never mtime-based.
	future := time.Now().Add(24 * time.Hour)
	if err := os.Chtimes(filepath.Join(closedTopic, "note.md"), future, future); err != nil {
		t.Fatal(err)
	}
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}

	page, err := FindPage(workspace, Options{Keyword: "dedupe-token"})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("results = %#v, want one deduped row", page.Items)
	}
	item := page.Items[0]
	if item.Storage != domain.StorageOpen.String() || item.Path != ".historic/00001-dual/note.md" {
		t.Fatalf("deduped result = %+v, want open path", item)
	}

	// Explicit closed scope keeps the closed copy untouched by dedupe.
	closedPage, err := FindPage(workspace, Options{Keyword: "dedupe-token", ClosedOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(closedPage.Items) != 1 || closedPage.Items[0].Storage != domain.StorageClosed.String() {
		t.Fatalf("closed-only results = %#v", closedPage.Items)
	}
}

func TestFindKeepsClosedOnlyFiles(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	closedTopic := filepath.Join(workspace.Database, "00002-closed-only")
	if err := os.MkdirAll(closedTopic, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteTopicMetadata(filepath.Join(closedTopic, markdown.MetaFilename), markdown.TopicMetadata{ID: "00002", Title: "Closed Only", Created: "2026-10-02"}); err != nil {
		t.Fatal(err)
	}
	document, err := markdown.NewDocument(domain.Frontmatter{ID: "00002", Title: "Archived Note", Status: domain.StatusComplete, Created: "2026-10-02"}, "closed-token body")
	if err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(filepath.Join(closedTopic, "note.md"), document); err != nil {
		t.Fatal(err)
	}
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}

	page, err := FindPage(workspace, Options{Keyword: "closed-token"})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("results = %#v, want the closed-only file", page.Items)
	}
	if item := page.Items[0]; item.Storage != domain.StorageClosed.String() || item.Path != ".historic/.database/00002-closed-only/note.md" {
		t.Fatalf("closed-only result = %+v", item)
	}
}

func TestFindDedupesBeforePagination(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for index := 1; index <= 3; index++ {
		id := domain.ID(fmt.Sprintf("0000%d", index))
		writeDualTopic(t, workspace, id, "topic", fmt.Sprintf("Page Topic %d", index), fmt.Sprintf("2026-09-%02d", index))
	}
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}

	first, err := FindPage(workspace, Options{Keyword: "dedupe-token", Page: 1, PageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 2 || !first.Pagination.HasMore || first.Pagination.NextPage == nil || *first.Pagination.NextPage != 2 {
		t.Fatalf("first page = %+v", first)
	}
	last, err := FindPage(workspace, Options{Keyword: "dedupe-token", Page: 2, PageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(last.Items) != 1 || last.Pagination.HasMore || last.Pagination.NextPage != nil {
		t.Fatalf("last page = %+v", last)
	}

	seen := map[string]struct{}{}
	for _, item := range append(first.Items, last.Items...) {
		if _, duplicate := seen[item.TopicID]; duplicate {
			t.Fatalf("topic %s appeared twice across pages", item.TopicID)
		}
		seen[item.TopicID] = struct{}{}
	}
	if len(seen) != 3 {
		t.Fatalf("topics across pages = %d, want 3", len(seen))
	}
}

func TestRecentTopicsPageDedupesMirrorVersions(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeDualTopic(t, workspace, "00001", "dual", "Dual Topic", "2026-10-01")
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	page, err := RecentTopicsPage(workspace, Options{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].Storage != domain.StorageOpen.String() {
		t.Fatalf("recent topics = %+v, want one open row", page.Items)
	}
	if page.Pagination.HasMore {
		t.Fatalf("pagination = %+v, want no more results", page.Pagination)
	}
}
