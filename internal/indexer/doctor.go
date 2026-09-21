package indexer

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"historic/internal/config"
)

type Diagnosis struct {
	BinaryVersion       string
	ExecutablePath      string
	WorkspaceFormat     int
	CurrentIndexSchema  int
	RequiredIndexSchema int
	Status              string
	MarkdownValid       bool
	IndexExists         bool
	Recommendation      string
}

func Diagnose(workspace config.Workspace, binaryVersion string) Diagnosis {
	diagnosis := Diagnosis{BinaryVersion: binaryVersion, RequiredIndexSchema: SchemaVersion, MarkdownValid: true}
	diagnosis.ExecutablePath, _ = os.Executable()
	if _, err := os.Stat(workspace.Histories); err != nil {
		diagnosis.Status = "workspace invalid"
		diagnosis.Recommendation = "run historic init in a valid workspace"
		return diagnosis
	}
	diagnosis.WorkspaceFormat = config.WorkspaceFormatVersion
	if _, err := os.Stat(workspace.Index); errors.Is(err, os.ErrNotExist) {
		diagnosis.Status = "rebuild required"
		diagnosis.Recommendation = "run historic rebuild"
		return diagnosis
	}
	diagnosis.IndexExists = true
	database, err := sql.Open("sqlite", workspace.Index)
	if err != nil {
		diagnosis.Status = "rebuild required"
		diagnosis.Recommendation = "run historic rebuild"
		return diagnosis
	}
	defer database.Close()
	if err := database.Ping(); err != nil {
		diagnosis.Status = "rebuild required"
		diagnosis.Recommendation = "run historic rebuild"
		return diagnosis
	}
	version, err := schemaVersion(database)
	if err != nil {
		diagnosis.Status = "rebuild required"
		diagnosis.Recommendation = "run historic upgrade to recreate the index schema from Markdown"
		return diagnosis
	}
	diagnosis.CurrentIndexSchema = version
	switch {
	case version < SchemaVersion:
		diagnosis.Status = "upgrade required"
		diagnosis.Recommendation = "run historic upgrade"
	case version > SchemaVersion:
		diagnosis.Status = "binary too old"
		diagnosis.Recommendation = "install a newer Historic binary"
	default:
		diagnosis.Status = "compatible"
		diagnosis.Recommendation = "no action required"
	}
	return diagnosis
}

func Upgrade(workspace config.Workspace) (int, string, error) {
	unlock, err := acquireUpgradeLock(workspace)
	if err != nil {
		return 0, "", err
	}
	defer unlock()
	if _, err := os.Stat(workspace.Histories); err != nil {
		return 0, "", fmt.Errorf("workspace invalid: %w", err)
	}
	count, err := rebuildToTemporaryAndReplace(workspace)
	if err != nil {
		return 0, "", err
	}
	return count, workspace.RelativePath(workspace.Index), nil
}

func rebuildToTemporaryAndReplace(workspace config.Workspace) (int, error) {
	if err := os.MkdirAll(filepath.Dir(workspace.Index), 0o755); err != nil {
		return 0, err
	}
	temporary, err := os.CreateTemp(filepath.Dir(workspace.Index), ".index.sqlite.tmp-")
	if err != nil {
		return 0, err
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
	backup := workspace.Index + ".backup"
	if _, err := os.Stat(workspace.Index); err == nil {
		if _, err := os.Stat(backup); err == nil {
			_ = os.Remove(backup)
		}
		if err := os.Rename(workspace.Index, backup); err != nil {
			return 0, err
		}
		if err := os.Rename(temporaryPath, workspace.Index); err != nil {
			_ = os.Rename(backup, workspace.Index)
			return 0, err
		}
	} else if errors.Is(err, os.ErrNotExist) {
		if err := os.Rename(temporaryPath, workspace.Index); err != nil {
			return 0, err
		}
	} else {
		return 0, err
	}
	return count, nil
}
