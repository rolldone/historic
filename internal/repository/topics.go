package repository

import (
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
)

var topicFolderPattern = regexp.MustCompile(`^([0-9]{5})-(.+)$`)

// TopicStore creates and reads topic directories in the working workspace.
type TopicStore struct {
	Workspace config.Workspace
}

// NewTopicStore creates a topic store rooted at workspace.
func NewTopicStore(workspace config.Workspace) TopicStore {
	return TopicStore{Workspace: workspace}
}

// CreateTopic creates a topic and its required _meta.md atomically enough to
// leave no topic directory behind when metadata generation fails.
func (store TopicStore) CreateTopic(title string, requestedID string) (domain.Topic, error) {
	if strings.TrimSpace(title) == "" {
		return domain.Topic{}, fmt.Errorf("%w: title is empty", domain.ErrInvalidSlug)
	}
	slug, err := domain.SlugTitle(title)
	if err != nil {
		return domain.Topic{}, err
	}
	id, err := store.nextID(requestedID)
	if err != nil {
		return domain.Topic{}, err
	}
	folderName, err := domain.TopicFolderName(id, title)
	if err != nil {
		return domain.Topic{}, err
	}
	topicPath := filepath.Join(store.Workspace.Histories, folderName)
	if _, err := os.Stat(topicPath); err == nil {
		return domain.Topic{}, fmt.Errorf("%w: %s", domain.ErrConflict, store.Workspace.RelativePath(topicPath))
	} else if !errors.Is(err, os.ErrNotExist) {
		return domain.Topic{}, fmt.Errorf("inspect topic path %s: %w", topicPath, err)
	}

	created := dateToday()
	metadata := domain.Frontmatter{ID: id, Title: title, Status: domain.StatusCreate, Created: created}
	if err := os.Mkdir(topicPath, 0o755); err != nil {
		return domain.Topic{}, fmt.Errorf("create topic directory %s: %w", topicPath, err)
	}
	metaPath := filepath.Join(topicPath, "_meta.md")
	if err := markdown.WriteMetaFile(metaPath, metadata, "", nil, ""); err != nil {
		_ = os.Remove(topicPath)
		return domain.Topic{}, fmt.Errorf("create topic metadata: %w", err)
	}
	return domain.Topic{ID: id, Title: title, Status: domain.StatusCreate, Created: parseDate(created), Path: topicPath, Slug: slug}, nil
}

func (store TopicStore) nextID(requested string) (domain.ID, error) {
	if requested != "" {
		id, err := domain.ParseID(requested)
		if err != nil {
			return "", err
		}
		if store.topicIDExists(id) {
			return "", fmt.Errorf("%w: %s", domain.ErrDuplicateID, id)
		}
		return id, nil
	}

	used := make(map[int]struct{})
	for _, directory := range []string{store.Workspace.Histories, store.Workspace.Database} {
		entries, err := os.ReadDir(directory)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("scan topics in %s: %w", directory, err)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			match := topicFolderPattern.FindStringSubmatch(entry.Name())
			if len(match) != 3 {
				continue
			}
			number, _ := strconv.Atoi(match[1])
			used[number] = struct{}{}
		}
	}
	for number := 1; number <= 99999; number++ {
		if _, exists := used[number]; !exists {
			return domain.ID(fmt.Sprintf("%05d", number)), nil
		}
	}
	return "", fmt.Errorf("%w: no available five-digit topic ID", domain.ErrConflict)
}

func (store TopicStore) topicIDExists(id domain.ID) bool {
	prefix := id.String() + "-"
	for _, directory := range []string{store.Workspace.Histories, store.Workspace.Database} {
		entries, err := os.ReadDir(directory)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) {
				return true
			}
		}
	}
	return false
}

// ListTopicDirectories returns canonical topic folders from active and archive roots.
func (store TopicStore) ListTopicDirectories() ([]string, error) {
	var result []string
	for _, directory := range []string{store.Workspace.Histories, store.Workspace.Database} {
		entries, err := os.ReadDir(directory)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() && topicFolderPattern.MatchString(entry.Name()) {
				result = append(result, filepath.Join(directory, entry.Name()))
			}
		}
	}
	sort.Strings(result)
	return result, nil
}

func dateToday() string { return timeNow().Format("2006-01-02") }

var timeNow = func() time.Time { return time.Now() }

func parseDate(value string) time.Time {
	parsed, _ := time.Parse("2006-01-02", value)
	return parsed
}
