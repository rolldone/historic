package config

import (
	"database/sql"
	"fmt"
)

func ensureIndexStorageColumn(database *sql.DB) error {
	rows, err := database.Query("PRAGMA table_info(index_records)")
	if err != nil {
		return fmt.Errorf("inspect index_records columns: %w", err)
	}
	defer rows.Close()
	found := false
	tableExists := false
	var cid int
	var name, columnType string
	var notNull, primaryKey int
	var defaultValue any
	for rows.Next() {
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return fmt.Errorf("read index_records columns: %w", err)
		}
		tableExists = true
		if name == "storage" {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate index_records columns: %w", err)
	}
	if !tableExists || found {
		return nil
	}
	if _, err := database.Exec("ALTER TABLE index_records ADD COLUMN storage TEXT NOT NULL DEFAULT 'open'"); err != nil {
		return fmt.Errorf("add storage column: %w", err)
	}
	return nil
}
