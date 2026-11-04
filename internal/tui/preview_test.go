package tui

import (
	"strings"
	"testing"

	"historic/internal/config"
	"historic/internal/search"
)

func previewTestModel(width, height int) model {
	model := newModel(config.Workspace{})
	model.width, model.height = width, height
	model.pagination = search.Pagination{Page: 1, PageSize: DefaultLimit}
	model.results = []search.Result{{
		Type:        "historic_file",
		TopicID:     "1793750581032",
		Title:       "WO 02 — TUI Preview Terstruktur",
		Description: "Restrukturisasi panel preview TUI search: description penuh, relasi tampil, dan truncation aman viewport agar hasil aktif selalu terlihat.",
		Status:      "progress",
		Storage:     "open",
		Path:        ".historic/1793750581032-search-display-improvement/wos/02-tui-preview.md",
		UpdatedAt:   "2026-11-04",
		Mtime:       "2026-11-04T00:12:24.861856821Z",
		Tags:        []string{"tui", "display"},
		Extends:     "1793622992705",
		ForwardedTo: "1793629449341",
		MatchCount:  3,
		MatchLine:   21,
		Snippet:     "panel preview terstruktur dengan relasi",
	}}
	model.input.SetValue("preview")
	return model
}

func TestPreviewShowsDescriptionRelationsAndLabels(t *testing.T) {
	model := previewTestModel(120, 40)
	view := model.View()
	for _, want := range []string{
		"reason",
		"Restrukturisasi panel preview",
		"  extends → 1793622992705",
		"  forwarded → 1793629449341",
		"  tag     tui, display",
		"  path    .historic/",
		"  time    updated 2026-11-04 · mtime 2026-11-04T00:12:24Z",
		"[OPEN] · progress",
		"3 lokasi · terbaik di sekitar baris 21",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("preview missing %q:\n%s", want, view)
		}
	}
	if lines := strings.Count(view, "\n") + 1; lines > model.height {
		t.Fatalf("view lines=%d exceeds height=%d\n%s", lines, model.height, view)
	}
}

func TestPreviewScrollClampsAndKeepsFooter(t *testing.T) {
	model := previewTestModel(80, 24)
	tail := "penanda-akhir-preview"
	words := make([]string, 0, 200)
	for index := 0; index < 120; index++ {
		words = append(words, "kalimat")
	}
	words = append(words, tail)
	model.results[0].Description = strings.Join(words, " ")

	view := model.View()
	if !strings.Contains(view, "Page 1 · showing 1 results") {
		t.Fatalf("footer not visible:\n%s", view)
	}
	if strings.Contains(view, tail) {
		t.Fatal("long preview should be clipped before scrolling")
	}

	model.previewOffset = 10000
	view = model.View()
	if !strings.Contains(view, tail) {
		t.Fatalf("preview did not clamp to the end:\n%s", view)
	}
	if lines := strings.Count(view, "\n") + 1; lines > model.height {
		t.Fatalf("view lines=%d exceeds height=%d\n%s", lines, model.height, view)
	}
}
