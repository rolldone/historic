package query

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"historic/internal/config"
	"historic/internal/domain"

	_ "modernc.org/sqlite"
)

const (
	// FileTypeHistoric identifies a managed Historic Markdown file.
	FileTypeHistoric = "historic_file"
	// FileTypeAsset identifies a non-managed topic asset.
	FileTypeAsset = "asset"

	// Keep the original package-local names as compatibility aliases for
	// callers and tests that are in this package.
	fileTypeHistoric = FileTypeHistoric
	fileTypeAsset    = FileTypeAsset
)

// Options contains composable filters for topic and file queries. Dates are
// compared lexically, so callers should provide ISO-8601 values.
type Options struct {
	Status    domain.Status
	StatusNot []domain.Status
	Tags      []string
	Storage   domain.StorageState
	Type      string
	TopicID   domain.ID
	Folder    string

	CreatedAfter  string
	CreatedBefore string
	UpdatedAfter  string
	UpdatedBefore string
	Sort          string
	Page          int
	PageSize      int

	// MinFiles and MaxFiles are aggregate filters. A zero MaxFiles means no
	// upper bound; MinFiles defaults to zero and therefore includes empty topics.
	MinFiles int
	MaxFiles int
}

// TopicSummary is a computed, read-only summary of one logical topic.
//
// The field declaration order is part of the JSON contract: encoding/json
// emits fields in this order, which keeps aggregate output stable for callers.
type TopicSummary struct {
	ID                string   `json:"id"`
	NumPadded         string   `json:"num_padded"`
	Title             string   `json:"title"`
	Description       string   `json:"description,omitempty"`
	Slug              string   `json:"slug"`
	Path              string   `json:"path"`
	Storage           string   `json:"storage"`
	CreatedAt         string   `json:"created_at"`
	UpdatedAt         string   `json:"updated_at,omitempty"`
	Tags              []string `json:"tags"`
	Related           []string `json:"related"`
	TotalFiles        int      `json:"total_files"`
	ActiveFiles       int      `json:"active_files"`
	ResolvedFiles     int      `json:"resolved_files"`
	CompleteFiles     int      `json:"complete_files"`
	CancelledFiles    int      `json:"cancelled_files"`
	ComputedStatus    string   `json:"computed_status,omitempty"`
	LastFileUpdatedAt string   `json:"last_file_updated_at,omitempty"`
}

// FileResult is a file read-model record with its aggregate topic context.
type FileResult struct {
	ID          string       `json:"id"`
	TopicID     string       `json:"topic_id"`
	Type        string       `json:"type"`
	Path        string       `json:"path"`
	Filename    string       `json:"filename"`
	Title       string       `json:"title"`
	Description string       `json:"description,omitempty"`
	Status      string       `json:"status,omitempty"`
	Tags        []string     `json:"tags"`
	Related     []string     `json:"related"`
	AssetKind   string       `json:"asset_kind,omitempty"`
	CreatedAt   string       `json:"created_at,omitempty"`
	UpdatedAt   string       `json:"updated_at,omitempty"`
	Mtime       string       `json:"mtime"`
	Hash        string       `json:"hash"`
	Size        int64        `json:"size"`
	WordCount   int          `json:"word_count"`
	Topic       TopicSummary `json:"topic"`
}

// Service queries the rebuildable SQLite read model.
type Service struct {
	workspace config.Workspace
}

type Pagination struct {
	Page     int  `json:"page"`
	PageSize int  `json:"page_size"`
	HasMore  bool `json:"has_more"`
	NextPage *int `json:"next_page"`
}

type Page[T any] struct {
	Items      []T        `json:"items"`
	Pagination Pagination `json:"pagination"`
}

func NormalizePagination(page, pageSize int) (Pagination, error) {
	if page == 0 {
		page = 1
	}
	if page < 1 {
		return Pagination{}, fmt.Errorf("page must be at least 1")
	}
	if pageSize == 0 {
		pageSize = 20
	}
	if pageSize < 1 {
		return Pagination{}, fmt.Errorf("page-size must be at least 1")
	}
	if pageSize > 100 {
		return Pagination{}, fmt.Errorf("page-size must not exceed 100")
	}
	maxInt := int(^uint(0) >> 1)
	if page > 1 && page-1 > maxInt/pageSize {
		return Pagination{}, fmt.Errorf("page and page-size produce an offset that overflows")
	}
	return Pagination{Page: page, PageSize: pageSize}, nil
}

