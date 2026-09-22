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
	"historic/internal/identifier"
	"historic/internal/markdown"
	"historic/internal/schema"

	_ "modernc.org/sqlite"
)

var topicFolderPattern = regexp.MustCompile(`^([0-9]{5})-(.+)$`)

// Record is the SQLite-backed representation of one Markdown file.
type Record struct {
	Num         int
	NumPadded   string
	Type        string
	Title       string
	Description string
	Status      domain.Status
	Tags        []string
	Related     []domain.ID
	CreatedAt   string
	UpdatedAt   string
	Path        string
	Storage     domain.StorageState
	FolderID    domain.ID
	FolderSlug  string
	Subfolder   string
	Filename    string
	FileOrder   int
	Content     string
	WordCount   int
	Mtime       string
	Hash        string
}

type fileReadModel struct {
	FileID      domain.FileID
	TopicID     domain.ID
	Type        string
	Path        string
	Filename    string
	Title       string
	Description string
	Status      domain.Status
	Tags        []string
	Related     []domain.ID
	Content     string
	AssetKind   string
	CreatedAt   string
	UpdatedAt   string
	Mtime       string
	Hash        string
	Size        int64
	WordCount   int
}

type topicReadModel struct {
	ID             domain.ID
	Title          string
	Description    string
	Slug           string
	Path           string
	Storage        domain.StorageState
	CreatedAt      string
	UpdatedAt      string
	Tags           []string
	Related        []domain.ID
	ComputedStatus domain.Status
	Content        string
}

type scanResult struct {
	records []Record
	topics  []topicReadModel
	files   []fileReadModel
}

const FTS5TableName = "historic_fts"

func Rebuild(workspace config.Workspace) (int, error) {
	return atomicRebuild(workspace)
}

