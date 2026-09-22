package indexer

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"historic/internal/config"
	"historic/internal/markdown"
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

func RebuildWithPreparation(workspace config.Workspace, prepare func() error) (int, error) {
	return atomicRebuildWithPreparation(workspace, func() (func(), error) {
		if err := prepare(); err != nil {
			return nil, err
		}
		return nil, nil
	})
}

// RebuildWithRollbackPreparation runs a preparation step and rolls back its
// filesystem changes if validation or index replacement fails.
func RebuildWithRollbackPreparation(workspace config.Workspace, prepare func() (func(), error)) (int, error) {
	return atomicRebuildWithPreparation(workspace, prepare)
}

func atomicRebuild(workspace config.Workspace) (int, error) {
	return atomicRebuildWithPreparation(workspace, nil)
}

func atomicRebuildWithPreparation(workspace config.Workspace, prepare func() (func(), error)) (int, error) {
	unlock, err := acquireUpgradeLock(workspace)
	if err != nil {
		return 0, err
	}
	defer unlock()

	var rollback func()
	if prepare != nil {
		rollback, err = prepare()
		if err != nil {
			return 0, err
		}
		if rollback != nil {
			defer func() {
				if err != nil {
					rollback()
				}
			}()
		}
		if err = validateRebuildSource(workspace); err != nil {
			return 0, err
		}
	}
	count, err := atomicRebuildUnlocked(workspace)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func validateRebuildSource(workspace config.Workspace) error {
	if _, err := scan(workspace); err != nil {
		return err
	}
	for _, root := range []string{workspace.Histories, workspace.Database} {
		entries, err := os.ReadDir(root)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("validate %s: %w", workspace.RelativePath(root), err)
		}
		for _, entry := range entries {
			if !topicFolderPattern.MatchString(entry.Name()) || !entry.IsDir() {
				continue
			}
			metaPath := filepath.Join(root, entry.Name(), markdown.MetaFilename)
			if _, err := markdown.ParseTopicMetadataFile(metaPath); err != nil {
				return fmt.Errorf("invalid topic metadata %s: %w", workspace.RelativePath(metaPath), err)
			}
		}
	}
	return nil
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
			if restoreErr := os.Rename(backup, workspace.Index); restoreErr != nil {
				return 0, fmt.Errorf("replace index: %w; restore backup: %v", err, restoreErr)
			}
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
