package indexer

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"

	_ "modernc.org/sqlite"
)

var topicFolderPattern = regexp.MustCompile(`^([0-9]{5})-(.+)$`)

// Record is the SQLite-backed representation of one Markdown file.
type Record struct {
	Num        int
	NumPadded  string
	Type       string
	Title      string
	Status     domain.Status
	Tags       []string
	Related    []domain.ID
	CreatedAt  string
	UpdatedAt  string
	Path       string
	Storage    domain.StorageState
	FolderID   domain.ID
	FolderSlug string
	Subfolder  string
	Filename   string
	FileOrder  int
	Content    string
	WordCount  int
	Mtime      string
	Hash       string
}

const FTS5TableName = "historic_fts"

// Rebuild scans Markdown source files and replaces the index in one transaction.
func Rebuild(workspace config.Workspace) (int, error) {
	records, err := scan(workspace)
	if err != nil {
		return 0, err
	}
	database, err := sql.Open("sqlite", workspace.Index)
	if err != nil {
		return 0, fmt.Errorf("open index: %w", err)
	}
	defer database.Close()
	if err := database.Ping(); err != nil {
		return 0, fmt.Errorf("ping index: %w", err)
	}
	transaction, err := database.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin index rebuild: %w", err)
	}
	defer transaction.Rollback()
	if _, err := transaction.Exec(`CREATE TABLE IF NOT EXISTS index_records (
		id INTEGER PRIMARY KEY, num INTEGER NOT NULL, num_padded TEXT NOT NULL, type TEXT NOT NULL DEFAULT '',
		title TEXT NOT NULL, status TEXT NOT NULL, storage TEXT NOT NULL DEFAULT 'open', tags TEXT NOT NULL DEFAULT '[]', related TEXT NOT NULL DEFAULT '[]',
		created_at TEXT NOT NULL, updated_at TEXT, path TEXT NOT NULL UNIQUE, folder_id TEXT NOT NULL,
		folder_slug TEXT NOT NULL DEFAULT '', subfolder TEXT NOT NULL DEFAULT '', filename TEXT NOT NULL,
		file_order INTEGER NOT NULL DEFAULT 0, content TEXT NOT NULL DEFAULT '', word_count INTEGER NOT NULL DEFAULT 0,
		mtime TEXT NOT NULL, hash TEXT NOT NULL DEFAULT '')`); err != nil {
		return 0, fmt.Errorf("create index table: %w", err)
	}
	if !hasStorageColumn(transaction) {
		if _, err := transaction.Exec(`ALTER TABLE index_records ADD COLUMN storage TEXT NOT NULL DEFAULT 'open'`); err != nil {
			return 0, fmt.Errorf("add storage index column: %w", err)
		}
	}
	if _, err := transaction.Exec(`CREATE INDEX IF NOT EXISTS idx_index_records_num ON index_records(num);
		CREATE INDEX IF NOT EXISTS idx_index_records_status ON index_records(status);
		CREATE INDEX IF NOT EXISTS idx_index_records_type ON index_records(type);
		CREATE INDEX IF NOT EXISTS idx_index_records_storage ON index_records(storage);
		CREATE INDEX IF NOT EXISTS idx_index_records_folder ON index_records(folder_id)`); err != nil {
		return 0, fmt.Errorf("create index indexes: %w", err)
	}
	if _, err := transaction.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS historic_fts USING fts5(
		path, filename, title, content, tokenize = 'unicode61'
	)`); err != nil {
		return 0, fmt.Errorf("create FTS5 table: %w", err)
	}
	if _, err := transaction.Exec("DELETE FROM index_records"); err != nil {
		return 0, fmt.Errorf("clear index: %w", err)
	}
	if _, err := transaction.Exec("DELETE FROM historic_fts"); err != nil {
		return 0, fmt.Errorf("clear FTS5 index: %w", err)
	}
	seenPaths := make(map[string]struct{}, len(records))
	statement, err := transaction.Prepare(`INSERT INTO index_records
		(num, num_padded, type, title, status, storage, tags, related, created_at, updated_at, path, folder_id, folder_slug,
		subfolder, filename, file_order, content, word_count, mtime, hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, fmt.Errorf("prepare index insert: %w", err)
	}
	defer statement.Close()
	ftsStatement, err := transaction.Prepare(`INSERT INTO historic_fts(rowid, path, filename, title, content) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, fmt.Errorf("prepare FTS5 insert: %w", err)
	}
	defer ftsStatement.Close()
	indexed := 0
	for _, record := range records {
		if _, exists := seenPaths[record.Path]; exists {
			continue
		}
		seenPaths[record.Path] = struct{}{}
		tags, _ := json.Marshal(record.Tags)
		related, _ := json.Marshal(record.Related)
		result, err := statement.Exec(record.Num, record.NumPadded, record.Type, record.Title, record.Status.String(), record.Storage.String(), string(tags), string(related), record.CreatedAt, nullable(record.UpdatedAt), record.Path, record.FolderID.String(), record.FolderSlug, record.Subfolder, record.Filename, record.FileOrder, record.Content, record.WordCount, record.Mtime, record.Hash)
		if err != nil {
			return 0, fmt.Errorf("insert %s: %w", record.Path, err)
		}
		rowID, err := result.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("read index row ID for %s: %w", record.Path, err)
		}
		if _, err := ftsStatement.Exec(rowID, record.Path, record.Filename, record.Title, record.Content); err != nil {
			return 0, fmt.Errorf("insert FTS5 record %s: %w", record.Path, err)
		}
		indexed++
	}
	if err := transaction.Commit(); err != nil {
		return 0, fmt.Errorf("commit index rebuild: %w", err)
	}
	return indexed, nil
}

func hasStorageColumn(transaction *sql.Tx) bool {
	rows, err := transaction.Query("PRAGMA table_info(index_records)")
	if err != nil {
		return false
	}
	defer rows.Close()
	var cid int
	var name, columnType string
	var notNull, primaryKey int
	var defaultValue any
	for rows.Next() {
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err == nil && name == "storage" {
			return true
		}
	}
	return false
}

func scan(workspace config.Workspace) ([]Record, error) {
	var records []Record
	for _, root := range []string{workspace.Histories, workspace.Database} {
		storage := domain.StorageOpen
		if filepath.Clean(root) == filepath.Clean(workspace.Database) {
			storage = domain.StorageClosed
		}
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if filepath.Clean(root) == filepath.Clean(workspace.Histories) && filepath.Clean(path) == filepath.Clean(workspace.Database) && entry.IsDir() {
				return filepath.SkipDir
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return nil
			}
			if entry.IsDir() || filepath.Ext(path) != ".md" {
				return nil
			}
			record, err := scanFile(workspace, path)
			record.Storage = storage
			if err != nil {
				if filepath.Base(path) == "_meta.md" {
					// A broken topic metadata file is not indexed, allowing batch operations to continue.
					return nil
				}
				if filepath.Ext(path) == ".md" {
					// Markdown without valid Historic frontmatter is an asset, not an index error.
					return nil
				}
				return err
			}
			records = append(records, record)
			return nil
		})
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("scan %s: %w", workspace.RelativePath(root), err)
		}
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Path < records[j].Path })
	for index := range records {
		records[index].FileOrder = index
	}
	return records, nil
}

func scanFile(workspace config.Workspace, path string) (Record, error) {
	document, err := markdown.ParseFile(path)
	if err != nil {
		return Record{}, fmt.Errorf("invalid Markdown %s: %w", workspace.RelativePath(path), err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return Record{}, fmt.Errorf("stat %s: %w", workspace.RelativePath(path), err)
	}
	relative := workspace.RelativePath(path)
	parts := strings.Split(filepath.ToSlash(relative), "/")
	folderID, folderSlug, subfolder := document.Frontmatter.ID, "", ""
	for index, part := range parts {
		if match := topicFolderPattern.FindStringSubmatch(part); len(match) == 3 {
			folderSlug = match[2]
			if index+1 < len(parts)-1 {
				subfolder = strings.Join(parts[index+1:len(parts)-1], "/")
			}
			break
		}
	}
	createdAt := document.Frontmatter.Created
	return Record{
		Num: folderID.Number(), NumPadded: folderID.String(), Type: inferType(relative), Title: document.Frontmatter.Title,
		Status: document.Frontmatter.Status, Tags: document.Frontmatter.Tags, Related: document.Frontmatter.Related,
		CreatedAt: createdAt, UpdatedAt: document.Frontmatter.Updated, Path: relative, FolderID: folderID,
		FolderSlug: folderSlug, Subfolder: subfolder, Filename: filepath.Base(path), Content: document.Body,
		WordCount: wordCount(document.Body), Mtime: info.ModTime().UTC().Format(time.RFC3339Nano), Hash: hashFile(path),
	}, nil
}

func inferType(path string) string {
	base := strings.ToLower(filepath.Base(path))
	if base == "_meta.md" {
		return "meta"
	}
	name := strings.TrimSuffix(base, ".md")
	for _, kind := range []string{"prd", "spec", "issue", "note", "decision", "task"} {
		if name == kind || strings.HasPrefix(name, kind+"-") {
			return kind
		}
	}
	if strings.Contains(filepath.ToSlash(path), "/wos/") {
		return "task"
	}
	return "file"
}

func wordCount(content string) int {
	return len(strings.Fields(content))
}

func hashFile(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

func nullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// Count returns the number of indexed records.
func Count(workspace config.Workspace) (int, error) {
	database, err := sql.Open("sqlite", workspace.Index)
	if err != nil {
		return 0, err
	}
	defer database.Close()
	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM index_records").Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

var _ = sort.Strings
var _ = filepath.Separator
var _ = strconv.Itoa
