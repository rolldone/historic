package search

import (
	"database/sql"
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

// Options controls a FTS5 search.
type Options struct {
	Keyword      string
	Status       domain.Status
	Folder       string
	Type         string
	ID           domain.ID
	ActiveOnly   bool
	ArchivedOnly bool
}

// Result is one matching Markdown entry or topic metadata file.
type Result struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	Path    string `json:"path"`
	Snippet string `json:"snippet"`
	Active  bool   `json:"active"`
}

// Find queries the rebuildable SQLite FTS5 index without invoking a shell.
func Find(workspace config.Workspace, options Options) ([]Result, error) {
	keyword := strings.TrimSpace(options.Keyword)
	if keyword == "" {
		return nil, fmt.Errorf("keyword must not be empty")
	}
	if options.ActiveOnly && options.ArchivedOnly {
		return nil, fmt.Errorf("active and archived filters cannot be combined")
	}
	if options.Type != "" && !validType(options.Type) {
		return nil, fmt.Errorf("invalid type %q", options.Type)
	}
	database, err := sql.Open("sqlite", workspace.Index)
	if err != nil {
		return nil, fmt.Errorf("open search index: %w", err)
	}
	defer database.Close()
	if err := database.Ping(); err != nil {
		return nil, fmt.Errorf("search index unavailable: run historic rebuild: %w", err)
	}
	var ftsTable string
	if err := database.QueryRow("SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'historic_fts'").Scan(&ftsTable); err != nil {
		return nil, fmt.Errorf("FTS5 index unavailable: run historic rebuild: %w", err)
	}
	var recordsTable string
	if err := database.QueryRow("SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'index_records'").Scan(&recordsTable); err != nil {
		return nil, fmt.Errorf("SQLite index unavailable: run historic rebuild: %w", err)
	}
	query, err := prepareQuery(keyword)
	if err != nil {
		return nil, err
	}
	where := []string{"historic_fts MATCH ?"}
	args := []any{query}
	if options.Status != "" {
		where = append(where, "r.status = ?")
		args = append(args, options.Status.String())
	}
	if options.Type != "" {
		where = append(where, "r.type = ?")
		args = append(args, options.Type)
	}
	if options.ID != "" {
		where = append(where, "r.num_padded = ?")
		args = append(args, options.ID.String())
	}
	if options.Folder != "" {
		folder := strings.Trim(filepath.ToSlash(strings.TrimSpace(options.Folder)), "/")
		if folder == "" || filepath.IsAbs(options.Folder) || strings.Contains(folder, "..") {
			return nil, fmt.Errorf("invalid folder filter %q", options.Folder)
		}
		where = append(where, "(r.path = ? OR r.path LIKE ?)")
		args = append(args, folder, folder+"/%")
	}
	if options.ActiveOnly {
		where = append(where, "r.path NOT LIKE '.historic/.database/%'")
	}
	if options.ArchivedOnly {
		where = append(where, "r.path LIKE '.historic/.database/%'")
	}
	statement := `SELECT r.num_padded, r.title, r.status, r.path, r.content, r.type,
		r.path LIKE '.historic/.database/%' AS archived, bm25(historic_fts) AS rank
		FROM historic_fts JOIN index_records AS r ON r.id = historic_fts.rowid
		WHERE ` + strings.Join(where, " AND ") + ` ORDER BY rank ASC, r.path ASC`
	rows, err := database.Query(statement, args...)
	if err != nil {
		return nil, fmt.Errorf("invalid FTS query: %w", err)
	}
	defer rows.Close()
	results := make([]Result, 0)
	for rows.Next() {
		var result Result
		var body, recordType string
		var archived bool
		var rank float64
		if err := rows.Scan(&result.ID, &result.Title, &result.Status, &result.Path, &body, &recordType, &archived, &rank); err != nil {
			return nil, fmt.Errorf("read search result: %w", err)
		}
		result.Snippet = snippet(body, keyword)
		result.Active = !archived
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate search results: %w", err)
	}
	return results, nil
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
	case "meta", "prd", "spec", "issue", "note", "decision", "task", "file":
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
