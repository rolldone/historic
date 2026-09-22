package indexer

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"historic/internal/config"
)

const SchemaVersion = 4

func schemaVersion(database *sql.DB) (int, error) {
	var version int
	err := database.QueryRow("SELECT value FROM historic_meta WHERE key = 'index_schema_version'").Scan(&version)
	if err != nil {
		return 0, err
	}
	return version, nil
}

func readIndexMetadata(database *sql.DB) (config.IndexMetadata, error) {
	rows, err := database.Query("SELECT key, value FROM historic_meta")
	if err != nil {
		return config.IndexMetadata{}, fmt.Errorf("read historic_meta: %w", err)
	}
	defer rows.Close()
	result := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return config.IndexMetadata{}, err
		}
		result[key] = value
	}
	if err := rows.Err(); err != nil {
		return config.IndexMetadata{}, err
	}
	code, err := strconv.Atoi(result["app_version_code"])
	if err != nil {
		code = 0
	}
	workspace, err := strconv.Atoi(result["workspace_format_version"])
	if err != nil {
		workspace = 0
	}
	indexSchema, err := strconv.Atoi(result["index_schema_version"])
	if err != nil {
		indexSchema = 0
	}
	return config.IndexMetadata{AppVersionName: result["app_version_name"], AppVersionCode: code, WorkspaceFormat: workspace, IndexSchema: indexSchema, BuiltAt: result["built_at"], BinaryCommit: result["binary_commit"]}, nil
}

func ensureSchemaMetadata(database *sql.Tx) error {
	const schema = `CREATE TABLE IF NOT EXISTS historic_meta (key TEXT PRIMARY KEY, value TEXT NOT NULL)`
	if _, err := database.Exec(schema); err != nil {
		return fmt.Errorf("create historic_meta: %w", err)
	}
	current := config.CurrentAppVersion()
	fields := map[string]string{
		"app_version_name":         current.Name,
		"app_version_code":         strconv.Itoa(current.Code),
		"workspace_format_version": strconv.Itoa(current.Workspace),
		"index_schema_version":     strconv.Itoa(current.IndexSchema),
		"built_at":                 time.Now().UTC().Format(time.RFC3339),
		"binary_commit":            "",
	}
	for key, value := range fields {
		if _, err := database.Exec("INSERT INTO historic_meta(key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value", key, value); err != nil {
			return fmt.Errorf("write historic_meta %s: %w", key, err)
		}
	}
	return nil
}
