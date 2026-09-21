package indexer

import (
	"database/sql"
	"fmt"
)

const SchemaVersion = 2

func schemaVersion(database *sql.DB) (int, error) {
	var version int
	err := database.QueryRow("SELECT value FROM historic_metadata WHERE key = 'schema_version'").Scan(&version)
	if err != nil {
		return 0, err
	}
	return version, nil
}

func ensureSchemaMetadata(database *sql.Tx) error {
	if _, err := database.Exec(`CREATE TABLE IF NOT EXISTS historic_metadata (key TEXT PRIMARY KEY, value INTEGER NOT NULL)`); err != nil {
		return fmt.Errorf("create index metadata: %w", err)
	}
	if _, err := database.Exec(`INSERT INTO historic_metadata(key, value) VALUES ('schema_version', ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, SchemaVersion); err != nil {
		return fmt.Errorf("write index schema version: %w", err)
	}
	return nil
}
