package schema

import (
	"database/sql"
	"fmt"
)

// Execer is implemented by both *sql.DB and *sql.Tx.
type Execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

// EnsureReadModel creates the schema used by the SQLite read model. The
// schema is additive to the legacy index_records/index FTS schema.
func EnsureReadModel(database Execer) error {
	const schema = `
CREATE TABLE IF NOT EXISTS topics (
    id TEXT PRIMARY KEY,
    num_padded TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT,
    slug TEXT NOT NULL,
    path TEXT NOT NULL,
    storage TEXT NOT NULL CHECK (storage IN ('open', 'closed')),
    created_at TEXT NOT NULL,
    updated_at TEXT,
    tags TEXT NOT NULL DEFAULT '[]',
    related TEXT NOT NULL DEFAULT '[]',
    computed_status TEXT
);
CREATE TABLE IF NOT EXISTS files (
    id INTEGER PRIMARY KEY,
    topic_id TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('historic_file', 'asset')),
    path TEXT NOT NULL CHECK (
        path <> '' AND path <> '..' AND substr(path, 1, 1) <> '/' AND
        path NOT LIKE '../%' AND path NOT LIKE '%/../%' AND path NOT LIKE '%/..' AND
        path NOT LIKE '.historic/%' AND path NOT LIKE '.database/%' AND
        path NOT LIKE 'archive_path/%' AND path NOT LIKE '%/archive_path/%'
    ),
    filename TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT,
    tags TEXT NOT NULL DEFAULT '[]',
    related TEXT NOT NULL DEFAULT '[]',
    content TEXT NOT NULL DEFAULT '',
    asset_kind TEXT,
    created_at TEXT,
    updated_at TEXT,
    mtime TEXT NOT NULL,
    hash TEXT NOT NULL,
    size INTEGER NOT NULL DEFAULT 0,
    word_count INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE,
    UNIQUE (topic_id, path),
    CHECK (type = 'historic_file' OR status IS NULL)
);
CREATE INDEX IF NOT EXISTS idx_topics_num_padded ON topics(num_padded);
CREATE INDEX IF NOT EXISTS idx_topics_storage ON topics(storage);
CREATE INDEX IF NOT EXISTS idx_topics_slug ON topics(slug);
CREATE INDEX IF NOT EXISTS idx_files_topic_id ON files(topic_id);
CREATE INDEX IF NOT EXISTS idx_files_type ON files(type);
CREATE INDEX IF NOT EXISTS idx_files_status ON files(status);
CREATE INDEX IF NOT EXISTS idx_files_path ON files(path);
CREATE INDEX IF NOT EXISTS idx_files_hash ON files(hash);
`
	if _, err := database.Exec(schema); err != nil {
		return fmt.Errorf("create topics/files schema: %w", err)
	}
	return nil
}
