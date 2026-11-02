package repository

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"historic/internal/domain"
	"historic/internal/markdown"
)

func TestListTopicsListsOpenAndClosedVersionsIndependently(t *testing.T) {
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
		metadata := markdown.TopicMetadata{ID: "00020", Title: folder, Created: "2026-09-21"}
		if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename), metadata); err != nil {
			t.Fatal(err)
		}
	}
	views, err := store.ListTopics(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 1 || views[0].ID != "00020" || views[0].Storage != "open" || views[0].Path != ".historic/00020-current-slug" {
		t.Fatalf("open views = %#v", views)
	}
	closed, err := store.ListTopics(true)
	if err != nil {
		t.Fatal(err)
	}
	if len(closed) != 1 || closed[0].Storage != "closed" || closed[0].Path != ".historic/.database/00020-old-slug" {
		t.Fatalf("closed views = %#v, want the closed mirror version", closed)
	}
	fallback, err := store.ShowTopic(domain.ID("00020"), false)
	if err != nil || fallback.Storage != "open" {
		t.Fatalf("show prefers open location: view=%#v err=%v", fallback, err)
	}
}

func TestShowTopicFallsBackToClosedCopy(t *testing.T) {
	store := newTestStore(t)
	closedTopic := filepath.Join(store.Workspace.Database, "00022-closed-topic")
	if err := os.MkdirAll(closedTopic, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteTopicMetadata(filepath.Join(closedTopic, markdown.MetaFilename), markdown.TopicMetadata{ID: "00022", Title: "Closed Topic", Created: "2026-09-21"}); err != nil {
		t.Fatal(err)
	}
	view, err := store.ShowTopic(domain.ID("00022"), false)
	if err != nil || view.Storage != "closed" || view.Path != ".historic/.database/00022-closed-topic" {
		t.Fatalf("closed fallback view=%#v err=%v", view, err)
	}
}

func TestListTopicsRejectsSymlinkedTopicRoot(t *testing.T) {
	store := newTestStore(t)
	target := t.TempDir()
	if err := os.Symlink(target, filepath.Join(store.Workspace.Histories, "00021-linked-topic")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := store.ListTopics(false); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("symlink topic error = %v, want conflict", err)
	}
}

func TestListTopicsDefaultsToActiveAndSortsByLegacyIDWithFlag(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.CreateTopic("Second", "00002", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateTopic("First", "00001", ""); err != nil {
		t.Fatal(err)
	}
	archived := filepath.Join(store.Workspace.Database, "00003-archived")
	if err := os.MkdirAll(archived, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(archived, markdown.MetaFilename), []byte("id: 00003\ntitle: Archived\nstatus: archived\ncreated: 2026-09-18\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	views, err := store.ListTopicsSorted(false, ListSortID)
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 2 || views[0].ID != "00001" || views[1].ID != "00002" {
		t.Fatalf("views = %#v", views)
	}
	closed, err := store.ListTopics(true)
	if err != nil {
		t.Fatal(err)
	}
	if len(closed) != 1 || closed[0].ID != "00003" || closed[0].Storage != "closed" || closed[0].Active {
		t.Fatalf("closed views = %#v", closed)
	}
}

func writeListTopic(t *testing.T, root, folder, id, title, created, updated string) {
	t.Helper()
	path := filepath.Join(root, folder)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	metadata := markdown.TopicMetadata{ID: domain.ID(id), Title: title, Created: created, Updated: updated}
	if err := markdown.WriteTopicMetadata(filepath.Join(path, markdown.MetaFilename), metadata); err != nil {
		t.Fatal(err)
	}
}

func viewIDs(views []TopicView) []string {
	ids := make([]string, 0, len(views))
	for _, view := range views {
		ids = append(ids, view.ID+":"+view.Storage)
	}
	return ids
}

func TestListTopicsDefaultsToUpdatedDescendingWithCreatedFallback(t *testing.T) {
	store := newTestStore(t)
	writeListTopic(t, store.Workspace.Histories, "00001-old", "00001", "Old", "2026-01-01", "2026-01-05")
	writeListTopic(t, store.Workspace.Histories, "00002-newest", "00002", "Newest", "2026-01-01", "2026-03-01")
	writeListTopic(t, store.Workspace.Histories, "00003-no-updated", "00003", "Fallback", "2026-02-15", "")

	views, err := store.ListTopics(false)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"00002:open", "00003:open", "00001:open"}
	if got := viewIDs(views); !equalStrings(got, want) {
		t.Fatalf("default order = %v, want %v", got, want)
	}

	created, err := store.ListTopicsSorted(false, ListSortCreated)
	if err != nil {
		t.Fatal(err)
	}
	wantCreated := []string{"00003:open", "00002:open", "00001:open"}
	if got := viewIDs(created); !equalStrings(got, wantCreated) {
		t.Fatalf("created order = %v, want %v", got, wantCreated)
	}
}

func TestListTopicsTieBreaksByIDDescendingThenOpenFirstThenPath(t *testing.T) {
	store := newTestStore(t)
	writeListTopic(t, store.Workspace.Histories, "00005-alpha", "00005", "Alpha", "2026-01-01", "2026-02-01")
	writeListTopic(t, store.Workspace.Histories, "00004-beta", "00004", "Beta", "2026-01-01", "2026-02-01")

	views, err := store.ListTopics(false)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"00005:open", "00004:open"}
	if got := viewIDs(views); !equalStrings(got, want) {
		t.Fatalf("tie-break order = %v, want %v", got, want)
	}
}

func TestListTopicsSortsTitleAndRejectsUnknownKey(t *testing.T) {
	store := newTestStore(t)
	writeListTopic(t, store.Workspace.Histories, "00001-zulu", "00001", "Zulu", "2026-01-01", "2026-02-01")
	writeListTopic(t, store.Workspace.Histories, "00002-alpha", "00002", "alpha", "2026-01-01", "2026-01-01")

	byTitle, err := store.ListTopicsSorted(false, ListSortTitle)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"00002:open", "00001:open"}
	if got := viewIDs(byTitle); !equalStrings(got, want) {
		t.Fatalf("title order = %v, want %v", got, want)
	}

	if _, err := store.ListTopicsSorted(false, ListSort("nope")); !errors.Is(err, ErrInvalidListSort) {
		t.Fatalf("unknown sort error = %v", err)
	}
	if _, err := ParseListSort("nope"); err == nil || !strings.Contains(err.Error(), "use updated, created, title, or id") {
		t.Fatalf("validation message = %v", err)
	}
}

func TestListTopicsSortsClosedScope(t *testing.T) {
	store := newTestStore(t)
	writeListTopic(t, store.Workspace.Database, "00001-older", "00001", "Older", "2026-01-01", "2026-01-02")
	writeListTopic(t, store.Workspace.Database, "00002-newer", "00002", "Newer", "2026-01-01", "2026-04-02")

	views, err := store.ListTopicsSorted(true, ListSortUpdated)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"00002:closed", "00001:closed"}
	if got := viewIDs(views); !equalStrings(got, want) {
		t.Fatalf("closed order = %v, want %v", got, want)
	}
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func TestListTopicsComputesDerivedTopicStatus(t *testing.T) {
	store := newTestStore(t)
	topic, err := store.CreateTopic("Topic", "00030", "")
	if err != nil {
		t.Fatal(err)
	}
	write := func(name string, status domain.Status) {
		t.Helper()
		document, err := markdown.NewDocument(domain.Frontmatter{ID: "00030", Title: name, Status: status, Created: "2026-09-21"}, "body")
		if err != nil {
			t.Fatal(err)
		}
		if err := markdown.WriteFile(filepath.Join(topic.Path, name+".md"), document); err != nil {
			t.Fatal(err)
		}
	}
	write("moved", domain.StatusForwarded)
	write("done", domain.StatusComplete)

	views, err := store.ListTopics(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 1 || views[0].ComputedStatus != "complete" {
		t.Fatalf("list view = %#v, want computed complete", views)
	}
	// List views stay lightweight but still carry the derived status.
	if len(views[0].Files) != 0 {
		t.Fatalf("list view unexpectedly reads files: %#v", views[0].Files)
	}

	write("live", domain.StatusBlocked)
	view, err := store.ShowTopic(topic.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if view.ComputedStatus != "blocked" || len(view.Files) != 3 {
		t.Fatalf("show view = %#v, want computed blocked with files", view)
	}
}

func TestShowTopicIncludesFilesAndRejectsMissing(t *testing.T) {
	store := newTestStore(t)
	topic, err := store.CreateTopic("Topic", "00014", "")
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
