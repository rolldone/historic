package query

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

func TestQueryTopicsPagePagination(t *testing.T) {
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
		writeQueryMetadata(t, filepath.Join(path, markdown.MetaFilename), domain.Frontmatter{ID: id, Title: fmt.Sprintf("Topic %d", index), Created: fmt.Sprintf("2026-09-%02d", index)})
	}
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}

	page, err := NewService(workspace).QueryTopicsPage(Options{Page: 1, PageSize: 2})
	if err != nil || len(page.Items) != 2 || !page.Pagination.HasMore || page.Pagination.NextPage == nil || *page.Pagination.NextPage != 2 {
		t.Fatalf("first topic page = %+v, %v", page, err)
	}
	last, err := NewService(workspace).QueryTopicsPage(Options{Page: 2, PageSize: 2})
	if err != nil || len(last.Items) != 1 || last.Pagination.HasMore || last.Pagination.NextPage != nil {
		t.Fatalf("last topic page = %+v, %v", last, err)
	}
}
