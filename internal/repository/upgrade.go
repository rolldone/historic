package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"historic/internal/config"
	"historic/internal/identifier"
	"historic/internal/indexer"
	"historic/internal/lifecycle"
)

// UpgradeOptions controls the legacy workspace upgrade orchestrator.
type UpgradeOptions struct {
	DryRun     bool
	BackupDir  string
	NoFileIDs  bool
	NoTopicIDs bool
}

// UpgradeReport is the stable command result for one upgrade attempt.
type UpgradeReport struct {
	DryRun            bool     `json:"dry_run"`
	MigrationRequired bool     `json:"migration_required"`
	TopicsScanned     int      `json:"topics_scanned"`
	FileIDsMigrated   int      `json:"file_ids_migrated"`
	TopicIDsMigrated  int      `json:"topic_ids_migrated"`
	IndexRebuilt      bool     `json:"index_rebuilt"`
	BackupPath        string   `json:"backup_path"`
	AliasesWritten    int      `json:"aliases_written"`
	Updated           int      `json:"updated"`
	Warnings          []string `json:"warnings"`
}

type upgradeManifest struct {
	CreatedAt       string                    `json:"created_at"`
	AppVersionName  string                    `json:"app_version_name"`
	AppVersionCode  int                       `json:"app_version_code"`
	SourceWorkspace int                       `json:"source_workspace_format"`
	TargetWorkspace int                       `json:"target_workspace_format"`
	SourceSchema    int                       `json:"source_schema_version"`
	TargetSchema    int                       `json:"target_schema_version"`
	TopicMappings   []TopicIDMigrationMapping `json:"topic_mappings"`
	FileMappings    []lifecycle.FileIDMapping `json:"file_mappings"`
	Checksums       map[string]string         `json:"checksums"`
}

// UpgradeWorkspace executes the complete legacy workspace upgrade pipeline.
func UpgradeWorkspace(workspace config.Workspace, options UpgradeOptions) (UpgradeReport, error) {
	lock := identifier.NewFileLock(filepath.Join(workspace.Histories, ".upgrade.lock"))
	if err := lock.Acquire(context.Background()); err != nil {
		return UpgradeReport{}, fmt.Errorf("stage acquire upgrade lock: %w", err)
	}
	defer lock.Release()

	legacyTopics, err := scanLegacyTopics(workspace, "")
	if err != nil {
		return UpgradeReport{}, fmt.Errorf("stage scan topics: %w", err)
	}
	report := UpgradeReport{DryRun: options.DryRun, TopicsScanned: countTopicDirectories(workspace), Warnings: []string{}}
	fileService := lifecycle.NewService(workspace)
	var fileReport lifecycle.FileIDMigrationReport
	if !options.NoFileIDs {
		fileReport, err = fileService.MigrateFileIDs(lifecycle.FileIDMigrationOptions{DryRun: true})
		if err != nil {
			return UpgradeReport{}, fmt.Errorf("stage scan file IDs: %w", err)
		}
		report.FileIDsMigrated = fileReport.FilesMigrated
	}
	topicPlan, err := planTopicIDMigration(workspace, legacyTopics, migrationClock{})
	if err != nil && !options.NoTopicIDs {
		return UpgradeReport{}, fmt.Errorf("stage scan topic IDs: %w", err)
	}
	if !options.NoTopicIDs {
		report.TopicIDsMigrated = countUniqueNewIDs(topicPlan)
	}
	report.MigrationRequired = report.FileIDsMigrated > 0 || report.TopicIDsMigrated > 0

	if options.NoFileIDs {
		report.Warnings = append(report.Warnings, "--no-file-ids: legacy FileID entries were not migrated")
	}
	if options.NoTopicIDs {
		report.Warnings = append(report.Warnings, "--no-topic-ids: legacy TopicID folders were not migrated")
	}
	if options.DryRun || !report.MigrationRequired {
		return report, nil
	}

	backupPath, err := createUpgradeBackup(workspace, options, topicPlan, fileReport)
	if err != nil {
		return report, fmt.Errorf("stage backup: %w", err)
	}
	report.BackupPath = workspace.RelativePath(backupPath)
	if err := commitUpgrade(workspace, options, topicPlan); err != nil {
		return report, fmt.Errorf("stage commit: %w (backup: %s; remediation: restore backup and rerun historic upgrade)", err, report.BackupPath)
	}
	if _, _, err := indexer.Upgrade(workspace); err != nil {
		return report, fmt.Errorf("stage rebuild SQLite: %w (backup: %s)", err, report.BackupPath)
	}
	report.IndexRebuilt = true
	report.AliasesWritten = countUniqueNewIDs(topicPlan)
	report.Updated = report.FileIDsMigrated + report.TopicIDsMigrated
	return report, nil
}

