package lifecycle

import (
	"os"
	"path/filepath"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"
)

func TestSyncMetaReconcilesRenameDeleteMoveAndClassification(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00001-topic")
	if err := os.MkdirAll(filepath.Join(topic, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	meta, _ := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Topic", Status: domain.StatusProgress, Created: "2026-09-19"}, "# Topic\n\n## Progress\n\nkeep this")
	if err := markdown.WriteFile(filepath.Join(topic, markdown.MetaFilename), meta); err != nil {
		t.Fatal(err)
	}
	managed, _ := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Managed", Status: domain.StatusProgress, Created: "2026-09-19"}, "managed")
	if err := markdown.WriteFile(filepath.Join(topic, "old.md"), managed); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(topic, "old.md"), filepath.Join(topic, "renamed.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(topic, "manual.txt"), []byte("asset"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(topic, "docs", "moved.md"), []byte("plain asset"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := SyncMeta(workspace, "00001"); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(topic, markdown.MetaFilename))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SyncMeta(workspace, "00001"); err != nil {
		t.Fatal(err)
	}
	updated, err := os.ReadFile(filepath.Join(topic, markdown.MetaFilename))
	if err != nil {
		t.Fatal(err)
	}
	if string(updated) != string(before) {
		t.Fatalf("canonical metadata changed during sync: %s", updated)
	}
	second, err := SyncMeta(workspace, "00001")
	if err != nil || second.Updated {
		t.Fatalf("second sync = %#v err=%v", second, err)
	}
}

func TestSyncMetaReconcilesManagedToAsset(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00001-topic")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	meta, _ := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Topic", Status: domain.StatusProgress, Created: "2026-09-19"}, "# Topic")
	managed, _ := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Changed", Status: domain.StatusProgress, Created: "2026-09-19"}, "body")
	if err := markdown.WriteFile(filepath.Join(topic, markdown.MetaFilename), meta); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(topic, "changed.md")
	if err := markdown.WriteFile(path, managed); err != nil {
		t.Fatal(err)
	}
	if _, err := SyncMeta(workspace, "00001"); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(topic, markdown.MetaFilename))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("now asset"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := SyncMeta(workspace, "00001"); err != nil {
		t.Fatal(err)
	}
	updated, err := os.ReadFile(filepath.Join(topic, markdown.MetaFilename))
	if err != nil {
		t.Fatal(err)
	}
	if string(updated) != string(before) {
		t.Fatalf("canonical metadata changed during classification sync: %s", updated)
	}
}
