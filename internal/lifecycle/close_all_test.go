package lifecycle

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"
)

func makeCloseAllTopic(t *testing.T, workspace config.Workspace, id, folder string) string {
	t.Helper()
	path := filepath.Join(workspace.Histories, folder)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	document, err := markdown.NewDocument(domain.Frontmatter{
		ID: domain.ID(id), Title: "Topic " + id, Status: domain.StatusProgress, Created: "2026-09-27",
	}, "body")
	if err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(filepath.Join(path, markdown.MetaFilename), document); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCloseAllEmptyAndDeterministic(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(workspace)
	result, err := service.CloseAll()
	if err != nil || result.Total != 0 || result.Closed != 0 || result.FailedCount != 0 {
		t.Fatalf("empty result = %#v, err=%v", result, err)
	}

	first := makeCloseAllTopic(t, workspace, "00003", "00003-third")
	second := makeCloseAllTopic(t, workspace, "00001", "00001-first")
	third := makeCloseAllTopic(t, workspace, "00002", "00002-second")
	if err := os.MkdirAll(filepath.Join(workspace.Histories, "nested", "00000-child"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(workspace.Histories, ".staging-00000-topic"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace.Histories, "00000-file"), []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err = service.CloseAll()
	if err != nil {
		t.Fatalf("close all: %v", err)
	}
	if result.Total != 3 || result.Closed != 3 || result.FailedCount != 0 {
		t.Fatalf("result = %#v", result)
	}
	if got := []string{result.Succeeded[0].ID, result.Succeeded[1].ID, result.Succeeded[2].ID}; got[0] != "00001" || got[1] != "00002" || got[2] != "00003" {
		t.Fatalf("success order = %v", got)
	}
	for _, path := range []string{first, second, third, filepath.Join(workspace.Histories, "nested", "00000-child")} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) && path != filepath.Join(workspace.Histories, "nested", "00000-child") {
			t.Fatalf("open topic remains %s: %v", path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(workspace.Database, "00001-first")); err != nil {
		t.Fatalf("closed snapshot missing: %v", err)
	}
}

func TestCloseAllContinuesAfterFailure(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	badPath := makeCloseAllTopic(t, workspace, "00001", "00001-bad")
	goodPath := makeCloseAllTopic(t, workspace, "00002", "00002-good")
	staging := filepath.Join(workspace.Database, ".staging-close-00001-bad")
	if err := os.MkdirAll(staging, 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := NewService(workspace).CloseAll()
	if err == nil || result.Total != 2 || result.Closed != 1 || result.FailedCount != 1 {
		t.Fatalf("result = %#v, err=%v", result, err)
	}
	if result.Failed[0].ID != "00001" {
		t.Fatalf("failure = %#v", result.Failed)
	}
	if _, err := os.Stat(badPath); err != nil {
		t.Fatalf("failed source removed: %v", err)
	}
	if _, err := os.Stat(goodPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("next topic was not processed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspace.Database, "00002-good")); err != nil {
		t.Fatalf("successful snapshot missing: %v", err)
	}
}

func TestCloseAllReportsInvalidEntriesAndContinues(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	validPath := makeCloseAllTopic(t, workspace, "00002", "00002-valid")
	invalidPath := filepath.Join(workspace.Histories, "0000000000001-invalid")
	if err := os.MkdirAll(invalidPath, 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := NewService(workspace).CloseAll()
	if err == nil || result.Total != 2 || result.Closed != 1 || result.FailedCount != 1 {
		t.Fatalf("result = %#v, err=%v", result, err)
	}
	if result.Succeeded[0].ID != "00002" {
		t.Fatalf("successes = %#v", result.Succeeded)
	}
	if _, err := os.Stat(validPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("valid topic not closed: %v", err)
	}
	if _, err := os.Stat(invalidPath); err != nil {
		t.Fatalf("invalid folder was modified: %v", err)
	}
}

func TestCloseAllReportsSymlinkWithoutFollowingIt(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "outside")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(workspace.Histories, "00003-symlink")
	if err := os.Symlink(target, linkPath); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	result, err := NewService(workspace).CloseAll()
	if err == nil || result.Total != 1 || result.Closed != 0 || result.FailedCount != 1 || result.Failed[0].ID != "00003" {
		t.Fatalf("result = %#v, err=%v", result, err)
	}
	if _, err := os.Lstat(linkPath); err != nil {
		t.Fatalf("topic symlink was modified: %v", err)
	}
}
