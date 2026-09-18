package repository

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"historic/internal/domain"
)

func TestListTopicsDefaultsToActiveAndSortsByID(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.CreateTopic("Second", "00002"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateTopic("First", "00001"); err != nil {
		t.Fatal(err)
	}
	archived := filepath.Join(store.Workspace.Database, "00003-archived")
	if err := os.MkdirAll(archived, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(archived, "_meta.md"), []byte("---\nid: 00003\ntitle: Archived\nstatus: archived\ncreated: 2026-09-18\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	views, err := store.ListTopics(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 2 || views[0].ID != "00001" || views[1].ID != "00002" {
		t.Fatalf("views = %#v", views)
	}
	all, err := store.ListTopics(true)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 || all[2].Active {
		t.Fatalf("archived views = %#v", all)
	}
}

func TestShowTopicIncludesFilesAndRejectsMissing(t *testing.T) {
	store := newTestStore(t)
	topic, err := store.CreateTopic("Topic", "00014")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddEntry(topic.ID, "prd", false); err != nil {
		t.Fatal(err)
	}
	view, err := store.ShowTopic(topic.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Files) != 1 || view.Files[0].Path != ".histories/00014-topic/prd.md" {
		t.Fatalf("files = %#v", view.Files)
	}
	if _, err := store.ShowTopic(domain.ID("00099"), false); !errors.Is(err, domain.ErrTopicMissing) {
		t.Fatalf("missing error = %v", err)
	}
}