func countTopicDirectories(workspace config.Workspace) int {
	count := 0
	for _, root := range []string{workspace.Histories, workspace.Database} {
		entries, _ := os.ReadDir(root)
		for _, entry := range entries {
			if entry.IsDir() && strings.Contains(entry.Name(), "-") && !strings.HasPrefix(entry.Name(), ".") {
				count++
			}
		}
	}
	return count
}

func countUniqueNewIDs(mappings []TopicIDMigrationMapping) int {
	seen := map[string]struct{}{}
	for _, mapping := range mappings {
		seen[mapping.NewID] = struct{}{}
	}
	return len(seen)
}

func createUpgradeBackup(workspace config.Workspace, options UpgradeOptions, topics []TopicIDMigrationMapping, files lifecycle.FileIDMigrationReport) (string, error) {
	root := options.BackupDir
	if root == "" {
		root = filepath.Join(workspace.Root, ".historic-upgrades", time.Now().UTC().Format("20060102T150405Z"))
	}
	if filepath.Clean(root) == filepath.Clean(workspace.Histories) || strings.HasPrefix(filepath.Clean(root), filepath.Clean(workspace.Histories)+string(filepath.Separator)) {
		return "", fmt.Errorf("backup directory must be outside .historic")
	}
	if err := os.MkdirAll(filepath.Join(root, "topics"), 0o755); err != nil {
		return "", err
	}
	for _, source := range []struct{ name, path string }{{"topics", workspace.Histories}, {"database", workspace.Database}} {
		if err := copyDirectory(source.path, filepath.Join(root, source.name)); err != nil {
			return "", fmt.Errorf("backup %s: %w", source.name, err)
		}
	}
	checksums, err := checksumTree(workspace.Histories)
	if err != nil {
		return "", err
	}
	manifest := upgradeManifest{CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), AppVersionName: config.VersionName, AppVersionCode: config.VersionCode, SourceWorkspace: config.WorkspaceFormatVersion - 1, TargetWorkspace: config.WorkspaceFormatVersion, SourceSchema: config.IndexSchemaVersion - 1, TargetSchema: config.IndexSchemaVersion, TopicMappings: topics, FileMappings: files.Mappings, Checksums: checksums}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), data, 0o644); err != nil {
		return "", err
	}
	return root, nil
}

func copyDirectory(source, destination string) error {
	info, err := os.Stat(source)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", source)
	}
	return filepath.Walk(source, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		defer input.Close()
		output, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(output, input)
		syncErr := output.Sync()
		closeErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		if syncErr != nil {
			return syncErr
		}
		return closeErr
	})
}

func checksumTree(root string) (map[string]string, error) {
	result := map[string]string{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || !info.Mode().IsRegular() {
			return nil
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		defer input.Close()
		hash := sha256.New()
		if _, err := io.Copy(hash, input); err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		result[filepath.ToSlash(rel)] = hex.EncodeToString(hash.Sum(nil))
		return nil
	})
	return result, err
}

func commitUpgrade(workspace config.Workspace, options UpgradeOptions, topicPlan []TopicIDMigrationMapping) error {
	if !options.NoFileIDs {
		if _, err := lifecycle.NewService(workspace).MigrateFileIDs(lifecycle.FileIDMigrationOptions{}); err != nil {
			return fmt.Errorf("migrate FileID: %w", err)
		}
	}
	if !options.NoTopicIDs && len(topicPlan) > 0 {
		if err := commitTopicIDMigration(workspace, mustScanLegacyTopics(workspace), topicPlan); err != nil {
			return fmt.Errorf("migrate TopicID: %w", err)
		}
	}
	return nil
}

func mustScanLegacyTopics(workspace config.Workspace) []legacyTopic {
	topics, _ := scanLegacyTopics(workspace, "")
	return topics
}
