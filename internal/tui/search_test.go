package tui

import (
	"strings"
	"testing"

	"historic/internal/config"
	"historic/internal/search"
)

func TestModelDefaultsAndPagination(t *testing.T) {
	model := newModel(config.Workspace{})
	if model.limit != DefaultLimit || model.scope != "active" || model.page != 0 {
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
	if got := CycleScopeForTest("active"); got != "archived" {
		t.Fatalf("scope cycle = %q", got)
	}
}
