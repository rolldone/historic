package search

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"historic/internal/config"
	"historic/internal/domain"

	_ "modernc.org/sqlite"
)

// Options controls a FTS5 search. Structured filters are applied by SQL and
// are never interpolated into the FTS query.
type Options struct {
	Keyword       string
	Status        domain.Status
	Tags          []string
	Folder        string
	Type          string
	ID            domain.ID
	OpenOnly      bool
	ClosedOnly    bool
	ActiveOnly    bool // compatibility for package callers; CLI rejects --active
	ArchivedOnly  bool // compatibility for package callers; CLI rejects --archived
	CreatedAfter  string
	CreatedBefore string
	UpdatedAfter  string
	UpdatedBefore string
}

// Result is an AI-friendly match from either the topic or file read model.
type Result struct {
	Type        string   `json:"type"`
	TopicID     string   `json:"topic_id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Storage     string   `json:"storage"`
	Tags        []string `json:"tags"`
	Path        string   `json:"path"`
	Score       float64  `json:"score"`
	MatchedIn   []string `json:"matched_in"`
	Snippet     string   `json:"snippet"`

	// Legacy fields retained for TUI and package compatibility.
	ID     string `json:"id,omitempty"`
	Active bool   `json:"active,omitempty"`
}

func RecentTopics(workspace config.Workspace, options Options) ([]Result, error) {
	if err := validateOptions(options, false); err != nil {
		return nil, err
	}
	database, err := openIndex(workspace)
	if err != nil {
		return nil, err
	}
	defer database.Close()
	where, args := topicPredicates(options)
	rows, err := database.Query(`SELECT id, title, COALESCE(description, ''), COALESCE(computed_status, ''), storage, path, tags
		FROM topics WHERE `+strings.Join(where, " AND ")+` ORDER BY path ASC`, args...)
	if err != nil {
		return nil, fmt.Errorf("query recent topics: %w", err)
	}
	defer rows.Close()
	results := make([]Result, 0)
	for rows.Next() {
		var result Result
		var tagsJSON string
		if err := rows.Scan(&result.TopicID, &result.Title, &result.Description, &result.Status, &result.Storage, &result.Path, &tagsJSON); err != nil {
			return nil, fmt.Errorf("read recent topic: %w", err)
		}
		result.Type, result.ID, result.Tags = "topic", result.TopicID, decodeStrings(tagsJSON)
		result.Active = result.Storage == domain.StorageOpen.String()
		results = append(results, result)
	}
	return results, rows.Err()
}

// Find searches topics and files using the shared read model and weighted FTS5.
func Find(workspace config.Workspace, options Options) ([]Result, error) {
	if err := validateOptions(options, true); err != nil {
		return nil, err
	}
	query, err := prepareQuery(strings.TrimSpace(options.Keyword))
	if err != nil {
		return nil, err
	}
	database, err := openIndex(workspace)
	if err != nil {
		return nil, err
	}
	defer database.Close()
	where := []string{"historic_fts MATCH ?"}
	args := []any{query}
	predicates, predicateArgs := resultPredicates(options)
	where = append(where, predicates...)
	args = append(args, predicateArgs...)
	statement := `SELECT historic_fts.entity_type, historic_fts.entity_id, COALESCE(t.id, f.topic_id),
		COALESCE(CASE WHEN historic_fts.entity_type = 'topic' THEN t.title ELSE f.title END, ''),
		COALESCE(CASE WHEN historic_fts.entity_type = 'topic' THEN t.description ELSE f.description END, ''),
		COALESCE(CASE WHEN historic_fts.entity_type = 'topic' THEN t.computed_status ELSE f.status END, ''),
		COALESCE(t.storage, ''), CASE WHEN historic_fts.entity_type = 'topic' THEN t.path ELSE t.path || '/' || f.path END,
		COALESCE(CASE WHEN historic_fts.entity_type = 'topic' THEN t.tags ELSE f.tags END, '[]'),
		bm25(historic_fts, 1.0, 1.0, 10.0, 8.0, 10.0, 1.0) AS rank,
		snippet(historic_fts, -1, '', '', ' … ', 24)
		FROM historic_fts
		LEFT JOIN files AS f ON historic_fts.entity_type = 'file' AND CAST(f.id AS TEXT) = historic_fts.entity_id
		LEFT JOIN topics AS t ON t.id = CASE WHEN historic_fts.entity_type = 'topic' THEN historic_fts.entity_id ELSE f.topic_id END
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY rank ASC, historic_fts.entity_type ASC, historic_fts.entity_id ASC`
	rows, err := database.Query(statement, args...)
	if err != nil {
		return nil, fmt.Errorf("invalid FTS query: %w", err)
	}
	defer rows.Close()
	results := make([]Result, 0)
	for rows.Next() {
		var result Result
		var tagsJSON, snippetText string
		var rank float64
		if err := rows.Scan(&result.Type, &result.ID, &result.TopicID, &result.Title, &result.Description, &result.Status, &result.Storage, &result.Path, &tagsJSON, &rank, &snippetText); err != nil {
			return nil, fmt.Errorf("read search result: %w", err)
		}
		result.Tags = decodeStrings(tagsJSON)
		result.Score = -rank
		result.MatchedIn = matchedFields(result.Type, result.Title, result.Description, result.Tags, result.Path, options.Keyword, snippetText)
		result.Snippet = snippetText
		if result.Type == "file" {
			result.Snippet = snippet(fallbackContent(database, result.ID), options.Keyword)
		}
		if result.Snippet == "" {
			result.Snippet = firstLine(result.Description)
		}
		if result.Type == "topic" {
			result.Path = filepath.ToSlash(filepath.Join(result.Path, "_meta.yaml"))
		} else {
			result.ID = result.TopicID
		}
		result.Active = result.Storage == domain.StorageOpen.String()
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate search results: %w", err)
	}
	return results, nil
}

func openIndex(workspace config.Workspace) (*sql.DB, error) {
	database, err := sql.Open("sqlite", workspace.Index)
	if err != nil {
		return nil, fmt.Errorf("open search index: %w", err)
	}
	if err := database.Ping(); err != nil {
		database.Close()
		return nil, fmt.Errorf("search index unavailable: run historic rebuild: %w", err)
	}
	var table string
	if err := database.QueryRow("SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'historic_fts'").Scan(&table); err != nil {
		database.Close()
		return nil, fmt.Errorf("FTS5 index unavailable: run historic rebuild: %w", err)
	}
	return database, nil
}

func validateOptions(options Options, requireKeyword bool) error {
	if options.OpenOnly && options.ClosedOnly || options.ActiveOnly && options.ArchivedOnly {
		return fmt.Errorf("open and closed filters cannot be combined")
	}
	if options.Type != "" && !validType(options.Type) {
		return fmt.Errorf("invalid type %q", options.Type)
	}
	if options.ID != "" && !options.ID.Valid() {
		return fmt.Errorf("invalid topic ID %q", options.ID)
	}
	if options.Folder != "" {
		folder := filepath.ToSlash(strings.Trim(strings.TrimSpace(options.Folder), "/"))
		if folder == "" || filepath.IsAbs(options.Folder) || folder == "." || strings.HasPrefix(folder, "../") || strings.Contains(folder, "/../") || strings.HasSuffix(folder, "/..") || folder == ".." {
			return fmt.Errorf("invalid folder filter %q", options.Folder)
		}
	}
	if requireKeyword && strings.TrimSpace(options.Keyword) == "" {
		return fmt.Errorf("keyword must not be empty")
	}
	return nil
}

func topicPredicates(options Options) ([]string, []any) {
	where := []string{"1 = 1"}
	args := make([]any, 0)
	if options.Status != "" {
		where = append(where, "COALESCE(computed_status, '') = ?")
		args = append(args, options.Status.String())
	}
	if options.ID != "" {
		where = append(where, "id = ?")
		args = append(args, options.ID.String())
	}
	if options.OpenOnly || options.ActiveOnly {
		where = append(where, "storage = 'open'")
	}
	if options.ClosedOnly || options.ArchivedOnly {
		where = append(where, "storage = 'closed'")
	}
	if options.CreatedAfter != "" {
		where = append(where, "created_at >= ?")
		args = append(args, options.CreatedAfter)
	}
	if options.CreatedBefore != "" {
		where = append(where, "created_at <= ?")
		args = append(args, options.CreatedBefore)
	}
	if options.UpdatedAfter != "" {
		where = append(where, "COALESCE(updated_at, created_at) >= ?")
		args = append(args, options.UpdatedAfter)
	}
	if options.UpdatedBefore != "" {
		where = append(where, "COALESCE(updated_at, created_at) <= ?")
		args = append(args, options.UpdatedBefore)
	}
	return where, args
}

func resultPredicates(options Options) ([]string, []any) {
	where := make([]string, 0)
	args := make([]any, 0)
	if options.OpenOnly || options.ActiveOnly {
		where = append(where, "t.storage = 'open'")
	}
	if options.ClosedOnly || options.ArchivedOnly {
		where = append(where, "t.storage = 'closed'")
	}
	if options.Status != "" {
		where = append(where, "((historic_fts.entity_type = 'topic' AND t.computed_status = ?) OR (historic_fts.entity_type = 'file' AND f.status = ?))")
		args = append(args, options.Status.String(), options.Status.String())
	}
	if options.Type != "" {
		if options.Type == "topic" {
			where = append(where, "historic_fts.entity_type = 'topic'")
		} else {
			where = append(where, "historic_fts.entity_type = 'file' AND (f.type = ? OR (? = 'task' AND f.path LIKE 'wos/%'))")
			args = append(args, options.Type, options.Type)
		}
	}
	if options.ActiveOnly || options.ArchivedOnly {
		where = append(where, "historic_fts.entity_type = 'file'")
	}
	if options.ID != "" {
		where = append(where, "t.id = ?")
		args = append(args, options.ID.String())
	}
	if options.Folder != "" {
		folder := filepath.ToSlash(strings.Trim(strings.TrimSpace(options.Folder), "/"))
		where = append(where, "(historic_fts.entity_type = 'topic' AND (t.path = ? OR t.path LIKE ?) OR historic_fts.entity_type = 'file' AND (f.path = ? OR f.path LIKE ? OR t.path = ? OR t.path LIKE ?))")
		args = append(args, folder, folder+"/%", folder, folder+"/%", folder, folder+"/%")
	}
	if options.CreatedAfter != "" {
		where = append(where, "(historic_fts.entity_type = 'topic' AND t.created_at >= ? OR historic_fts.entity_type = 'file' AND COALESCE(f.created_at, f.mtime) >= ?)")
		args = append(args, options.CreatedAfter, options.CreatedAfter)
	}
	if options.CreatedBefore != "" {
		where = append(where, "(historic_fts.entity_type = 'topic' AND t.created_at <= ? OR historic_fts.entity_type = 'file' AND COALESCE(f.created_at, f.mtime) <= ?)")
		args = append(args, options.CreatedBefore, options.CreatedBefore)
	}
	if options.UpdatedAfter != "" {
		where = append(where, "(historic_fts.entity_type = 'topic' AND COALESCE(t.updated_at, t.created_at) >= ? OR historic_fts.entity_type = 'file' AND COALESCE(f.updated_at, f.created_at, f.mtime) >= ?)")
		args = append(args, options.UpdatedAfter, options.UpdatedAfter)
	}
	if options.UpdatedBefore != "" {
		where = append(where, "(historic_fts.entity_type = 'topic' AND COALESCE(t.updated_at, t.created_at) <= ? OR historic_fts.entity_type = 'file' AND COALESCE(f.updated_at, f.created_at, f.mtime) <= ?)")
		args = append(args, options.UpdatedBefore, options.UpdatedBefore)
	}
	for _, tag := range options.Tags {
		where = append(where, `(EXISTS (SELECT 1 FROM json_each(CASE WHEN historic_fts.entity_type = 'topic' THEN t.tags ELSE '[]' END) WHERE value = ?) OR EXISTS (SELECT 1 FROM json_each(CASE WHEN historic_fts.entity_type = 'file' THEN f.tags ELSE '[]' END) WHERE value = ?))`)
		args = append(args, tag, tag)
	}
	return where, args
}

func fallbackContent(database *sql.DB, id string) string {
	var content string
	if err := database.QueryRow("SELECT content FROM files WHERE id = ?", id).Scan(&content); err != nil {
		return ""
	}
	return content
}

func matchedFields(kind, title, description string, tags []string, path, keyword, snippetText string) []string {
	terms := strings.Fields(strings.ToLower(keyword))
	contains := func(value string) bool {
		value = strings.ToLower(value)
		for _, term := range terms {
			if strings.Contains(value, term) {
				return true
			}
		}
		return false
	}
	matches := make([]string, 0)
	if contains(title) {
		matches = append(matches, "title")
	}
	if contains(description) {
		matches = append(matches, "description")
	}
	for _, tag := range tags {
		if contains(tag) {
			matches = append(matches, "tags")
			break
		}
	}
	if contains(path) {
		matches = append(matches, "path")
	}
	if kind == "file" && snippetText != "" && len(matches) == 0 {
		matches = append(matches, "content")
	}
	return matches
}

func decodeStrings(value string) []string {
	var decoded []string
	if json.Unmarshal([]byte(value), &decoded) != nil || decoded == nil {
		return []string{}
	}
	return decoded
}

func prepareQuery(keyword string) (string, error) {
	parts := strings.Fields(keyword)
	if len(parts) == 0 {
		return "", fmt.Errorf("keyword must not be empty")
	}
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		quoted = append(quoted, `"`+strings.ReplaceAll(part, `"`, `""`)+`"`)
	}
	if len(quoted) == 0 {
		return "", fmt.Errorf("keyword must not be empty")
	}
	return strings.Join(quoted, " AND "), nil
}

