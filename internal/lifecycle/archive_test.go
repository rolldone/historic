package lifecycle

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"
	"historic/internal/search"
)

func TestArchiveStatusChangesWorkStatusWithoutMovingTopic(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(workspace.Histories, "00001-topic")
	if err := os.MkdirAll(filepath.Join(source, "wos"), 0o755); err != nil {
		t.Fatal(err)
	}
	document, _ := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Topic", Status: domain.StatusProgress, Created: "2026-09-18"}, "meta")
	if err := markdown.WriteFile(filepath.Join(source, "_meta.md"), document); err != nil {
		t.Fatal(err)
	}
	entry, _ := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Task", Status: domain.StatusProgress, Created: "2026-09-18"}, "entry")
	if err := markdown.WriteFile(filepath.Join(source, "wos", "01-task.md"), entry); err != nil {
		t.Fatal(err)
	}
	change, err := NewService(workspace).ArchiveStatus("00001", domain.StatusComplete)
	if err != nil {
		t.Fatal(err)
	}
	if change.Archived || change.Current != domain.StatusComplete || change.Storage != domain.StorageOpen || change.Path != ".historic/00001-topic" {
		t.Fatalf("change = %#v", change)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(workspace.Database, "00001-topic")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("closed topic exists: %v", err)
	}
}

func TestCloseAndOpenPreserveStatusAndFiles(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(workspace.Histories, "00001-topic")
	if err := os.MkdirAll(filepath.Join(source, "wos"), 0o755); err != nil {
		t.Fatal(err)
	}
	document, _ := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Topic", Status: domain.StatusBlocked, Created: "2026-09-18"}, "meta")
	if err := markdown.WriteFile(filepath.Join(source, "_meta.md"), document); err != nil {
		t.Fatal(err)
	}
	content := []byte("entry bytes\n")
	if err := os.WriteFile(filepath.Join(source, "wos", "01-task.md"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	service := NewService(workspace)
	closed, err := service.Close("00001")
	if err != nil || closed.Storage != domain.StorageClosed {
		t.Fatalf("close = %#v err=%v", closed, err)
	}
	opened, err := service.Open("00001")
	if err != nil || opened.Storage != domain.StorageOpen || opened.Current != domain.StatusBlocked {
		t.Fatalf("open = %#v err=%v", opened, err)
	}
	if _, err := os.Stat(filepath.Join(workspace.Database, "00001-topic", "wos", "01-task.md")); err != nil {
		t.Fatalf("closed snapshot missing after open: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(source, "wos", "01-task.md"))
	if err != nil || string(got) != string(content) {
		t.Fatalf("content = %q err=%v", got, err)
	}
	closedMeta, err := markdown.ParseFile(filepath.Join(workspace.Database, "00001-topic", "_meta.md"))
	if err != nil || closedMeta.Frontmatter.Status != domain.StatusBlocked {
		t.Fatalf("closed metadata = %#v err=%v", closedMeta.Frontmatter, err)
	}
	if checksumFile(filepath.Join(source, "wos", "01-task.md")) != checksumFile(filepath.Join(workspace.Database, "00001-topic", "wos", "01-task.md")) {
		t.Fatal("open copy checksum differs from closed snapshot")
	}
}

func TestOpenCopyBasedIsolatedSmoke(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	closed := filepath.Join(workspace.Database, "00002-renamed-topic")
	if err := os.MkdirAll(filepath.Join(closed, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	meta, _ := markdown.NewDocument(domain.Frontmatter{ID: "00002", Title: "Smoke Topic", Status: domain.StatusComplete, Created: "2026-09-21"}, "manual body")
	if err := markdown.WriteFile(filepath.Join(closed, "_meta.md"), meta); err != nil {
		t.Fatal(err)
	}
	asset := []byte{0, 1, 2, 3, 255}
	if err := os.WriteFile(filepath.Join(closed, "nested", "asset.bin"), asset, 0o640); err != nil {
		t.Fatal(err)
	}

	change, err := NewService(workspace).Open("00002")
	if err != nil {
		t.Fatal(err)
	}
	if change.Path != ".historic/00002-renamed-topic" || change.Storage != domain.StorageOpen || change.PreviousStorage != domain.StorageClosed {
		t.Fatalf("change = %#v", change)
	}
	open := filepath.Join(workspace.Histories, "00002-renamed-topic")
	if checksumFile(filepath.Join(open, "nested", "asset.bin")) != checksumBytes(asset) {
		t.Fatal("isolated smoke copy changed asset bytes")
	}
	if _, err := os.Stat(closed); err != nil {
		t.Fatalf("closed snapshot was removed: %v", err)
	}
	results, err := search.Find(workspace, search.Options{Keyword: "manual"})
	if err != nil || len(results) != 1 || results[0].Storage != domain.StorageOpen.String() {
		t.Fatalf("search results = %#v err=%v", results, err)
	}
}

func TestOpenResolvesOpenCopyAsCurrentIdentity(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	closed := filepath.Join(workspace.Database, "00004-old-slug")
	open := filepath.Join(workspace.Histories, "00004-new-slug")
	for _, topic := range []string{closed, open} {
		if err := os.MkdirAll(topic, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for topic, body := range map[string]string{closed: "snapshot", open: "current"} {
		meta, _ := markdown.NewDocument(domain.Frontmatter{ID: "00004", Title: "Topic", Status: domain.StatusComplete, Created: "2026-09-21"}, body)
		if err := markdown.WriteFile(filepath.Join(topic, "_meta.md"), meta); err != nil {
			t.Fatal(err)
		}
	}
	path, storage, err := TopicLocation(workspace, "00004")
	if err != nil || storage != domain.StorageOpen || path != open {
		t.Fatalf("location = %q, %s, err=%v", path, storage, err)
	}
	if _, err := NewService(workspace).Open("00004"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("open with current copy error = %v, want conflict", err)
	}
}

func TestOpenRejectsSymlinkWithoutPartialWorkdirOrSnapshotDamage(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	closed := filepath.Join(workspace.Database, "00003-symlink-topic")
	if err := os.MkdirAll(closed, 0o755); err != nil {
		t.Fatal(err)
	}
	meta, _ := markdown.NewDocument(domain.Frontmatter{ID: "00003", Title: "Symlink Topic", Status: domain.StatusComplete, Created: "2026-09-21"}, "body")
	metaPath := filepath.Join(closed, "_meta.md")
	if err := markdown.WriteFile(metaPath, meta); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(target, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(closed, "linked.txt")); err != nil {
		t.Fatal(err)
	}
	before := checksumFile(metaPath)
	if _, err := NewService(workspace).Open("00003"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("open error = %v, want conflict", err)
	}
	if _, err := os.Lstat(filepath.Join(workspace.Histories, "00003-symlink-topic")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial workdir exists: %v", err)
	}
	if checksumFile(metaPath) != before {
		t.Fatal("closed snapshot changed after rejected open")
	}
}

func checksumFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return checksumBytes(data)
}

func checksumBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func TestArchiveStatusRejectsDestinationConflictWithoutDeletingSource(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(workspace.Histories, "00001-topic")
	destination := filepath.Join(workspace.Database, "00001-topic")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	document, _ := markdown.NewDocument(domain.Frontmatter{ID: "00001", Title: "Topic", Status: domain.StatusProgress, Created: "2026-09-18"}, "meta")
	if err := markdown.WriteFile(filepath.Join(source, "_meta.md"), document); err != nil {
		t.Fatal(err)
	}
	if _, err := NewService(workspace).ArchiveStatus("00001", domain.StatusComplete); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("error = %v, want conflict", err)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("source removed: %v", err)
	}
}