// Rebuild scans Markdown source files and replaces the index in one transaction.
func rebuildInto(workspace config.Workspace) (int, error) {
	scanned, err := scan(workspace)
	if err != nil {
		return 0, err
	}
	records := scanned.records
	database, err := sql.Open("sqlite", workspace.Index)
	if err != nil {
		return 0, fmt.Errorf("open index: %w", err)
	}
	defer database.Close()
	if err := database.Ping(); err != nil {
		return 0, fmt.Errorf("ping index: %w", err)
	}
	if _, err := database.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return 0, fmt.Errorf("enable index foreign keys: %w", err)
	}
	transaction, err := database.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin index rebuild: %w", err)
	}
	defer transaction.Rollback()
	if _, err := transaction.Exec(`CREATE TABLE IF NOT EXISTS index_records (
		id INTEGER PRIMARY KEY, num INTEGER NOT NULL, num_padded TEXT NOT NULL, type TEXT NOT NULL DEFAULT '',
		title TEXT NOT NULL, description TEXT, status TEXT NOT NULL, storage TEXT NOT NULL DEFAULT 'open', tags TEXT NOT NULL DEFAULT '[]', related TEXT NOT NULL DEFAULT '[]',
		created_at TEXT NOT NULL, updated_at TEXT, path TEXT NOT NULL UNIQUE, folder_id TEXT NOT NULL,
		folder_slug TEXT NOT NULL DEFAULT '', subfolder TEXT NOT NULL DEFAULT '', filename TEXT NOT NULL,
		file_order INTEGER NOT NULL DEFAULT 0, content TEXT NOT NULL DEFAULT '', word_count INTEGER NOT NULL DEFAULT 0,
		mtime TEXT NOT NULL, hash TEXT NOT NULL DEFAULT '')`); err != nil {
		return 0, fmt.Errorf("create index table: %w", err)
	}
	if err := schema.EnsureReadModel(transaction); err != nil {
		return 0, err
	}
	if !hasDescriptionColumn(transaction) {
		if _, err := transaction.Exec(`ALTER TABLE index_records ADD COLUMN description TEXT`); err != nil {
			return 0, fmt.Errorf("add description index column: %w", err)
		}
	}
	if err := replaceReadModel(transaction, scanned.topics, scanned.files); err != nil {
		return 0, err
	}
	if _, err := transaction.Exec(`CREATE INDEX IF NOT EXISTS idx_index_records_num ON index_records(num);
		CREATE INDEX IF NOT EXISTS idx_index_records_status ON index_records(status);
		CREATE INDEX IF NOT EXISTS idx_index_records_type ON index_records(type);
		CREATE INDEX IF NOT EXISTS idx_index_records_storage ON index_records(storage);
		CREATE INDEX IF NOT EXISTS idx_index_records_folder ON index_records(folder_id)`); err != nil {
		return 0, fmt.Errorf("create index indexes: %w", err)
	}
	if _, err := transaction.Exec("DROP TABLE IF EXISTS historic_fts"); err != nil {
		return 0, fmt.Errorf("replace FTS5 table: %w", err)
	}
	if _, err := transaction.Exec(`CREATE VIRTUAL TABLE historic_fts USING fts5(
		entity_type UNINDEXED, entity_id UNINDEXED,
		path, filename, title, description, tags, content, tokenize = 'unicode61'
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
		(num, num_padded, type, title, description, status, storage, tags, related, created_at, updated_at, path, folder_id, folder_slug,
		subfolder, filename, file_order, content, word_count, mtime, hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, fmt.Errorf("prepare index insert: %w", err)
	}
	defer statement.Close()
	indexed := 0
	for _, record := range records {
		if _, exists := seenPaths[record.Path]; exists {
			continue
		}
		seenPaths[record.Path] = struct{}{}
		tags, _ := json.Marshal(record.Tags)
		related, _ := json.Marshal(record.Related)
		if _, err := statement.Exec(record.Num, record.NumPadded, record.Type, record.Title, nullable(record.Description), record.Status.String(), record.Storage.String(), string(tags), string(related), record.CreatedAt, nullable(record.UpdatedAt), record.Path, record.FolderID.String(), record.FolderSlug, record.Subfolder, record.Filename, record.FileOrder, record.Content, record.WordCount, record.Mtime, record.Hash); err != nil {
			return 0, fmt.Errorf("insert %s: %w", record.Path, err)
		}
		indexed++
	}
	if _, err := transaction.Exec(`INSERT INTO historic_fts(entity_type, entity_id, path, filename, title, description, tags, content)
		SELECT 'topic', t.id, t.path, '_meta.yaml', t.title, COALESCE(t.description, ''), t.tags, ''
		FROM topics AS t
		UNION ALL
		SELECT 'file', CAST(id AS TEXT), path, filename, title, COALESCE(description, ''), tags, content FROM files`); err != nil {
		return 0, fmt.Errorf("populate FTS5 index: %w", err)
	}
	if err := ensureSchemaMetadata(transaction); err != nil {
		return 0, err
	}
	if err := transaction.Commit(); err != nil {
		return 0, fmt.Errorf("commit index rebuild: %w", err)
	}
	return indexed, nil
}

func replaceReadModel(transaction *sql.Tx, topics []topicReadModel, files []fileReadModel) error {
	if _, err := transaction.Exec("DELETE FROM files"); err != nil {
		return fmt.Errorf("clear files read model: %w", err)
	}
	if _, err := transaction.Exec("DELETE FROM topics"); err != nil {
		return fmt.Errorf("clear topics read model: %w", err)
	}
	topicStatement, err := transaction.Prepare(`INSERT INTO topics
		(id, num_padded, title, description, slug, path, storage, created_at, updated_at, tags, related, computed_status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare topic read model insert: %w", err)
	}
	defer topicStatement.Close()
	for _, topic := range topics {
		tags, _ := json.Marshal(topic.Tags)
		related, _ := json.Marshal(topic.Related)
		if _, err := topicStatement.Exec(topic.ID.String(), topic.ID.String(), topic.Title, nullable(topic.Description), topic.Slug, topic.Path, topic.Storage.String(), topic.CreatedAt, nullable(topic.UpdatedAt), string(tags), string(related), nullable(topic.ComputedStatus.String())); err != nil {
			return fmt.Errorf("insert topic %s: %w", topic.ID, err)
		}
	}
	fileStatement, err := transaction.Prepare(`INSERT INTO files
		(id, topic_id, type, path, filename, title, description, status, tags, related, content, asset_kind, created_at, updated_at, mtime, hash, size, word_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare file read model insert: %w", err)
	}
	defer fileStatement.Close()
	for _, file := range files {
		tags, _ := json.Marshal(file.Tags)
		related, _ := json.Marshal(file.Related)
		var status any
		if file.Type == "historic_file" {
			status = file.Status.String()
		}
		fileID := file.FileID.String()
		if fileID == "" {
			fileID = "asset:" + file.TopicID.String() + ":" + file.Path
		}
		if _, err := fileStatement.Exec(fileID, file.TopicID.String(), file.Type, file.Path, file.Filename, file.Title, nullable(file.Description), status, string(tags), string(related), file.Content, nullable(file.AssetKind), nullable(file.CreatedAt), nullable(file.UpdatedAt), file.Mtime, file.Hash, file.Size, file.WordCount); err != nil {
			return fmt.Errorf("insert file %s/%s: %w", file.TopicID, file.Path, err)
		}
	}
	return nil
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

func hasDescriptionColumn(transaction *sql.Tx) bool {
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
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err == nil && name == "description" {
			return true
		}
	}
	return false
}

func scan(workspace config.Workspace) (scanResult, error) {
	candidates := make(map[domain.ID][]topicCandidate)
	for _, root := range []struct {
		path    string
		storage domain.StorageState
	}{
		{workspace.Histories, domain.StorageOpen},
		{workspace.Database, domain.StorageClosed},
	} {
		entries, err := os.ReadDir(root.path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return scanResult{}, fmt.Errorf("scan %s: %w", workspace.RelativePath(root.path), err)
		}
		seenRoot := make(map[domain.ID]string)
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".staging-") || entry.Name() == config.IndexFileName {
				continue
			}
			match := topicFolderPattern.FindStringSubmatch(entry.Name())
			if len(match) != 3 {
				continue
			}
			id, err := domain.ParseID(match[1])
			if err != nil {
				continue
			}
			candidatePath := filepath.Join(root.path, entry.Name())
			info, err := os.Lstat(candidatePath)
			if err != nil {
				return scanResult{}, fmt.Errorf("inspect topic %s: %w", workspace.RelativePath(candidatePath), err)
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return scanResult{}, fmt.Errorf("%w: topic symlink %s", domain.ErrConflict, workspace.RelativePath(candidatePath))
			}
			if !info.IsDir() {
				continue
			}
			if previous, exists := seenRoot[id]; exists {
				return scanResult{}, fmt.Errorf("%w: duplicate topic ID %s in %s (%s and %s)", domain.ErrConflict, id, root.storage, workspace.RelativePath(previous), workspace.RelativePath(candidatePath))
			}
			seenRoot[id] = candidatePath
			candidates[id] = append(candidates[id], topicCandidate{path: candidatePath, storage: root.storage, slug: match[2]})
		}
	}

	ids := make([]domain.ID, 0, len(candidates))
	for id := range candidates {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	result := scanResult{records: make([]Record, 0), topics: make([]topicReadModel, 0, len(ids)), files: make([]fileReadModel, 0)}
	for _, id := range ids {
		locations := candidates[id]
		chosen := locations[0]
		for _, location := range locations[1:] {
			if location.storage == domain.StorageOpen {
				chosen = location
			}
		}
		topic, records, files, err := scanTopic(workspace, id, chosen)
		if err != nil {
			return scanResult{}, err
		}
		result.topics = append(result.topics, topic)
		result.records = append(result.records, records...)
		result.files = append(result.files, files...)
	}
	sort.Slice(result.records, func(i, j int) bool { return result.records[i].Path < result.records[j].Path })
	for index := range result.records {
		result.records[index].FileOrder = index
	}
	return result, nil
}

type topicCandidate struct {
	path    string
	storage domain.StorageState
	slug    string
}

func scanTopic(workspace config.Workspace, id domain.ID, candidate topicCandidate) (topicReadModel, []Record, []fileReadModel, error) {
	metaPath := filepath.Join(candidate.path, markdown.MetaFilename)
	meta, err := markdown.ReadTopicMetadata(candidate.path)
	if err != nil {
		return topicReadModel{}, nil, nil, err
	}
	if meta.ID != id {
		return topicReadModel{}, nil, nil, fmt.Errorf("%w: metadata ID %s does not match folder ID %s at %s", domain.ErrConflict, meta.ID, id, workspace.RelativePath(candidate.path))
	}
	manifestByPath := make(map[string]domain.FileID, len(meta.Files))
	for _, file := range meta.Files {
		manifestByPath[file.Path] = file.ID
	}
	topic := topicReadModel{
		ID: id, Title: meta.Title, Description: meta.Description, Slug: candidate.slug,
		Path: workspace.RelativePath(candidate.path), Storage: candidate.storage,
		CreatedAt: meta.Created, UpdatedAt: meta.Updated, Tags: meta.Tags, Related: meta.Related,
	}
	info, statErr := os.Stat(metaPath)
	if statErr != nil {
		return topicReadModel{}, nil, nil, fmt.Errorf("inspect topic metadata %s: %w", workspace.RelativePath(metaPath), statErr)
	}
	records := []Record{{
		Num: id.Number(), NumPadded: id.String(), Type: "meta", Title: meta.Title, Description: meta.Description,
		Status: "", Tags: meta.Tags, Related: meta.Related, CreatedAt: meta.Created,
		UpdatedAt: meta.Updated, Storage: candidate.storage,
		Path:     filepath.ToSlash(filepath.Join(workspace.RelativePath(candidate.path), markdown.MetaFilename)),
		FolderID: id, FolderSlug: candidate.slug, Filename: markdown.MetaFilename,
		Mtime: info.ModTime().UTC().Format(time.RFC3339Nano), Hash: hashFile(metaPath),
	}}
	files := make([]fileReadModel, 0)
	statuses := make([]domain.Status, 0)
	err = filepath.WalkDir(candidate.path, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: symlink %s", domain.ErrConflict, workspace.RelativePath(path))
		}
		if entry.IsDir() || path == metaPath {
			return nil
		}
		relativeOS, err := filepath.Rel(candidate.path, path)
		if err != nil {
			return err
		}
		relative := filepath.ToSlash(relativeOS)
		stat, err := entry.Info()
		if err != nil {
			return err
		}
		if strings.EqualFold(filepath.Ext(path), ".md") && filepath.Base(path) != "_meta.md" {
			document, parseErr := markdown.ParseFile(path)
			if parseErr == nil {
				fileID := manifestByPath[relative]
				if fileID == "" {
					fileID, err = identifier.Default.New()
					if err != nil {
						return err
					}
				}
				record, recordErr := scanFile(workspace, path, id)
				if recordErr != nil {
					return recordErr
				}
				record.Storage = candidate.storage
				record.Path = filepath.ToSlash(filepath.Join(workspace.RelativePath(candidate.path), relative))
				record.FolderID = id
				record.FolderSlug = candidate.slug
				record.Subfolder = filepath.ToSlash(filepath.Dir(relative))
				record.Filename = filepath.Base(path)
				records = append(records, record)
				statuses = append(statuses, document.Frontmatter.Status)
				files = append(files, fileReadModelFromDocument(id, relative, path, stat, document, "historic_file", fileID))
				return nil
			}
		}
		files = append(files, fileReadModelFromAsset(id, relative, path, stat))
		return nil
	})
	if err != nil {
		return topicReadModel{}, nil, nil, fmt.Errorf("scan topic %s: %w", workspace.RelativePath(candidate.path), err)
	}
	topic.ComputedStatus = aggregateStatus(statuses)
	return topic, records, files, nil
}

func fileReadModelFromDocument(id domain.ID, relative, path string, info os.FileInfo, document markdown.Document, fileType string, fileID domain.FileID) fileReadModel {
	return fileReadModel{FileID: fileID, TopicID: id, Type: fileType, Path: relative, Filename: filepath.Base(path), Title: document.Frontmatter.Title, Description: document.Frontmatter.Description, Status: document.Frontmatter.Status, Tags: document.Frontmatter.Tags, Related: document.Frontmatter.Related, Content: document.Body, CreatedAt: document.Frontmatter.Created, UpdatedAt: document.Frontmatter.Updated, Mtime: info.ModTime().UTC().Format(time.RFC3339Nano), Hash: hashFile(path), Size: info.Size(), WordCount: wordCount(document.Body)}
}

func fileReadModelFromAsset(id domain.ID, relative, path string, info os.FileInfo) fileReadModel {
	return fileReadModel{TopicID: id, Type: "asset", Path: relative, Filename: filepath.Base(path), Title: filepath.Base(path), AssetKind: assetKind(path), Mtime: info.ModTime().UTC().Format(time.RFC3339Nano), Hash: hashFile(path), Size: info.Size()}
}

func assetKind(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".md" || ext == ".markdown" {
		return "markdown"
	}
	if ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" || ext == ".webp" {
		return "image"
	}
	if ext == ".pdf" {
		return "pdf"
	}
	return strings.TrimPrefix(ext, ".")
}

func aggregateStatus(statuses []domain.Status) domain.Status {
	if len(statuses) == 0 {
		return ""
	}
	for _, status := range statuses {
		if status != domain.StatusComplete && status != domain.StatusCancelled {
			return status
		}
	}
	if allStatus(statuses, domain.StatusCancelled) {
		return domain.StatusCancelled
	}
	return domain.StatusComplete
}

func allStatus(statuses []domain.Status, expected domain.Status) bool {
	for _, status := range statuses {
		if status != expected {
			return false
		}
	}
	return true
}

func scanFile(workspace config.Workspace, path string, topicID domain.ID) (Record, error) {
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
	folderID, folderSlug, subfolder := topicID, "", ""
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
		Description: document.Frontmatter.Description, Status: document.Frontmatter.Status, Tags: document.Frontmatter.Tags, Related: document.Frontmatter.Related,
		CreatedAt: createdAt, UpdatedAt: document.Frontmatter.Updated, Path: relative, FolderID: folderID,
		FolderSlug: folderSlug, Subfolder: subfolder, Filename: filepath.Base(path), Content: document.Body,
		WordCount: wordCount(document.Body), Mtime: info.ModTime().UTC().Format(time.RFC3339Nano), Hash: hashFile(path),
	}, nil
}

func inferType(path string) string {
	base := strings.ToLower(filepath.Base(path))
	if base == markdown.MetaFilename {
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