// NewService creates a query service for a Historic workspace.
func NewService(workspace config.Workspace) Service {
	return Service{workspace: workspace}
}

// QueryTopics returns aggregate topic summaries in deterministic order.
func (service Service) QueryTopics(options Options) ([]TopicSummary, error) {
	page, err := service.QueryTopicsPage(options)
	if err != nil {
		return nil, err
	}
	return page.Items, nil
}

// QueryTopicsPage returns paginated aggregate topic summaries.
func (service Service) QueryTopicsPage(options Options) (Page[TopicSummary], error) {
	pagination, err := NormalizePagination(options.Page, options.PageSize)
	if err != nil {
		return Page[TopicSummary]{}, err
	}
	if err := validateOptions(options); err != nil {
		return Page[TopicSummary]{}, err
	}
	database, err := service.open()
	if err != nil {
		return Page[TopicSummary]{}, err
	}
	defer database.Close()

	topicWhere, topicArgs := topicPredicates(options)
	joinPredicate, fileArgs := fileJoinPredicate(options)
	args := append(append([]any{}, fileArgs...), topicArgs...)
	args = append(args, options.MinFiles)
	if options.MaxFiles > 0 {
		args = append(args, options.MaxFiles)
	}

	query := `WITH grouped AS (
		SELECT
			t.id, t.num_padded, t.title, t.description, t.slug, t.path, t.storage,
			t.created_at, t.updated_at, t.tags, t.related,
			COUNT(CASE WHEN f.type = 'historic_file' THEN 1 END) AS total_files,
			SUM(CASE WHEN f.type = 'historic_file' AND f.status NOT IN ('complete', 'cancelled') THEN 1 ELSE 0 END) AS active_files,
			COUNT(CASE WHEN f.type = 'historic_file' AND f.status IN ('complete', 'cancelled') THEN 1 END) AS resolved_files,
			COUNT(CASE WHEN f.type = 'historic_file' AND f.status = 'complete' THEN 1 END) AS complete_files,
			COUNT(CASE WHEN f.type = 'historic_file' AND f.status = 'cancelled' THEN 1 END) AS cancelled_files,
			MAX(CASE WHEN f.type = 'historic_file' THEN COALESCE(f.updated_at, f.created_at, f.mtime) END) AS last_file_updated_at
		FROM topics AS t
		LEFT JOIN files AS f ON f.topic_id = t.id AND ` + joinPredicate + `
		WHERE ` + strings.Join(topicWhere, " AND ") + `
		GROUP BY t.id, t.num_padded, t.title, t.description, t.slug, t.path, t.storage,
			t.created_at, t.updated_at, t.tags, t.related
		HAVING COUNT(CASE WHEN f.type = 'historic_file' THEN 1 END) >= ?` + func() string {
		if options.MaxFiles > 0 {
			return " AND COUNT(CASE WHEN f.type = 'historic_file' THEN 1 END) <= ?"
		}
		return ""
	}() + `
	)
	SELECT id, num_padded, title, description, slug, path, storage, created_at, updated_at,
		tags, related, total_files, active_files, resolved_files, complete_files, cancelled_files,
		CASE
			WHEN total_files = 0 THEN NULL
			WHEN active_files > 0 THEN 'active'
			WHEN cancelled_files = resolved_files THEN 'cancelled'
			ELSE 'complete'
		END AS computed_status,
		last_file_updated_at
	FROM grouped
	ORDER BY ` + aggregateOrder(options.Sort) + ` LIMIT ? OFFSET ?`
	args = append(args, pagination.PageSize+1, (pagination.Page-1)*pagination.PageSize)

	rows, err := database.Query(query, args...)
	if err != nil {
		return Page[TopicSummary]{}, fmt.Errorf("query topic aggregates: %w", err)
	}
	defer rows.Close()
	results := make([]TopicSummary, 0, pagination.PageSize+1)
	for rows.Next() {
		var result TopicSummary
		var description, updatedAt, computedStatus, lastFileUpdatedAt sql.NullString
		var tagsJSON, relatedJSON string
		if err := rows.Scan(&result.ID, &result.NumPadded, &result.Title, &description, &result.Slug, &result.Path, &result.Storage, &result.CreatedAt, &updatedAt, &tagsJSON, &relatedJSON, &result.TotalFiles, &result.ActiveFiles, &result.ResolvedFiles, &result.CompleteFiles, &result.CancelledFiles, &computedStatus, &lastFileUpdatedAt); err != nil {
			return Page[TopicSummary]{}, fmt.Errorf("read topic aggregate: %w", err)
		}
		result.Description = description.String
		result.UpdatedAt = updatedAt.String
		result.ComputedStatus = computedStatus.String
		result.LastFileUpdatedAt = lastFileUpdatedAt.String
		result.Tags = decodeStrings(tagsJSON)
		result.Related = decodeStrings(relatedJSON)
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return Page[TopicSummary]{}, fmt.Errorf("iterate topic aggregates: %w", err)
	}
	return makePage(results, pagination), nil
}

