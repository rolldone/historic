package lifecycle

import (
	"os"
	"path/filepath"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/gitproxy"
	"historic/internal/markdown"
)

func TestValidateRestorePath(t *testing.T) {
	for _, path := range []string{"", "../topic", "/tmp/topic", "topic/file"} {
		if err := validateRestorePath(path); err == nil {
			t.Errorf("validateRestorePath(%q) accepted", path)
		}
	}
	if err := validateRestorePath("00001-topic"); err != nil {
		t.Fatal(err)
	}
}

func TestRestoreRejectsEmptySnapshotWithoutChangingWorkspace(t *testing.T) {
	workspace := config.NewWorkspace(t.TempDir())
	if err := Restore(workspace, domain.ID("00001"), "", false); err == nil {
		t.Fatal("empty snapshot accepted")
	}
}

func TestRestoreResolvesSnapshotFolder(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(workspace.Database, "00001-restore-topic")
	if err := os.MkdirAll(archive, 0o755); err != nil {
		t.Fatal(err)
	}
	meta, _ := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Restore Topic", Status: domain.StatusComplete, Created: "2026-09-18"}, "snapshot")
	if err := markdown.WriteFile(filepath.Join(archive, "_meta.md"), meta); err != nil {
		t.Fatal(err)
	}
	repository, err := gitproxy.Open(workspace.Database)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.AddAll(); err != nil {
		t.Fatal(err)
	}
	commit, err := repository.Commit("snapshot")
	if err != nil {
		t.Fatal(err)
	}
	if err := Restore(workspace, "00001", commit, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(workspace.Histories, "00001-restore-topic", "_meta.md")); err != nil {
		t.Fatal(err)
	}
}
