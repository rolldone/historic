package lifecycle

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/identifier"
	"historic/internal/markdown"
)

type testClock struct{ value int64 }

func (clock testClock) Now() time.Time { return time.UnixMilli(clock.value) }

func newTestGenerator() identifier.Generator {
	return identifier.NewUUIDv7Generator(testClock{value: 1_700_000_000_000}, bytes.NewReader(bytes.Repeat([]byte{1}, 80)))
}

func writeLegacyManifest(t *testing.T, path, topicID, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestMigrateFileIDsDryRunPreservesBytes(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00001-topic")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := "id: \"00001\"\ntitle: Topic\ncreated: \"2026-09-22\"\nfiles:\n  - id: \"\"\n    path: note.md\n    type: note\n    status: progress\nassets: []\n"
	writeLegacyManifest(t, filepath.Join(topic, markdown.MetaFilename), "00001", legacy)
	before, err := os.ReadFile(filepath.Join(topic, markdown.MetaFilename))
	if err != nil {
		t.Fatal(err)
	}
	options := FileIDMigrationOptions{DryRun: true}
	report, err := NewService(workspace).MigrateFileIDs(options)
	if err != nil {
		t.Fatal(err)
	}
	if !report.DryRun {
		t.Fatal("dry_run should be true")
	}
	if report.FilesMigrated != 1 {
		t.Fatalf("files_migrated = %d, want 1", report.FilesMigrated)
	}
	if report.ManifestsUpdated != 0 {
		t.Fatalf("manifests_updated = %d, want 0", report.ManifestsUpdated)
	}
	after, err := os.ReadFile(filepath.Join(topic, markdown.MetaFilename))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("dry-run modified metadata bytes")
	}
}