// QueryFiles returns files matching the same topic/file filters as
// QueryTopics. Each result includes the computed topic aggregate as context.
func (service Service) QueryFiles(options Options) ([]FileResult, error) {
	page, err := service.QueryFilesPage(options)
	if err != nil {
		return nil, err
	}
	return page.Items, nil
}

// QueryFilesPage returns paginated file results with topic context.
func (service Service) QueryFilesPage(options Options) (Page[FileResult], error) {
	pagination, err := NormalizePagination(options.Page, options.PageSize)
	if err != nil {
		return Page[FileResult]{}, err
	}
	if err := validateOptions(options); err != nil {
		return Page[FileResult]{}, err
	}
	topics, err := service.QueryTopics(topicContextOptions(options))
	if err != nil {
		return Page[FileResult]{}, err
	}
	topicByID := make(map[string]TopicSummary, len(topics))
	for _, topic := range topics {
		topicByID[topic.ID] = topic
	}
	database, err := service.open()
	if err != nil {
		return Page[FileResult]{}, err
	}
	defer database.Close()

	topicWhere, topicArgs := topicPredicates(options)
	joinPredicate, fileArgs := fileJoinPredicate(options)
	args := append(append([]any{}, topicArgs...), fileArgs...)
	rows, err := database.Query(`SELECT CAST(f.id AS TEXT), f.topic_id, f.type, f.path, f.filename, f.title, f.description,
		f.status, f.tags, f.related, f.asset_kind, f.created_at, f.updated_at, f.mtime, f.hash,
		f.size, f.word_count
		FROM files AS f JOIN topics AS t ON t.id = f.topic_id
		WHERE `+strings.Join(topicWhere, " AND ")+` AND `+joinPredicate+`
		ORDER BY `+fileOrder(options.Sort)+` LIMIT ? OFFSET ?`, append(args, pagination.PageSize+1, (pagination.Page-1)*pagination.PageSize)...)
	if err != nil {
		return Page[FileResult]{}, fmt.Errorf("query files: %w", err)
	}
	defer rows.Close()
	results := make([]FileResult, 0, pagination.PageSize+1)
	for rows.Next() {
		var result FileResult
		var description, status, assetKind, createdAt, updatedAt sql.NullString
		var tagsJSON, relatedJSON string
		if err := rows.Scan(&result.ID, &result.TopicID, &result.Type, &result.Path, &result.Filename, &result.Title, &description, &status, &tagsJSON, &relatedJSON, &assetKind, &createdAt, &updatedAt, &result.Mtime, &result.Hash, &result.Size, &result.WordCount); err != nil {
			return Page[FileResult]{}, fmt.Errorf("read file result: %w", err)
		}
		result.Description = description.String
		result.Status = status.String
		result.AssetKind = assetKind.String
		result.CreatedAt = createdAt.String
		result.UpdatedAt = updatedAt.String
		result.Tags = decodeStrings(tagsJSON)
		result.Related = decodeStrings(relatedJSON)
		result.Topic = topicByID[result.TopicID]
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return Page[FileResult]{}, fmt.Errorf("iterate file results: %w", err)
	}
	return makePage(results, pagination), nil
}

