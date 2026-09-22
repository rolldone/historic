package repository

import (
	"errors"
	"fmt"
	"os"
	"path"
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
)

var (
	topicFolderPattern = regexp.MustCompile(`^([0-9]{5})-(.+)$`)
	workOrderPattern   = regexp.MustCompile(`^([0-9]{2})-(.+)$`)
)

// TopicStore creates and reads topic directories in the working workspace.
type TopicStore struct {
	Workspace config.Workspace
}

// NewTopicStore creates a topic store rooted at workspace.
func NewTopicStore(workspace config.Workspace) TopicStore {
	return TopicStore{Workspace: workspace}
}

// CreateTopic creates a topic and its required _meta.yaml atomically enough to
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
	metadata := markdown.TopicMetadata{ID: id, Title: title, Description: "", Created: created}
	if err := os.Mkdir(topicPath, 0o755); err != nil {
		return domain.Topic{}, fmt.Errorf("create topic directory %s: %w", topicPath, err)
	}
	metaPath := filepath.Join(topicPath, markdown.MetaFilename)
	if err := markdown.WriteTopicMetadata(metaPath, metadata); err != nil {
		_ = os.Remove(topicPath)
		return domain.Topic{}, fmt.Errorf("create topic metadata: %w", err)
	}
	return domain.Topic{ID: id, Title: title, Description: metadata.Description, Created: parseDate(created), Path: topicPath, Slug: slug}, nil
}

// AddEntry creates a Markdown entry in an active topic and keeps canonical
// YAML metadata unchanged.
func (store TopicStore) AddEntry(topicID domain.ID, name string, force bool) (domain.Entry, error) {
	if !topicID.Valid() {
		return domain.Entry{}, fmt.Errorf("%w: %q", domain.ErrInvalidID, topicID)
	}
	topicPath, err := store.activeTopicPath(topicID)
	if err != nil {
		return domain.Entry{}, err
	}
	relativeName, err := normalizeEntryName(name)
	if err != nil {
		return domain.Entry{}, err
	}
	if strings.HasPrefix(relativeName, "wos/") && !workOrderPattern.MatchString(filepath.Base(relativeName)) {
		relativeName, err = store.nextWorkOrderPath(topicPath, relativeName)
		if err != nil {
			return domain.Entry{}, err
		}
	}
	entryPath := filepath.Join(topicPath, filepath.FromSlash(relativeName))
	if _, err := os.Stat(entryPath); err == nil && !force {
		return domain.Entry{}, fmt.Errorf("%w: file already exists: %s (use --force to overwrite)", domain.ErrConflict, store.Workspace.RelativePath(entryPath))
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return domain.Entry{}, fmt.Errorf("inspect entry path %s: %w", entryPath, err)
	}

	meta, err := markdown.ReadTopicMetadata(topicPath)
	if err != nil {
		return domain.Entry{}, fmt.Errorf("read topic metadata: %w", err)
	}
	created := dateToday()
	fileID := domain.FileID("")
	for _, file := range meta.Files {
		if file.Path == relativeName {
			fileID = file.ID
			break
		}
	}
	if fileID == "" {
		fileID, err = identifier.Default.New()
		if err != nil {
			return domain.Entry{}, fmt.Errorf("generate managed file ID: %w", err)
		}
	}
	metadata := domain.Frontmatter{Title: entryTitle(relativeName), Description: "", Status: domain.StatusCreate, Created: created}
	document, err := markdown.NewDocument(metadata, "# "+metadata.Title+"\n")
	if err != nil {
		return domain.Entry{}, fmt.Errorf("create entry metadata: %w", err)
	}
	if err := markdown.WriteFile(entryPath, document); err != nil {
		return domain.Entry{}, fmt.Errorf("write entry: %w", err)
	}
	updatedMeta := append([]markdown.ManifestFile(nil), meta.Files...)
	found := false
	for index := range updatedMeta {
		if updatedMeta[index].Path == relativeName {
			fileType := "file"
			if strings.HasPrefix(relativeName, "wos/") {
				fileType = "task"
			}
			updatedMeta[index] = markdown.ManifestFile{ID: fileID, Path: relativeName, Type: fileType, Status: metadata.Status}
			found = true
			break
		}
	}
	if !found {
		fileType := "file"
		if strings.HasPrefix(relativeName, "wos/") {
			fileType = "task"
		}
		updatedMeta = append(updatedMeta, markdown.ManifestFile{ID: fileID, Path: relativeName, Type: fileType, Status: metadata.Status})
	}
	if _, err := markdown.WriteTopicMetadataManifest(filepath.Join(topicPath, markdown.MetaFilename), meta, updatedMeta, meta.Assets); err != nil {
		if !force {
			_ = os.Remove(entryPath)
		}
		return domain.Entry{}, fmt.Errorf("update topic metadata: %w", err)
	}
	return domain.Entry{ID: fileID, Title: metadata.Title, Description: metadata.Description, Created: parseDate(created), Path: entryPath, Filename: relativeName, Content: document.Body}, nil
}

