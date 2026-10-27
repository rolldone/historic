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

func ensureIndexDescriptionColumn(database *sql.DB) error {
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
		if name == "description" {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate index_records columns: %w", err)
	}
	if !tableExists || found {
		return nil
	}
	if _, err := database.Exec("ALTER TABLE index_records ADD COLUMN description TEXT"); err != nil {
		return fmt.Errorf("add description column: %w", err)
	}
	return nil
}

// ensureTopicsExtendsColumn backfills the nullable extends_id column on read
// model tables created before topic extension existed. Rebuild remains the
// source of truth; this keeps an in-place index readable until then.
func ensureTopicsExtendsColumn(database *sql.DB) error {
	rows, err := database.Query("PRAGMA table_info(topics)")
	if err != nil {
		return fmt.Errorf("inspect topics columns: %w", err)
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
			return fmt.Errorf("read topics columns: %w", err)
		}
		tableExists = true
		if name == "extends_id" {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate topics columns: %w", err)
	}
	if !tableExists || found {
		return nil
	}
	if _, err := database.Exec("ALTER TABLE topics ADD COLUMN extends_id TEXT"); err != nil {
		return fmt.Errorf("add extends_id column: %w", err)
	}
	return nil
}

// ensureFilesForwardedToColumn backfills the nullable forwarded_to column on
// read model tables created before the forwarded status existed.
func ensureFilesForwardedToColumn(database *sql.DB) error {
	rows, err := database.Query("PRAGMA table_info(files)")
	if err != nil {
		return fmt.Errorf("inspect files columns: %w", err)
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
			return fmt.Errorf("read files columns: %w", err)
		}
		tableExists = true
		if name == "forwarded_to" {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate files columns: %w", err)
	}
	if !tableExists || found {
		return nil
	}
	if _, err := database.Exec("ALTER TABLE files ADD COLUMN forwarded_to TEXT"); err != nil {
		return fmt.Errorf("add forwarded_to column: %w", err)
	}
	return nil
}
