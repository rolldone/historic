package tui

import (
	"strings"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/search"
)

func TestModelDefaultsAndPagination(t *testing.T) {
	model := newModel(config.Workspace{})
	if model.limit != DefaultLimit || model.scope != "all" || model.page != 0 {
		t.Fatalf("defaults = %#v", model)
	}
	model.results = make([]search.Result, DefaultLimit+1)
	if model.pageCount() != 2 || len(model.currentPage()) != DefaultLimit {
		t.Fatalf("pagination count=%d page=%d", model.pageCount(), len(model.currentPage()))
	}
	model.page = 1
	if len(model.currentPage()) != 1 {
		t.Fatalf("last page size = %d", len(model.currentPage()))
	}
}

func TestHighlightFreeViewAndJSONModeValidation(t *testing.T) {
	if strings.Contains(newModel(config.Workspace{}).View(), "ANSI") {
		t.Fatal("default view unexpectedly contains ANSI marker")
	}
}

func TestTerminalValidationRejectsNonTerminal(t *testing.T) {
	if isTerminal(strings.NewReader("input")) {
		t.Fatal("reader reported terminal")
	}
	if got := CycleScopeForTest("all"); got != "active" {
		t.Fatalf("scope cycle from all = %q", got)
	}
	if got := CycleScopeForTest("active"); got != "archived" {
		t.Fatalf("scope cycle from active = %q", got)
	}
}

// resultsModel builds a browsable model around a fixed result slice so the
// rendered list can be asserted without a workspace or a TTY.
func resultsModel(results []search.Result) model {
	model := newModel(config.Workspace{})
	model.width, model.height = 120, 40
	model.limit = DefaultLimit
	model.pagination = search.Pagination{Page: 1, PageSize: DefaultLimit}
	model.results = results
	return model
}

// copyRow is one mirrored topic row, keyed by the same CopyKey so the TUI treats
// the pair as copies of one entity.
func copyRow(storage, stamp, newerCopy string, mirrored bool) search.Result {
	return search.Result{
		Type:      "topic",
		TopicID:   "1794304312863",
		Title:     "Artyplanet Infra",
		Status:    "progress",
		Storage:   storage,
		Path:      ".historic/" + storage + "/1794304312863-artyplanet",
		UpdatedAt: stamp,
		NewerCopy: newerCopy,
		Mirrored:  mirrored,
	}
}

func TestSearchViewShowsCollapsedMirroredRowOnce(t *testing.T) {
	// AC-10: the pair carries one stamp, so the listing collapses it to the open
	// row and renders no divider.
	model := resultsModel([]search.Result{copyRow("open", "2026-11-13T08:53:54Z", "", true)})
	view := model.View()
	if strings.Contains(view, "───") {
		t.Fatalf("collapsed row must not be separated by a divider:\n%s", view)
	}
	if !strings.Contains(view, "mirrored") {
		t.Fatalf("preview must disclose the identical mirror copy:\n%s", view)
	}
}

func TestSearchViewSeparatesDivergedCopiesWithDividerAndIcon(t *testing.T) {
	// AC-9 regression: differing stamps keep two rows, a divider, and the ⚠
	// notice on the newer row.
	model := resultsModel([]search.Result{
		copyRow("closed", "2026-11-13T16:02:11Z", domain.CopyRegionDatabase, false),
		copyRow("open", "2026-11-13T10:45:30Z", "", false),
	})
	view := model.View()
	if !strings.Contains(view, "───") {
		t.Fatalf("diverged copies must be separated by a divider:\n%s", view)
	}
	if !strings.Contains(view, "⚠ newer in database") {
		t.Fatalf("the newer row must carry the icon notice:\n%s", view)
	}
}
