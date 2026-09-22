package search

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/indexer"
	"historic/internal/markdown"
)

func TestNormalizePagination(t *testing.T) {
	defaultPage, err := NormalizePagination(0, 0)
	if err != nil || defaultPage.Page != 1 || defaultPage.PageSize != 20 {
		t.Fatalf("defaults = %+v, %v", defaultPage, err)
	}
	for _, test := range []struct {
		page     int
		pageSize int
	}{
		{page: -1, pageSize: 20},
		{page: 1, pageSize: -1},
		{page: 1, pageSize: 101},
		{page: int(^uint(0) >> 1), pageSize: 100},
	} {
		if _, err := NormalizePagination(test.page, test.pageSize); err == nil {
			t.Fatalf("invalid pagination accepted: %+v", test)
		}
	}
}

func TestFindPageAndRecentTopicsPageUsePageSizePlusOne(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for index := 1; index <= 3; index++ {
		id := domain.ID(fmt.Sprintf("0000%d", index))
		path := filepath.Join(workspace.Histories, id.String()+"-topic")
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
		created := fmt.Sprintf("2026-09-%02d", index)
		if err := markdown.WriteTopicMetadata(filepath.Join(path, markdown.MetaFilename), markdown.TopicMetadata{ID: id, Title: "Page Topic", Created: created}); err != nil {
			t.Fatal(err)
		}
		document, err := markdown.NewDocument(domain.Frontmatter{ID: id, Title: "Page Match", Status: domain.StatusProgress, Created: created}, "page-token")
		if err != nil {
			t.Fatal(err)
		}
		if err := markdown.WriteFile(filepath.Join(path, "note.md"), document); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	first, err := FindPage(workspace, Options{Keyword: "page-token", Page: 1, PageSize: 2})
	if err != nil || len(first.Items) != 2 || !first.Pagination.HasMore || first.Pagination.NextPage == nil || *first.Pagination.NextPage != 2 {
		t.Fatalf("first page = %+v, %v", first, err)
	}
	last, err := FindPage(workspace, Options{Keyword: "page-token", Page: 2, PageSize: 2})
	if err != nil || len(last.Items) != 1 || last.Pagination.HasMore || last.Pagination.NextPage != nil {
		t.Fatalf("last page = %+v, %v", last, err)
	}
	recent, err := RecentTopicsPage(workspace, Options{Page: 2, PageSize: 2})
	if err != nil || len(recent.Items) != 1 || recent.Pagination.HasMore {
		t.Fatalf("recent page = %+v, %v", recent, err)
	}
}
