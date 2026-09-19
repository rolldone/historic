package tui

import (
	"strings"
	"testing"

	"historic/internal/config"
	"historic/internal/search"
)

func TestViewRespectsTerminalViewportAndKeepsSelectedVisible(t *testing.T) {
	model := newModel(config.Workspace{})
	model.width = 42
	model.height = 10
	model.results = make([]search.Result, DefaultLimit)
	for index := range model.results {
		model.results[index] = search.Result{ID: "00001", Title: "very long result title that must be clipped safely"}
	}
	model.selected = 8
	view := model.View()
	if lines := strings.Count(view, "\n") + 1; lines > model.height {
		t.Fatalf("view lines=%d exceeds height=%d\n%s", lines, model.height, view)
	}
	if !strings.Contains(view, "> 00001") {
		t.Fatal("selected item is not visible")
	}
	if !strings.Contains(view, "...") {
		t.Fatal("long result was not truncated")
	}
}

func TestViewUsesCompactLayoutForSmallTerminal(t *testing.T) {
	model := newModel(config.Workspace{})
	model.width = 30
	model.height = 7
	model.results = []search.Result{{ID: "00001", Title: "topic"}}
	view := model.View()
	if lines := strings.Count(view, "\n") + 1; lines > model.height {
		t.Fatalf("compact view lines=%d exceeds height=%d\n%s", lines, model.height, view)
	}
	if !strings.Contains(view, "↑↓ navigate") {
		t.Fatalf("footer missing: %q", view)
	}
}

func TestTruncatePreservesUnicodeBoundaries(t *testing.T) {
	value := truncate("café topic", 5)
	if !strings.HasSuffix(value, "...") || strings.ContainsRune(value, '\ufffd') {
		t.Fatalf("unsafe unicode truncation: %q", value)
	}
}