func validType(value string) bool {
	switch value {
	case "topic", "meta", "prd", "spec", "issue", "note", "decision", "task", "file", "historic_file", "asset":
		return true
	default:
		return false
	}
}
func snippet(body, keyword string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	words := strings.Fields(keyword)
	if len(words) == 0 {
		return firstLine(body)
	}
	lowerBody := strings.ToLower(body)
	lowerKeyword := strings.ToLower(words[0])
	if index := strings.Index(lowerBody, lowerKeyword); index >= 0 {
		start := index - 80
		if start < 0 {
			start = 0
		}
		end := index + len(lowerKeyword) + 120
		if end > len(body) {
			end = len(body)
		}
		return strings.TrimSpace(body[start:end])
	}
	return firstLine(body)
}

// HighlightHuman adds terminal-safe emphasis to matching terms for human output.
// It is intentionally not used in Result.Snippet, keeping JSON plain text.
func HighlightHuman(text, keyword string) string {
	if text == "" || strings.TrimSpace(keyword) == "" {
		return text
	}
	terms := make([]string, 0)
	for _, term := range strings.Fields(keyword) {
		term = strings.TrimFunc(term, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsPunct(r) })
		if term != "" {
			terms = append(terms, regexp.QuoteMeta(term))
		}
	}
	if len(terms) == 0 {
		return text
	}
	pattern := regexp.MustCompile(`(?i)(` + strings.Join(terms, "|") + `)`)
	return pattern.ReplaceAllString(text, "\x1b[1;33m$1\x1b[0m")
}

func firstLine(body string) string {
	if index := strings.IndexByte(body, '\n'); index >= 0 {
		return strings.TrimSpace(body[:index])
	}
	return strings.TrimSpace(body)
}

// Keep deterministic sorting available to callers/tests that combine result sets.
func sortResults(results []Result) {
	sort.Slice(results, func(i, j int) bool { return results[i].Path < results[j].Path })
}
