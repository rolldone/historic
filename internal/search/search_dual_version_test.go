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

// TestFindReturnsBothStorageVersionsForTopicsAndFiles verifies that one TopicID
// mirrored into closed storage produces one result per version, for topics and
// for managed files, with the open copy ordered first.
func TestFindReturnsBothStorageVersionsForTopicsAndFiles(t *testing.T) {
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
	fileStorages := map[string]int{}
	for _, item := range filePages.Items {
		fileStorages[item.Storage]++
	}
	if fileStorages[domain.StorageOpen.String()] != 1 || fileStorages[domain.StorageClosed.String()] != 1 {
		t.Fatalf("file versions = %v, want one open and one closed", fileStorages)
	}

	topicPages, err := RecentTopicsPage(workspace, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(topicPages.Items) != 2 {
		t.Fatalf("recent topics = %d, want 2 versions", len(topicPages.Items))
	}
	if topicPages.Items[0].Storage != domain.StorageOpen.String() || topicPages.Items[1].Storage != domain.StorageClosed.String() {
		t.Fatalf("recent topic order = %q, %q", topicPages.Items[0].Storage, topicPages.Items[1].Storage)
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