func makePage[T any](items []T, pagination Pagination) Page[T] {
	if len(items) > pagination.PageSize {
		items = items[:pagination.PageSize]
		pagination.HasMore = true
		next := pagination.Page + 1
		pagination.NextPage = &next
	}
	return Page[T]{Items: items, Pagination: pagination}
}

func aggregateOrder(sortOption string) string {
	switch sortOption {
	case "title":
		return "title ASC, path ASC, id ASC"
	case "created":
		return "created_at DESC, path ASC, id ASC"
	case "updated":
		return "last_file_updated_at DESC, updated_at DESC, path ASC, id ASC"
	default:
		return "num_padded ASC, path ASC, id ASC"
	}
}

func fileOrder(sortOption string) string {
	switch sortOption {
	case "title":
		return "f.title ASC, f.path ASC, f.id ASC"
	case "created":
		return "f.created_at DESC, f.path ASC, f.id ASC"
	case "updated":
		return "COALESCE(f.updated_at, f.created_at, f.mtime) DESC, f.path ASC, f.id ASC"
	default:
		return "t.num_padded ASC, f.path ASC, f.id ASC"
	}
}

func topicContextOptions(options Options) Options {
	context := options
	context.Status = ""
	context.Type = ""
	context.Folder = ""
	context.Tags = nil
	context.CreatedAfter = ""
	context.CreatedBefore = ""
	context.UpdatedAfter = ""
	context.UpdatedBefore = ""
	context.Sort = ""
	context.MinFiles = 0
	context.MaxFiles = 0
	context.Page = 0
	context.PageSize = 0
	return context
}

// QueryTopics is a convenience function for callers that do not need to keep
// a service value.
func QueryTopics(workspace config.Workspace, options Options) ([]TopicSummary, error) {
	return NewService(workspace).QueryTopics(options)
}

// QueryFiles is a convenience function for callers that do not need to keep a
// service value.
func QueryFiles(workspace config.Workspace, options Options) ([]FileResult, error) {
	return NewService(workspace).QueryFiles(options)
}

func (service Service) open() (*sql.DB, error) {
	database, err := sql.Open("sqlite", service.workspace.Index)
	if err != nil {
		return nil, fmt.Errorf("open query index: %w", err)
	}
	if err := database.Ping(); err != nil {
		database.Close()
		return nil, fmt.Errorf("query index unavailable: run historic rebuild: %w", err)
	}
	return database, nil
}

func validateOptions(options Options) error {
	if _, err := NormalizePagination(options.Page, options.PageSize); err != nil {
		return err
	}
	if options.Status != "" && !options.Status.IsValid() {
		return fmt.Errorf("invalid status %q", options.Status)
	}
	for _, status := range options.StatusNot {
		if !status.IsValid() {
			return fmt.Errorf("invalid excluded status %q", status)
		}
		if status == options.Status && options.Status != "" {
			return fmt.Errorf("status %q cannot appear in both inclusion and exclusion filters", status)
		}
	}
	if options.Storage != "" && !options.Storage.IsValid() {
		return fmt.Errorf("invalid storage %q", options.Storage)
	}
	if options.Type != "" && options.Type != fileTypeHistoric && options.Type != fileTypeAsset {
		return fmt.Errorf("invalid file type %q", options.Type)
	}
	if options.Sort != "" && options.Sort != "updated" && options.Sort != "created" && options.Sort != "title" {
		return fmt.Errorf("invalid sort %q; use updated, created, or title", options.Sort)
	}
	if options.TopicID != "" && !options.TopicID.Valid() {
		return fmt.Errorf("invalid topic ID %q", options.TopicID)
	}
	if options.MinFiles < 0 || options.MaxFiles < 0 {
		return fmt.Errorf("aggregate file limits must not be negative")
	}
	if options.MaxFiles > 0 && options.MinFiles > options.MaxFiles {
		return fmt.Errorf("minimum files cannot exceed maximum files")
	}
	if options.Folder != "" {
		folder := filepath.ToSlash(strings.Trim(strings.TrimSpace(options.Folder), "/"))
		if folder == "" || filepath.IsAbs(options.Folder) || folder == "." || strings.HasPrefix(folder, "../") || strings.Contains(folder, "/../") || strings.HasSuffix(folder, "/..") || folder == ".." {
			return fmt.Errorf("invalid folder filter %q", options.Folder)
		}
	}
	return nil
}