func (store TopicStore) activeTopicPath(id domain.ID) (string, error) {
	prefix := id.String() + "-"
	entries, err := os.ReadDir(store.Workspace.Histories)
	if errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("%w: %s", domain.ErrTopicMissing, id)
	}
	if err != nil {
		return "", fmt.Errorf("scan active topics: %w", err)
	}
	var found string
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), prefix) || strings.HasPrefix(entry.Name(), ".staging-") {
			continue
		}
		path := filepath.Join(store.Workspace.Histories, entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("%w: topic symlink %s", domain.ErrConflict, store.Workspace.RelativePath(path))
		}
		if !info.IsDir() {
			continue
		}
		if found != "" {
			return "", fmt.Errorf("%w: duplicate open topic ID %s", domain.ErrConflict, id)
		}
		found = path
	}
	if found == "" {
		return "", fmt.Errorf("%w: %s", domain.ErrTopicMissing, id)
	}
	return found, nil
}

func normalizeEntryName(name string) (string, error) {
	name = strings.TrimSpace(strings.ReplaceAll(name, "\\", "/"))
	if name == "" {
		return "", fmt.Errorf("%w: entry name is empty", domain.ErrConflict)
	}
	if filepath.IsAbs(name) || strings.HasPrefix(name, "/") || isWindowsAbsolutePath(name) {
		return "", fmt.Errorf("%w: absolute entry path is not allowed", domain.ErrConflict)
	}
	if strings.HasSuffix(name, "/") {
		return "", fmt.Errorf("%w: entry basename is empty", domain.ErrConflict)
	}
	for _, component := range strings.Split(name, "/") {
		if component == ".." {
			return "", fmt.Errorf("%w: path traversal is not allowed", domain.ErrConflict)
		}
	}
	clean := path.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("%w: path traversal is not allowed", domain.ErrConflict)
	}
	components := strings.Split(clean, "/")
	for _, component := range components[:len(components)-1] {
		if component == "" || component == "." || strings.HasPrefix(component, ".") {
			return "", fmt.Errorf("%w: hidden entry path is not allowed", domain.ErrConflict)
		}
	}

	directory := strings.Join(components[:len(components)-1], "/")
	basename := components[len(components)-1]
	if strings.HasPrefix(basename, ".") {
		return "", fmt.Errorf("%w: hidden entry path is not allowed", domain.ErrConflict)
	}
	for strings.HasSuffix(strings.ToLower(basename), ".md") {
		basename = basename[:len(basename)-len(".md")]
	}
	if basename == "" {
		return "", fmt.Errorf("%w: entry basename is empty", domain.ErrConflict)
	}

	prefix := ""
	if strings.HasPrefix(directory, "wos/") {
		prefix = workOrderPrefix(basename)
		basename = strings.TrimPrefix(basename, prefix)
	}
	slug, err := domain.SlugTitle(basename)
	if err != nil {
		return "", fmt.Errorf("%w: entry name %q has no usable basename", domain.ErrConflict, name)
	}
	basename = prefix + slug + ".md"
	if directory == "" {
		return basename, nil
	}
	return directory + "/" + basename, nil
}

