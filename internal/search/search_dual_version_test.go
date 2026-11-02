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

// TestFindDedupesStorageVersionsForTopicsAndFiles verifies that one TopicID
// mirrored into closed storage produces one result per logical file, preferring
// the open storage region regardless of mtime/updated_at.
func TestFindDedupesStorageVersionsForTopicsAndFiles(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	openTopic := filepath.Join(workspace.Histories, "00001-dual")
	closedTopic := filepath.Join(workspace.Database, "00001-dual")
	for _, topic := range []string{openTopic, closedTopic} {
		if err := os.MkdirAll(topic, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename), markdown.TopicMetadata{ID: "00001", Title: "Dual Topic", Created: "2026-10-01"}); err != nil {
			t.Fatal(err)
		}
		document, err := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Dual File", Status: domain.StatusProgress, Created: "2026-10-01"}, "dualtoken body")
		if err != nil {
			t.Fatal(err)
		}
		if err := markdown.WriteFile(filepath.Join(topic, "note.md"), document); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}

	filePages, err := FindPage(workspace, Options{Keyword: "dualtoken"})
	if err != nil {
		t.Fatal(err)
	}
	if len(filePages.Items) != 1 {
		t.Fatalf("deduped file results = %#v, want exactly one", filePages.Items)
	}
	if item := filePages.Items[0]; item.Storage != domain.StorageOpen.String() || item.Path != ".historic/00001-dual/note.md" {
		t.Fatalf("deduped file result = %+v, want open path", item)
	}

	topicPages, err := RecentTopicsPage(workspace, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(topicPages.Items) != 1 {
		t.Fatalf("recent topics = %d, want one deduped version", len(topicPages.Items))
	}
	if topicPages.Items[0].Storage != domain.StorageOpen.String() {
		t.Fatalf("recent topic storage = %q, want open", topicPages.Items[0].Storage)
	}

	openPages, err := FindPage(workspace, Options{Keyword: "dualtoken", OpenOnly: true})
	if err != nil || len(openPages.Items) != 1 || openPages.Items[0].Storage != domain.StorageOpen.String() {
		t.Fatalf("open-only scope = %#v, %v", openPages.Items, err)
	}
	closedPages, err := FindPage(workspace, Options{Keyword: "dualtoken", ClosedOnly: true})
	if err != nil || len(closedPages.Items) != 1 || closedPages.Items[0].Storage != domain.StorageClosed.String() {
		t.Fatalf("closed-only scope = %#v, %v", closedPages.Items, err)
	}
}
