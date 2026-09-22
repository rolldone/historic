package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	StatusCreate    Status = "create"
	StatusDraft     Status = "draft"
	StatusPending   Status = "pending"
	StatusProgress  Status = "progress"
	StatusReview    Status = "review"
	StatusBlocked   Status = "blocked"
	StatusComplete  Status = "complete"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
	StatusArchived  Status = "archived"
)

var (
	ErrDuplicateID   = errors.New("topic ID already exists")
	ErrInvalidID     = errors.New("invalid topic ID")
	ErrInvalidFileID = errors.New("invalid managed file ID")
	ErrInvalidSlug   = errors.New("invalid topic slug")
	ErrInvalidStatus = errors.New("invalid status")
	ErrTopicMissing  = errors.New("topic not found")
	ErrConflict      = errors.New("operation conflicts with existing data")
)

// Status is a lifecycle state. Values are intentionally case-sensitive.
type Status string

func (s Status) String() string { return string(s) }

func (s Status) IsValid() bool {
	_, ok := validStatuses[s]
	return ok
}

func (s Status) IsOpen() bool {
	_, ok := openStatuses[s]
	return ok
}

func (s Status) IsClose() bool {
	_, ok := closeStatuses[s]
	return ok
}

func ParseStatus(value string) (Status, error) {
	status := Status(value)
	if !status.IsValid() {
		return "", fmt.Errorf("%w: %q", ErrInvalidStatus, value)
	}
	return status, nil
}

var validStatuses = map[Status]struct{}{
	StatusCreate: {}, StatusDraft: {}, StatusPending: {}, StatusProgress: {}, StatusReview: {}, StatusBlocked: {},
	StatusComplete: {}, StatusFailed: {}, StatusCancelled: {}, StatusArchived: {},
}

var openStatuses = map[Status]struct{}{
	StatusCreate: {}, StatusDraft: {}, StatusPending: {}, StatusProgress: {}, StatusReview: {}, StatusBlocked: {},
}

var closeStatuses = map[Status]struct{}{
	StatusComplete: {}, StatusFailed: {}, StatusCancelled: {}, StatusArchived: {},
}

// ID represents a five-digit topic identifier and preserves its padded form.
type ID string

var idPattern = regexp.MustCompile(`^[0-9]{5}$`)
var fileIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// FileID is the canonical lowercase UUIDv7 identifier of a managed file.
type FileID string

func ParseFileID(value string) (FileID, error) {
	id := FileID(value)
	if !id.Valid() {
		return "", fmt.Errorf("%w: %q must be a lowercase canonical UUIDv7", ErrInvalidFileID, value)
	}
	return id, nil
}

func (id FileID) String() string { return string(id) }
func (id FileID) Valid() bool    { return fileIDPattern.MatchString(string(id)) }

func ParseID(value string) (ID, error) {
	if !idPattern.MatchString(value) {
		return "", fmt.Errorf("%w: %q must contain exactly five digits", ErrInvalidID, value)
	}
	return ID(value), nil
}

func (id ID) String() string { return string(id) }

func (id ID) Number() int {
	n, _ := strconv.Atoi(string(id))
	return n
}

func (id ID) Valid() bool { return idPattern.MatchString(string(id)) }

// Topic is the metadata and identity of a Historic topic.
type Topic struct {
	ID          ID
	Title       string
	Description string
	Created     time.Time
	Updated     *time.Time
	Tags        []string
	Related     []ID
	Path        string
	Slug        string
}

// Entry is a Markdown document belonging to a topic.
type Entry struct {
	ID          FileID
	Title       string
	Description string
	Status      Status
	Created     time.Time
	Updated     *time.Time
	Tags        []string
	Related     []ID
	Path        string
	Filename    string
	Content     string
	WordCount   int
}

// WorkOrder is a sortable work-order document within a topic.
type WorkOrder struct {
	Entry
	Order int
}

// Frontmatter is the metadata serialized at the start of a Markdown file.
type Frontmatter struct {
	ID          ID       `yaml:"id,omitempty"`
	Title       string   `yaml:"title"`
	Description string   `yaml:"description"`
	Status      Status   `yaml:"status"`
	Created     string   `yaml:"created"`
	Updated     string   `yaml:"updated,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
	Related     []ID     `yaml:"related,omitempty"`
}

// IndexRecord is the rebuildable SQLite index representation of an entry.
type IndexRecord struct {
	Num         int
	NumPadded   string
	Type        string
	Title       string
	Description string
	Status      Status
	Tags        []string
	Related     []ID
	CreatedAt   time.Time
	UpdatedAt   *time.Time
	Path        string
	FolderID    ID
	FolderSlug  string
	Subfolder   string
	Filename    string
	FileOrder   int
	Content     string
	WordCount   int
	Mtime       time.Time
	Hash        string
}

// SlugTitle converts a title to the filesystem slug used in topic folders.
func SlugTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", fmt.Errorf("%w: title is empty", ErrInvalidSlug)
	}

	var b strings.Builder
	separator := false
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			if separator && b.Len() > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r)
			separator = false
		case r == '-' || r == '_' || r == ' ' || r == '\t':
			separator = true
		}
	}

	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "", fmt.Errorf("%w: title %q has no usable characters", ErrInvalidSlug, title)
	}
	return slug, nil
}

// TopicFolderName returns the canonical ID-slug folder name.
func TopicFolderName(id ID, title string) (string, error) {
	if !id.Valid() {
		return "", fmt.Errorf("%w: %q", ErrInvalidID, id)
	}
	slug, err := SlugTitle(title)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s", id, slug), nil
}

// CanTransition reports whether a topic may move from current to next.
func CanTransition(current, next Status) bool {
	if !current.IsValid() || !next.IsValid() {
		return false
	}
	if current == next {
		return true
	}
	return !current.IsClose()
}

// ValidateTransition validates a lifecycle transition and returns a domain error.
func ValidateTransition(current, next Status) error {
	if !current.IsValid() {
		return fmt.Errorf("%w: current status %q", ErrInvalidStatus, current)
	}
	if !next.IsValid() {
		return fmt.Errorf("%w: next status %q", ErrInvalidStatus, next)
	}
	if current == next {
		return nil
	}
	if !CanTransition(current, next) {
		return fmt.Errorf("%w: cannot transition from %q to %q", ErrConflict, current, next)
	}
	return nil
}
