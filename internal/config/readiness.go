package config

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ReadinessCode identifies the result of a workspace preflight.
type ReadinessCode string

const (
	ReadinessReady                ReadinessCode = "ready"
	ReadinessNotHistoricWorkspace ReadinessCode = "not_historic_workspace"
	ReadinessLegacyConflict       ReadinessCode = "legacy_conflict"
	ReadinessMissingIndex         ReadinessCode = "missing_index"
	ReadinessInvalidIndex         ReadinessCode = "invalid_index"
	ReadinessInvalidStructure     ReadinessCode = "invalid_structure"
)

// ReadinessError is a stable, actionable error returned by workspace preflight.
type ReadinessError struct {
	Code    ReadinessCode
	Message string
	Detail  string
}

func (err *ReadinessError) Error() string {
	if err == nil {
		return ""
	}
	return err.Message
}

// WorkspaceReadiness is the result of a reject-only workspace validation.
type WorkspaceReadiness struct {
	State     ReadinessCode
	Workspace Workspace
}

// DiscoverRootForReadiness finds the nearest directory containing either the
// canonical or legacy workspace marker. It never creates or changes anything.
func DiscoverRootForReadiness(start string) (string, error) {
	root, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve workspace root: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return "", fmt.Errorf("inspect workspace root %s: %w", root, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%w: %s", ErrNotDirectory, root)
	}

	for current := root; ; current = filepath.Dir(current) {
		canonicalExists := pathExists(filepath.Join(current, HistoricDirName))
		legacyExists := pathExists(filepath.Join(current, LegacyHistoriesDirName))
		if canonicalExists || legacyExists {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return root, nil
		}
	}
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ValidateWorkspace performs a read-only, reject-only readiness check. It
// does not initialize directories, run rebuild, or open a writable database.
func ValidateWorkspace(workspace Workspace) (WorkspaceReadiness, error) {
	if err := validateWorkspaceRoot(workspace); err != nil {
		return WorkspaceReadiness{State: ReadinessInvalidStructure, Workspace: workspace}, err
	}

	canonicalInfo, canonicalErr := os.Stat(workspace.Histories)
	legacyPath := filepath.Join(workspace.Root, LegacyHistoriesDirName)
	_, legacyErr := os.Stat(legacyPath)
	if canonicalErr == nil && legacyErr == nil {
		return reject(ReadinessLegacyConflict, "Pisahkan atau migrasikan .histories secara manual; Historic tidak menggabungkan direktori otomatis.", "canonical and legacy workspace directories both exist", workspace)
	}
	if errors.Is(canonicalErr, os.ErrNotExist) {
		return reject(ReadinessNotHistoricWorkspace, "Jalankan historic init terlebih dahulu.", "canonical .historic directory is missing", workspace)
	}
	if canonicalErr != nil {
		return reject(ReadinessInvalidStructure, "Perbaiki struktur workspace .historic terlebih dahulu.", canonicalErr.Error(), workspace)
	}
	if !canonicalInfo.IsDir() {
		return reject(ReadinessInvalidStructure, "Perbaiki struktur workspace .historic terlebih dahulu.", ".historic exists but is not a directory", workspace)
	}
	if legacyErr != nil && !errors.Is(legacyErr, os.ErrNotExist) {
		return reject(ReadinessInvalidStructure, "Perbaiki struktur workspace terlebih dahulu.", legacyErr.Error(), workspace)
	}

	if err := validateMinimumStructure(workspace); err != nil {
		return reject(ReadinessInvalidStructure, "Perbaiki struktur minimum workspace Historic terlebih dahulu.", err.Error(), workspace)
	}

	indexInfo, err := os.Stat(workspace.Index)
	if errors.Is(err, os.ErrNotExist) {
		return reject(ReadinessMissingIndex, "Jalankan historic rebuild terlebih dahulu.", "index file is missing", workspace)
	}
	if err != nil {
		return reject(ReadinessInvalidIndex, "Jalankan historic rebuild terlebih dahulu.", err.Error(), workspace)
	}
	if !indexInfo.Mode().IsRegular() {
		return reject(ReadinessInvalidIndex, "Jalankan historic rebuild terlebih dahulu.", "index path is not a regular file", workspace)
	}
	if err := validateReadableIndex(workspace.Index); err != nil {
		return reject(ReadinessInvalidIndex, "Jalankan historic rebuild terlebih dahulu.", err.Error(), workspace)
	}
	return WorkspaceReadiness{State: ReadinessReady, Workspace: workspace}, nil
}

// ValidateRebuildWorkspace is the separate preflight used by rebuild. It
// intentionally permits a missing or damaged index so rebuild can recover it.
func ValidateRebuildWorkspace(workspace Workspace) error {
	if err := validateWorkspaceRoot(workspace); err != nil {
		return err
	}
	canonicalInfo, canonicalErr := os.Stat(workspace.Histories)
	_, legacyErr := os.Stat(filepath.Join(workspace.Root, LegacyHistoriesDirName))
	if canonicalErr == nil && legacyErr == nil {
		return &ReadinessError{Code: ReadinessLegacyConflict, Message: "Pisahkan atau migrasikan .histories secara manual; Historic tidak menggabungkan direktori otomatis.", Detail: "canonical and legacy workspace directories both exist"}
	}
	if errors.Is(canonicalErr, os.ErrNotExist) {
		return &ReadinessError{Code: ReadinessNotHistoricWorkspace, Message: "Jalankan historic init terlebih dahulu.", Detail: "canonical .historic directory is missing"}
	}
	if canonicalErr != nil {
		return &ReadinessError{Code: ReadinessInvalidStructure, Message: "Perbaiki struktur workspace .historic terlebih dahulu.", Detail: canonicalErr.Error()}
	}
	if !canonicalInfo.IsDir() {
		return &ReadinessError{Code: ReadinessInvalidStructure, Message: "Perbaiki struktur workspace .historic terlebih dahulu.", Detail: ".historic exists but is not a directory"}
	}
	if legacyErr != nil && !errors.Is(legacyErr, os.ErrNotExist) {
		return &ReadinessError{Code: ReadinessInvalidStructure, Message: "Perbaiki struktur workspace terlebih dahulu.", Detail: legacyErr.Error()}
	}
	return nil
}

func validateWorkspaceRoot(workspace Workspace) error {
	info, err := os.Stat(workspace.Root)
	if err != nil {
		return &ReadinessError{Code: ReadinessInvalidStructure, Message: "Perbaiki root workspace terlebih dahulu.", Detail: err.Error()}
	}
	if !info.IsDir() {
		return &ReadinessError{Code: ReadinessInvalidStructure, Message: "Perbaiki root workspace terlebih dahulu.", Detail: "workspace root is not a directory"}
	}
	return nil
}

func validateMinimumStructure(workspace Workspace) error {
	for _, path := range []string{workspace.Database, filepath.Join(workspace.Database, ".git")} {
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("required directory %s: %w", path, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("required path %s is not a directory", path)
		}
	}
	return nil
}

func validateReadableIndex(path string) error {
	// mode=ro prevents validation from creating or modifying SQLite sidecars.
	database, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?mode=ro")
	if err != nil {
		return fmt.Errorf("open index read-only: %w", err)
	}
	defer database.Close()
	if err := database.Ping(); err != nil {
		return fmt.Errorf("ping index: %w", err)
	}
	var integrity string
	if err := database.QueryRow("PRAGMA integrity_check").Scan(&integrity); err != nil {
		return fmt.Errorf("check index integrity: %w", err)
	}
	if strings.ToLower(strings.TrimSpace(integrity)) != "ok" {
		return fmt.Errorf("index integrity check returned %q", integrity)
	}

	metadata, err := readReadinessMetadata(database)
	if err != nil {
		return err
	}
	decision := CheckCompatibility(CurrentAppVersion(), metadata)
	switch decision.Action {
	case ActionUseExisting, ActionUpdateAppMeta:
		return nil
	default:
		return fmt.Errorf("%s", decision.Reason)
	}
}

func readReadinessMetadata(database *sql.DB) (IndexMetadata, error) {
	rows, err := database.Query("SELECT key, value FROM historic_meta")
	if err != nil {
		return IndexMetadata{}, fmt.Errorf("read index metadata: %w", err)
	}
	defer rows.Close()
	values := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return IndexMetadata{}, fmt.Errorf("scan index metadata: %w", err)
		}
		values[key] = value
	}
	if err := rows.Err(); err != nil {
		return IndexMetadata{}, fmt.Errorf("read index metadata rows: %w", err)
	}
	code, err := strconv.Atoi(values["app_version_code"])
	if err != nil || code <= 0 {
		return IndexMetadata{}, fmt.Errorf("invalid app_version_code in index metadata")
	}
	workspace, err := strconv.Atoi(values["workspace_format_version"])
	if err != nil || workspace <= 0 {
		return IndexMetadata{}, fmt.Errorf("invalid workspace_format_version in index metadata")
	}
	indexSchema, err := strconv.Atoi(values["index_schema_version"])
	if err != nil || indexSchema <= 0 {
		return IndexMetadata{}, fmt.Errorf("invalid index_schema_version in index metadata")
	}
	if strings.TrimSpace(values["app_version_name"]) == "" {
		return IndexMetadata{}, fmt.Errorf("invalid app_version_name in index metadata")
	}
	var exists int
	if err := database.QueryRow("SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = 'index_records'").Scan(&exists); err != nil {
		return IndexMetadata{}, fmt.Errorf("index_records table is missing: %w", err)
	}
	return IndexMetadata{AppVersionName: values["app_version_name"], AppVersionCode: code, WorkspaceFormat: workspace, IndexSchema: indexSchema, BuiltAt: values["built_at"], BinaryCommit: values["binary_commit"]}, nil
}

func reject(code ReadinessCode, message, detail string, workspace Workspace) (WorkspaceReadiness, error) {
	return WorkspaceReadiness{State: code, Workspace: workspace}, &ReadinessError{Code: code, Message: message, Detail: detail}
}
