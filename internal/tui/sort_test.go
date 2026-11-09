package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/indexer"
	"historic/internal/markdown"
	"historic/internal/search"
)

func TestCycleSortWrapsThroughEveryMode(t *testing.T) {
	if got := CycleSortForTest(""); got != SortUpdated {
		t.Fatalf("cycle from default = %q, want %q", got, SortUpdated)
	}
	if got := CycleSortForTest(SortRelevance); got != SortUpdated {
		t.Fatalf("cycle from relevance = %q, want %q", got, SortUpdated)
	}
	if got := CycleSortForTest(SortUpdated); got != SortCreated {
		t.Fatalf("cycle from updated = %q, want %q", got, SortCreated)
	}
	if got := CycleSortForTest(SortCreated); got != SortTitle {
		t.Fatalf("cycle from created = %q, want %q", got, SortTitle)
	}
	if got := CycleSortForTest(SortTitle); got != SortRelevance {
		t.Fatalf("cycle from title = %q, want %q", got, SortRelevance)
	}
}

func TestSortOptionMapsRelevanceToSearchDefault(t *testing.T) {
	cases := map[string]string{
		"":            "",
		SortRelevance: "",
		SortUpdated:   "updated",
		SortCreated:   "created",
		SortTitle:     "title",
	}
	for mode, want := range cases {
		model := model{sortMode: mode}
		if got := model.sortOption(); got != want {
			t.Fatalf("sortOption(%q) = %q, want %q", mode, got, want)
		}
	}
}

func TestSortNameDefaultsToRelevance(t *testing.T) {
	if got := (&model{}).sortName(); got != SortRelevance {
		t.Fatalf("empty sortName = %q, want %q", got, SortRelevance)
	}
}

// seedTuiSortWorkspace creates one topic with two files: "Alpha" is older while
// "Zeta" is newer, so title order and freshness order disagree.
func seedTuiSortWorkspace(t *testing.T) config.Workspace {
	t.Helper()
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00001-sort")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename),
		markdown.TopicMetadata{ID: "00001", Title: "Sort", Created: "2026-09-01"}); err != nil {
		t.Fatal(err)
	}
	write := func(name, title, created string) {
		t.Helper()
		document, err := markdown.NewDocument(domain.Frontmatter{
			ID: "00001", Title: title, Status: domain.StatusProgress, Created: created,
		}, "shared keyword body")
		if err != nil {
			t.Fatal(err)
		}
		if err := markdown.WriteFile(filepath.Join(topic, name), document); err != nil {
			t.Fatal(err)
		}
	}
	write("alpha.md", "Alpha", "2026-09-01")
	write("zeta.md", "Zeta", "2026-09-10")
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	return workspace
}

func TestSortModeChangesResultOrder(t *testing.T) {
	workspace := seedTuiSortWorkspace(t)

	order := func(mode string) []string {
		t.Helper()
		model := model{sortMode: mode}
		page, err := search.FindPage(workspace, search.Options{Keyword: "keyword", Sort: model.sortOption()})
		if err != nil {
			t.Fatal(err)
		}
		titles := make([]string, 0, len(page.Items))
		for _, item := range page.Items {
			if item.Type == "historic_file" {
				titles = append(titles, item.Title)
			}
		}
		return titles
	}

	titleOrder := order(SortTitle)
	if len(titleOrder) != 2 || titleOrder[0] != "Alpha" || titleOrder[1] != "Zeta" {
		t.Fatalf("title order = %v, want [Alpha Zeta]", titleOrder)
	}
	updatedOrder := order(SortUpdated)
	if len(updatedOrder) != 2 || updatedOrder[0] != "Zeta" {
		t.Fatalf("updated order = %v, want Zeta first", updatedOrder)
	}
	if titleOrder[0] == updatedOrder[0] {
		t.Fatalf("sort modes did not reorder results: title=%v updated=%v", titleOrder, updatedOrder)
	}
}

func TestViewShowsActiveSortMode(t *testing.T) {
	model := newModel(config.Workspace{})
	model.width, model.height = 120, 30
	if !strings.Contains(model.View(), "sort=relevance") {
		t.Fatalf("default view does not show sort indicator")
	}
	model.sortMode = SortTitle
	view := model.View()
	if !strings.Contains(view, "sort=title") {
		t.Fatalf("view does not show the active sort mode: %q", view)
	}
}