func isWindowsAbsolutePath(name string) bool {
	return len(name) >= 3 && ((name[0] >= 'a' && name[0] <= 'z') || (name[0] >= 'A' && name[0] <= 'Z')) && name[1] == ':' && name[2] == '/'
}

func workOrderPrefix(basename string) string {
	match := workOrderPattern.FindStringSubmatch(basename)
	if len(match) == 3 {
		return match[1] + "-"
	}
	return ""
}

func (store TopicStore) nextWorkOrderPath(topicPath, path string) (string, error) {
	directory := filepath.Join(topicPath, filepath.FromSlash(filepath.Dir(path)))
	entries, err := os.ReadDir(directory)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Sprintf("%s/%02d-%s", filepath.ToSlash(filepath.Dir(path)), 1, filepath.Base(path)), nil
	}
	if err != nil {
		return "", fmt.Errorf("scan work orders: %w", err)
	}
	used := make(map[int]struct{})
	for _, entry := range entries {
		match := workOrderPattern.FindStringSubmatch(entry.Name())
		if len(match) == 3 {
			number, _ := strconv.Atoi(match[1])
			used[number] = struct{}{}
		}
	}
	for number := 1; number <= 99; number++ {
		if _, exists := used[number]; !exists {
			return fmt.Sprintf("%s/%02d-%s", filepath.ToSlash(filepath.Dir(path)), number, filepath.Base(path)), nil
		}
	}
	return "", fmt.Errorf("%w: no available work order number", domain.ErrConflict)
}

func entryTitle(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	base = strings.ReplaceAll(base, "-", " ")
	return strings.TrimSpace(base)
}

func addFileToMeta(body, relativeName string) string {
	link := "- [" + filepath.Base(relativeName) + "](./" + filepath.ToSlash(relativeName) + ")"
	if strings.Contains(body, link) {
		return body
	}
	const heading = "## Files"
	start := strings.Index(body, heading)
	if start < 0 {
		body = strings.TrimRight(body, "\n") + "\n\n" + heading + "\n\n"
		return body + link + "\n"
	}
	sectionEnd := len(body)
	if next := strings.Index(body[start+len(heading):], "\n## "); next >= 0 {
		sectionEnd = start + len(heading) + next + 1
	}
	addition := "\n" + link + "\n"
	return body[:sectionEnd] + addition + body[sectionEnd:]
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
	type location struct {
		path    string
		storage string
	}
	byID := make(map[string]location)
	for _, directory := range []struct {
		path    string
		storage string
	}{{store.Workspace.Histories, "open"}, {store.Workspace.Database, "closed"}} {
		entries, err := os.ReadDir(directory.path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			match := topicFolderPattern.FindStringSubmatch(entry.Name())
			if len(match) != 3 {
				continue
			}
			path := filepath.Join(directory.path, entry.Name())
			info, err := os.Lstat(path)
			if err != nil {
				return nil, err
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return nil, fmt.Errorf("%w: topic symlink %s", domain.ErrConflict, store.Workspace.RelativePath(path))
			}
			if !info.IsDir() {
				continue
			}
			if previous, exists := byID[match[1]]; exists {
				if previous.storage == directory.storage {
					return nil, fmt.Errorf("%w: duplicate %s topic ID %s", domain.ErrConflict, directory.storage, match[1])
				}
				if previous.storage == "open" {
					continue
				}
			}
			byID[match[1]] = location{path: path, storage: directory.storage}
		}
	}
	result := make([]string, 0, len(byID))
	for _, topic := range byID {
		result = append(result, topic.path)
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
