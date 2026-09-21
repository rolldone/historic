package lifecycle

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"
	"historic/internal/query"
	"historic/internal/search"
)

func setupDeleteTopic(t *testing.T) (config.Workspace, string) {
	t.Helper()
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00001-delete-topic")
	if err := os.MkdirAll(filepath.Join(topic, "wos"), 0o755); err != nil {
		t.Fatal(err)
	}
	meta, err := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Delete Topic", Status: domain.StatusProgress, Created: "2026-09-21"}, "topic body")
	if err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(filepath.Join(topic, markdown.MetaFilename), meta); err != nil {
		t.Fatal(err)
	}
	entry, err := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Delete Task", Status: domain.StatusProgress, Created: "2026-09-21"}, "unique-delete-content")
	if err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(filepath.Join(topic, "wos", "01-task.md"), entry); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(topic, "asset.bin"), []byte("asset-only"), 0o644); err != nil {
		t.Fatal(err)
	}
	return workspace, topic
}

func TestDeleteFileRebuildsReadModelAndIgnoresAssets(t *testing.T) {
	workspace, topic := setupDeleteTopic(t)
	service := NewService(workspace)
	if _, err := service.DeleteFile("wos/01-task.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(topic, "wos", "01-task.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("deleted file still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(topic, "asset.bin")); err != nil {
		t.Fatalf("asset was affected by managed file delete: %v", err)
	}
	results, err := search.Find(workspace, search.Options{Keyword: "unique-delete-content", OpenOnly: true})
	if err != nil || len(results) != 0 {
		t.Fatalf("search after delete = %#v err=%v", results, err)
	}
	summaries, err := query.NewService(workspace).QueryTopics(query.Options{TopicID: "00001"})
	if err != nil || len(summaries) != 1 || summaries[0].TotalFiles != 0 {
		t.Fatalf("aggregate after delete = %#v err=%v", summaries, err)
	}
	if _, err := os.Stat(filepath.Join(topic, markdown.MetaFilename)); err != nil {
		t.Fatalf("canonical metadata missing after delete: %v", err)
	}
}

func TestDeleteRejectsClosedStorageAndRequiresScopeForCopies(t *testing.T) {
	workspace, _ := setupDeleteTopic(t)
	service := NewService(workspace)
	if _, err := service.Close("00001"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Open("00001"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.DeleteFile(".historic/.database/00001-delete-topic/wos/01-task.md"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("closed file delete error = %v, want conflict", err)
	}
	if _, err := service.DeleteTopic("00001", ""); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("unscoped duplicate delete error = %v, want conflict", err)
	}
	change, err := service.DeleteTopic("00001", domain.StorageOpen)
	if err != nil || change.Storage != domain.StorageOpen {
		t.Fatalf("scoped open delete = %#v err=%v", change, err)
	}
	if _, err := os.Stat(filepath.Join(workspace.Database, "00001-delete-topic")); err != nil {
		t.Fatalf("closed snapshot changed by open delete: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspace.Database, ".git")); err != nil {
		t.Fatalf("internal git was removed: %v", err)
	}
}

func TestDeleteRejectsTraversalAndSymlink(t *testing.T) {
	workspace, topic := setupDeleteTopic(t)
	service := NewService(workspace)
	for _, input := range []string{"../outside.md", ".historic/.database/.git/config"} {
		if _, err := service.DeleteFile(input); !errors.Is(err, domain.ErrConflict) {
			t.Fatalf("delete %q error = %v, want conflict", input, err)
		}
	}
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(topic, "linked.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := service.DeleteFile("linked.md"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("symlink delete error = %v, want conflict", err)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("outside symlink target changed: %v", err)
	}
}