func topicPredicates(options Options) ([]string, []any) {
	where := []string{"1 = 1"}
	args := make([]any, 0)
	if options.Status != "" {
		where = append(where, "COALESCE(t.computed_status, '') = ?")
		args = append(args, options.Status.String())
	}
	for _, status := range options.StatusNot {
		where = append(where, "COALESCE(t.computed_status, '') <> ?")
		args = append(args, status.String())
	}
	if options.TopicID != "" {
		where = append(where, "t.id = ?")
		args = append(args, options.TopicID.String())
	}
	if options.Storage != "" {
		where = append(where, "t.storage = ?")
		args = append(args, options.Storage.String())
	}
	if options.CreatedAfter != "" {
		where = append(where, "t.created_at >= ?")
		args = append(args, options.CreatedAfter)
	}
	if options.CreatedBefore != "" {
		where = append(where, "t.created_at <= ?")
		args = append(args, options.CreatedBefore)
	}
	if options.UpdatedAfter != "" {
		where = append(where, "COALESCE(t.updated_at, t.created_at) >= ?")
		args = append(args, options.UpdatedAfter)
	}
	if options.UpdatedBefore != "" {
		where = append(where, "COALESCE(t.updated_at, t.created_at) <= ?")
		args = append(args, options.UpdatedBefore)
	}
	return where, args
}

func fileJoinPredicate(options Options) (string, []any) {
	predicates := []string{"1 = 1"}
	args := make([]any, 0)
	if options.Type != "" {
		predicates = append(predicates, "f.type = ?")
		args = append(args, options.Type)
	}
	if options.Status != "" {
		predicates = append(predicates, "f.status = ?")
		args = append(args, options.Status.String())
	}
	for _, status := range options.StatusNot {
		predicates = append(predicates, "COALESCE(f.status, '') <> ?")
		args = append(args, status.String())
	}
	if options.Folder != "" {
		folder := filepath.ToSlash(strings.Trim(strings.TrimSpace(options.Folder), "/"))
		predicates = append(predicates, "(f.path = ? OR f.path LIKE ?)")
		args = append(args, folder, folder+"/%")
	}
	if options.CreatedAfter != "" {
		predicates = append(predicates, "COALESCE(f.created_at, f.mtime) >= ?")
		args = append(args, options.CreatedAfter)
	}
	if options.CreatedBefore != "" {
		predicates = append(predicates, "COALESCE(f.created_at, f.mtime) <= ?")
		args = append(args, options.CreatedBefore)
	}
	if options.UpdatedAfter != "" {
		predicates = append(predicates, "COALESCE(f.updated_at, f.created_at, f.mtime) >= ?")
		args = append(args, options.UpdatedAfter)
	}
	if options.UpdatedBefore != "" {
		predicates = append(predicates, "COALESCE(f.updated_at, f.created_at, f.mtime) <= ?")
		args = append(args, options.UpdatedBefore)
	}
	for _, tag := range options.Tags {
		predicates = append(predicates, `(EXISTS (SELECT 1 FROM json_each(COALESCE(t.tags, '[]')) WHERE value = ?) OR EXISTS (SELECT 1 FROM json_each(COALESCE(f.tags, '[]')) WHERE value = ?))`)
		args = append(args, tag, tag)
	}
	return strings.Join(predicates, " AND "), args
}

func decodeStrings(value string) []string {
	var decoded []string
	if err := json.Unmarshal([]byte(value), &decoded); err != nil || decoded == nil {
		return []string{}
	}
	return decoded
}

// Keep the package's ordering contract explicit if callers sort copied values.
func sortTopicSummaries(results []TopicSummary) {
	sort.Slice(results, func(i, j int) bool {
		if results[i].NumPadded != results[j].NumPadded {
			return results[i].NumPadded < results[j].NumPadded
		}
		return results[i].Path < results[j].Path
	})
}
