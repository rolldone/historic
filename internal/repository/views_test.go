package repository

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"historic/internal/domain"
	"historic/internal/markdown"
)

func TestListTopicsDeduplicatesOpenAndClosedSlugCopies(t *testing.T) {
	store := newTestStore(t)
	for _, folder := range []string{"00020-old-slug", "00020-current-slug"} {
		root := store.Workspace.Database
		if folder == "00020-current-slug" {
			root = store.Workspace.Histories
		}
		topic := filepath.Join(root, folder)
		if err := os.MkdirAll(topic, 0o755); err != nil {
			t.Fatal(err)
		}
		metadata := markdown.TopicMetadata{ID: "00020", Title: folder, Status: domain.StatusProgress, Created: "2026-09-21"}
		if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename), metadata); err != nil {
			t.Fatal(err)
		}
	}
	views, err := store.ListTopics(true)
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 1 || views[0].ID != "00020" || views[0].Storage != "open" || views[0].Path != ".historic/00020-current-slug" {
		t.Fatalf("deduplicated views = %#v", views)
	}
}

func TestListTopicsRejectsSymlinkedTopicRoot(t *testing.T) {
	store := newTestStore(t)
	target := t.TempDir()
	if err := os.Symlink(target, filepath.Join(store.Workspace.Histories, "00021-linked-topic")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := store.ListTopics(true); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("symlink topic error = %v, want conflict", err)
	}
}

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
	if err := os.WriteFile(filepath.Join(archived, markdown.MetaFilename), []byte("id: 00003\ntitle: Archived\nstatus: archived\ncreated: 2026-09-18\n"), 0o644); err != nil {
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
	metaPath := filepath.Join(topic.Path, markdown.MetaFilename)
	meta, err := markdown.ParseTopicMetadataFile(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	meta.Description = "Topic summary"
	if err := markdown.WriteTopicMetadata(metaPath, meta); err != nil {
		t.Fatal(err)
	}
	entryPath := filepath.Join(topic.Path, "prd.md")
	entry, err := markdown.ParseFile(entryPath)
	if err != nil {
		t.Fatal(err)
	}
	entry.Frontmatter.Description = "File summary"
	if err := markdown.WriteFile(entryPath, entry); err != nil {
		t.Fatal(err)
	}
	view, err := store.ShowTopic(topic.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if view.Description != "Topic summary" || len(view.Files) != 1 || view.Files[0].Path != ".historic/00014-topic/prd.md" || view.Files[0].Description != "File summary" {
		t.Fatalf("view = %#v, want manual descriptions", view)
	}
	if _, err := store.ShowTopic(domain.ID("00099"), false); !errors.Is(err, domain.ErrTopicMissing) {
		t.Fatalf("missing error = %v", err)
	}
}