func TestMigrateFileIDsAssignsNewUUIDsAndReportsLegacyID(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00002-topic")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := "id: \"00002\"\ntitle: Topic\ncreated: \"2026-09-22\"\nfiles:\n  - id: \"00001\"\n    path: wos/01-task.md\n    type: task\n    status: progress\nassets: []\n"
	writeLegacyManifest(t, filepath.Join(topic, markdown.MetaFilename), "00002", legacy)
	report, err := NewService(workspace).MigrateFileIDs(FileIDMigrationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if report.FilesMigrated != 1 {
		t.Fatalf("files_migrated = %d, want 1", report.FilesMigrated)
	}
	if report.ManifestsUpdated != 1 {
		t.Fatalf("manifests_updated = %d, want 1", report.ManifestsUpdated)
	}
	if len(report.Mappings) != 1 {
		t.Fatalf("mappings = %d, want 1", len(report.Mappings))
	}
	mapping := report.Mappings[0]
	if mapping.LegacyID != "00001" {
		t.Fatalf("legacy_id = %q, want 00001", mapping.LegacyID)
	}
	if !mapping.FileID.Valid() {
		t.Fatalf("file_id = %q, want valid UUIDv7", mapping.FileID)
	}
	if mapping.Action != "assign" {
		t.Fatalf("action = %q, want assign", mapping.Action)
	}
	metadata, err := markdown.ParseTopicMetadataFile(filepath.Join(topic, markdown.MetaFilename))
	if err != nil {
		t.Fatal(err)
	}
	if metadata.ID != "00002" {
		t.Fatalf("topic ID changed to %q", metadata.ID)
	}
	if len(metadata.Files) != 1 || !metadata.Files[0].ID.Valid() {
		t.Fatalf("manifest = %#v", metadata.Files)
	}
}

func TestMigrateFileIDsPreservesExistingUUIDs(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00003-topic")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	existing := "01a0c82f-b6ff-7cb8-9b94-539ccf40de6b"
	legacy := "id: \"00003\"\ntitle: Topic\ncreated: \"2026-09-22\"\nfiles:\n  - id: " + existing + "\n    path: note.md\n    type: note\n    status: progress\nassets: []\n"
	writeLegacyManifest(t, filepath.Join(topic, markdown.MetaFilename), "00003", legacy)
	report, err := NewService(workspace).MigrateFileIDs(FileIDMigrationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if report.FilesPreserved != 1 {
		t.Fatalf("files_preserved = %d, want 1", report.FilesPreserved)
	}
	if report.FilesMigrated != 0 {
		t.Fatalf("files_migrated = %d, want 0", report.FilesMigrated)
	}
	metadata, err := markdown.ParseTopicMetadataFile(filepath.Join(topic, markdown.MetaFilename))
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Files[0].ID.String() != existing {
		t.Fatalf("UUID changed to %q", metadata.Files[0].ID)
	}
}

func TestMigrateFileIDsIdempotentRerunProducesZeroMigrated(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00004-topic")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := "id: \"00004\"\ntitle: Topic\ncreated: \"2026-09-22\"\nfiles:\n  - id: \"\"\n    path: note.md\n    type: note\n    status: progress\nassets: []\n"
	writeLegacyManifest(t, filepath.Join(topic, markdown.MetaFilename), "00004", legacy)
	if _, err := NewService(workspace).MigrateFileIDs(FileIDMigrationOptions{}); err != nil {
		t.Fatal(err)
	}
	report, err := NewService(workspace).MigrateFileIDs(FileIDMigrationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if report.FilesMigrated != 0 {
		t.Fatalf("files_migrated = %d, want 0", report.FilesMigrated)
	}
	if report.ManifestsUpdated != 0 {
		t.Fatalf("manifests_updated = %d, want 0", report.ManifestsUpdated)
	}
}

func TestMigrateFileIDsRejectsDuplicatePaths(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00005-topic")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	duplicate := "id: \"00005\"\ntitle: Topic\ncreated: \"2026-09-22\"\nfiles:\n  - id: \"\"\n    path: note.md\n    type: note\n    status: progress\n  - id: \"\"\n    path: note.md\n    type: note\n    status: complete\nassets: []\n"
	writeLegacyManifest(t, filepath.Join(topic, markdown.MetaFilename), "00005", duplicate)
	if _, err := NewService(workspace).MigrateFileIDs(FileIDMigrationOptions{}); err == nil {
		t.Fatal("duplicate path should be rejected")
	}
}

func TestMigrateFileIDsRejectsDuplicateUUIDs(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00006-topic")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	uuid := "01a0c82f-b6ff-7cb8-9b94-539ccf40de6b"
	duplicate := "id: \"00006\"\ntitle: Topic\ncreated: \"2026-09-22\"\nfiles:\n  - id: " + uuid + "\n    path: note.md\n    type: note\n    status: progress\n  - id: " + uuid + "\n    path: other.md\n    type: note\n    status: complete\nassets: []\n"
	writeLegacyManifest(t, filepath.Join(topic, markdown.MetaFilename), "00006", duplicate)
	if _, err := NewService(workspace).MigrateFileIDs(FileIDMigrationOptions{}); err == nil {
		t.Fatal("duplicate UUID should be rejected")
	}
}

func TestMigrateFileIDsFilterTopicID(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct{ id, title string }{{"00001", "A"}, {"00002", "B"}} {
		topic := filepath.Join(workspace.Histories, item.id+"-"+item.title)
		if err := os.MkdirAll(topic, 0o755); err != nil {
			t.Fatal(err)
		}
		writeLegacyManifest(t, filepath.Join(topic, markdown.MetaFilename), item.id, "id: \""+item.id+"\"\ntitle: "+item.title+"\ncreated: \"2026-09-22\"\nfiles:\n  - id: \"\"\n    path: note.md\n    type: note\n    status: progress\nassets: []\n")
	}
	id := domain.ID("00001")
	report, err := NewService(workspace).MigrateFileIDs(FileIDMigrationOptions{TopicID: &id})
	if err != nil {
		t.Fatal(err)
	}
	if report.FilesMigrated != 1 {
		t.Fatalf("files_migrated = %d, want 1", report.FilesMigrated)
	}
}

func TestMigrateFileIDsRejectsInvalidStorageScope(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	storage := domain.StorageState("unknown")
	if _, err := NewService(workspace).MigrateFileIDs(FileIDMigrationOptions{Storage: &storage}); err == nil {
		t.Fatal("invalid storage should be rejected")
	}
}

func TestMigrateFileIDSHumanAndJSONOutputAreStable(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00001-smoke")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	writeLegacyManifest(t, filepath.Join(topic, markdown.MetaFilename), "00001", "id: \"00001\"\ntitle: Smoke\ncreated: \"2026-09-22\"\nfiles:\n  - id: \"\"\n    path: wos/01-task.md\n    type: task\n    status: progress\nassets: []\n")
	report, err := NewService(workspace).MigrateFileIDs(FileIDMigrationOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Mappings) != 1 {
		t.Fatalf("mappings = %d, want 1", len(report.Mappings))
	}
	if report.Mappings[0].TopicID != "00001" || report.Mappings[0].Path != "wos/01-task.md" || report.Mappings[0].Action != "assign" {
		t.Fatalf("mapping = %+v", report.Mappings[0])
	}
}
