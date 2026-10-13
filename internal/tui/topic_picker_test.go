package tui

import (
	"strings"
	"testing"
	"time"

	"historic/internal/repository"

	tea "github.com/charmbracelet/bubbletea"
)

func pickerViews() []repository.TopicView {
	views := []repository.TopicView{
		{ID: "00002", Title: "Second", Updated: "2026-10-10", Mtime: "2026-10-10T00:00:00Z"},
		{ID: "00001", Title: "First", Updated: "", Mtime: "2026-10-12T00:00:00Z"},
		{ID: "00003", Title: "Third", Updated: "2026-10-12", Mtime: "2026-10-01T00:00:00Z"},
	}
	for index := range views {
		views[index].Body = strings.ToLower(views[index].ID + " " + views[index].Title)
	}
	return views
}

func TestPickerTimestampPrefersUpdatedOverMtime(t *testing.T) {
	old := pickerTimestamp(repository.TopicView{Updated: "", Mtime: "2026-10-10T00:00:00Z"})
	recent := pickerTimestamp(repository.TopicView{Updated: "2026-10-12", Mtime: "2026-10-01T00:00:00Z"})
	if !recent.After(old) {
		t.Fatalf("expected updated timestamp to win: %v vs %v", recent, old)
	}
	if got := pickerTimestamp(repository.TopicView{}); !got.Equal(time.Time{}) {
		t.Fatalf("expected zero time, got %v", got)
	}
}

func TestPickerFilteredQueryMatchesIDTitle(t *testing.T) {
	model := topicPickerModel{views: pickerViews()}
	if len(model.filtered()) != 3 {
		t.Fatalf("expected all views, got %d", len(model.filtered()))
	}
	model.query = "second"
	rows := model.filtered()
	if len(rows) != 1 || rows[0].ID != "00002" {
		t.Fatalf("query filter = %+v", rows)
	}
}

func TestPickerNavigationAndCancel(t *testing.T) {
	model := topicPickerModel{views: pickerViews()}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(topicPickerModel)
	if model.selected != 1 {
		t.Fatalf("selected = %d", model.selected)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = updated.(topicPickerModel)
	if model.selected != 0 {
		t.Fatalf("selected = %d", model.selected)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if !updated.(topicPickerModel).cancelled {
		t.Fatal("expected cancelled")
	}
	empty := topicPickerModel{views: pickerViews()}
	updated, _ = empty.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !updated.(topicPickerModel).cancelled {
		t.Fatal("expected q to cancel empty query")
	}
}
