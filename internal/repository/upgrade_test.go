package repository

import (
	"os"
	"path/filepath"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"
)

func writeUpgradeLegacyTopic(t *testing.T, workspace config.Workspace, id, title string) string {
	t.Helper()
	path := filepath.Join(workspace.Histories, id+"-"+title)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	metadata := markdown.TopicMetadata{ID: domain.ID(id), Title: title, Created: "2026-09-22", Files: []markdown.ManifestFile{}, Assets: []markdown.ManifestAsset{}}
	if err := markdown.WriteTopicMetadata(filepath.Join(path, markdown.MetaFilename), metadata); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestUpgradeWorkspaceDryRunIsNonDestructive(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := writeUpgradeLegacyTopic(t, workspace, "00001", "legacy-topic")
	before, err := os.ReadFile(filepath.Join(path, markdown.MetaFilename))
	if err != nil {
		t.Fatal(err)
	}
	report, err := UpgradeWorkspace(workspace, UpgradeOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if !report.MigrationRequired || report.TopicIDsMigrated != 1 {
		t.Fatalf("report = %+v", report)
	}
	after, err := os.ReadFile(filepath.Join(path, markdown.MetaFilename))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("dry-run changed metadata")
	}
	if _, err := os.Stat(filepath.Join(workspace.Histories, ".topic-id-aliases.yaml")); !os.IsNotExist(err) {
		t.Fatalf("dry-run wrote aliases: %v", err)
	}
}

func TestUpgradeWorkspaceMigratesTopicAndIsIdempotent(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeUpgradeLegacyTopic(t, workspace, "00001", "legacy-topic")
	first, err := UpgradeWorkspace(workspace, UpgradeOptions{BackupDir: filepath.Join(t.TempDir(), "backup")})
	if err != nil {
		t.Fatal(err)
	}
	if first.TopicIDsMigrated != 1 || first.AliasesWritten != 1 || !first.IndexRebuilt {
		t.Fatalf("first report = %+v", first)
	}
	modernEntries, err := os.ReadDir(workspace.Histories)
	if err != nil {
		t.Fatal(err)
	}
	foundModern := false
	for _, entry := range modernEntries {
		if _, ok := domain.TopicIDFromFolder(entry.Name()); ok {
			foundModern = true
		}
	}
	if !foundModern {
		t.Fatal("modern topic folder not found")
	}
	second, err := UpgradeWorkspace(workspace, UpgradeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if second.MigrationRequired || second.Updated != 0 || second.AliasesWritten != 0 {
		t.Fatalf("second report = %+v", second)
	}
}

func TestUpgradeWorkspaceRejectsBackupInsideHistoric(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeUpgradeLegacyTopic(t, workspace, "00001", "legacy-topic")
	_, err = UpgradeWorkspace(workspace, UpgradeOptions{BackupDir: workspace.Histories})
	if err == nil {
		t.Fatal("backup inside .historic was accepted")
	}
}
