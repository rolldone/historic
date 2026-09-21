package indexer

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"historic/internal/config"
)

var ErrUpgradeLocked = errors.New("index upgrade is already in progress")

func lockPath(workspace config.Workspace) string { return workspace.Index + ".upgrade.lock" }

func acquireUpgradeLock(workspace config.Workspace) (func(), error) {
	file, err := os.OpenFile(lockPath(workspace), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, ErrUpgradeLocked
		}
		return nil, fmt.Errorf("create index upgrade lock: %w", err)
	}
	_ = file.Close()
	return func() { _ = os.Remove(lockPath(workspace)) }, nil
}

func atomicRebuild(workspace config.Workspace) (int, error) {
	unlock, err := acquireUpgradeLock(workspace)
	if err != nil {
		return 0, err
	}
	defer unlock()
	return atomicRebuildUnlocked(workspace)
}
func atomicRebuildUnlocked(workspace config.Workspace) (int, error) {
	if err := os.MkdirAll(filepath.Dir(workspace.Index), 0o755); err != nil {
		return 0, err
	}
	temporary, err := os.CreateTemp(filepath.Dir(workspace.Index), ".index.sqlite.tmp-")
	if err != nil {
		return 0, fmt.Errorf("create temporary index: %w", err)
	}
	temporaryPath := temporary.Name()
	if err := temporary.Close(); err != nil {
		_ = os.Remove(temporaryPath)
		return 0, err
	}
	defer os.Remove(temporaryPath)

	count, err := rebuildInto(config.Workspace{Root: workspace.Root, Histories: workspace.Histories, Database: workspace.Database, Index: temporaryPath})
	if err != nil {
		return 0, err
	}
	if err := validateDatabase(temporaryPath); err != nil {
		return 0, err
	}
	if _, err := os.Stat(workspace.Index); err == nil {
		backup := fmt.Sprintf("%s.backup", workspace.Index)
		if err := os.Rename(workspace.Index, backup); err != nil {
			return 0, fmt.Errorf("backup index: %w", err)
		}
		if err := os.Rename(temporaryPath, workspace.Index); err != nil {
			_ = os.Rename(backup, workspace.Index)
			return 0, fmt.Errorf("replace index: %w", err)
		}
	} else if errors.Is(err, os.ErrNotExist) {
		if err := os.Rename(temporaryPath, workspace.Index); err != nil {
			return 0, fmt.Errorf("install index: %w", err)
		}
	} else {
		return 0, err
	}
	return count, nil
}

func validateDatabase(path string) error {
	database, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer database.Close()
	version, err := schemaVersion(database)
	if err != nil {
		return fmt.Errorf("read rebuilt index schema: %w", err)
	}
	if version != SchemaVersion {
		return fmt.Errorf("rebuilt index schema %d, required %d", version, SchemaVersion)
	}
	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM index_records").Scan(&count); err != nil {
		return fmt.Errorf("validate rebuilt index: %w", err)
	}
	return nil
}
