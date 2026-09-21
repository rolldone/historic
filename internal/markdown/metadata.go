package markdown

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"historic/internal/domain"

	"gopkg.in/yaml.v3"
)

const (
	// MetaFilename is the canonical topic metadata filename.
	MetaFilename = "_meta.yaml"
)

// TopicMetadata is the canonical, non-Markdown metadata for a topic.
// Status is retained for lifecycle compatibility; child Markdown frontmatter
// remains authoritative for managed file status.
type TopicMetadata struct {
	ID          domain.ID     `yaml:"id"`
	Title       string        `yaml:"title"`
	Description string        `yaml:"description,omitempty"`
	Status      domain.Status `yaml:"status,omitempty"`
	Created     string        `yaml:"created"`
	Updated     string        `yaml:"updated,omitempty"`
	Tags        []string      `yaml:"tags,omitempty"`
	Related     []domain.ID   `yaml:"related,omitempty"`
}

// TopicMetadataFromFrontmatter converts legacy topic frontmatter without
// carrying generated Markdown sections into the YAML representation.
func TopicMetadataFromFrontmatter(frontmatter domain.Frontmatter, _ string) TopicMetadata {
	return TopicMetadata{
		ID: frontmatter.ID, Title: frontmatter.Title, Description: frontmatter.Description,
		Status: frontmatter.Status, Created: frontmatter.Created, Updated: frontmatter.Updated,
		Tags: append([]string(nil), frontmatter.Tags...), Related: append([]domain.ID(nil), frontmatter.Related...),
	}
}

// Frontmatter converts topic metadata to the legacy-compatible representation
// used by lifecycle responses and callers that still expose Status.
func (metadata TopicMetadata) Frontmatter() domain.Frontmatter {
	return domain.Frontmatter{
		ID: metadata.ID, Title: metadata.Title, Description: metadata.Description,
		Status: metadata.Status, Created: metadata.Created, Updated: metadata.Updated,
		Tags: append([]string(nil), metadata.Tags...), Related: append([]domain.ID(nil), metadata.Related...),
	}
}

// ValidateTopicMetadata validates the canonical YAML fields.
func ValidateTopicMetadata(metadata TopicMetadata) error {
	if !metadata.ID.Valid() {
		return fmt.Errorf("%w: field %q must be a five-digit ID", ErrInvalidFrontmatter, "id")
	}
	if strings.TrimSpace(metadata.Title) == "" {
		return fmt.Errorf("%w: field %q is required", ErrInvalidFrontmatter, "title")
	}
	if metadata.Status != "" && !metadata.Status.IsValid() {
		return fmt.Errorf("%w: field %q has invalid value %q", ErrInvalidFrontmatter, "status", metadata.Status)
	}
	if _, err := time.Parse(dateLayout, metadata.Created); err != nil {
		return fmt.Errorf("%w: field %q must use YYYY-MM-DD", ErrInvalidFrontmatter, "created")
	}
	if metadata.Updated != "" {
		if _, err := time.Parse(dateLayout, metadata.Updated); err != nil {
			return fmt.Errorf("%w: field %q must use YYYY-MM-DD", ErrInvalidFrontmatter, "updated")
		}
	}
	for index, related := range metadata.Related {
		if !related.Valid() {
			return fmt.Errorf("%w: field %q item %d is not a five-digit ID", ErrInvalidFrontmatter, "related", index)
		}
	}
	return nil
}

// ParseTopicMetadata parses canonical _meta.yaml bytes.
func ParseTopicMetadata(path string, input []byte) (TopicMetadata, error) {
	if !utf8Valid(input) {
		return TopicMetadata{}, documentError(path, "metadata", errors.New("file is not valid UTF-8"))
	}
	var metadata TopicMetadata
	decoder := yaml.NewDecoder(bytes.NewReader(input))
	decoder.KnownFields(true)
	if err := decoder.Decode(&metadata); err != nil {
		return TopicMetadata{}, documentError(path, "metadata", fmt.Errorf("%w: %v", ErrInvalidFrontmatter, err))
	}
	if err := ValidateTopicMetadata(metadata); err != nil {
		return TopicMetadata{}, documentError(path, "metadata", err)
	}
	return metadata, nil
}

// ParseTopicMetadataFile reads and validates canonical topic metadata.
func ParseTopicMetadataFile(path string) (TopicMetadata, error) {
	input, err := os.ReadFile(path)
	if err != nil {
		return TopicMetadata{}, fmt.Errorf("%s: read: %w", path, err)
	}
	return ParseTopicMetadata(path, input)
}

// WriteTopicMetadata writes canonical topic metadata atomically and validates
// the installed bytes before returning.
func WriteTopicMetadata(path string, metadata TopicMetadata) error {
	if err := ValidateTopicMetadata(metadata); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	data, err := yaml.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("%s: marshal metadata: %w", path, err)
	}
	if err := atomicWrite(path, data); err != nil {
		return err
	}
	if _, err := ParseTopicMetadataFile(path); err != nil {
		return fmt.Errorf("%s: validate metadata after write: %w", path, err)
	}
	return nil
}

// ReadTopicMetadata reads and validates the canonical metadata file. The
// canonical file is required; Markdown files with other names are ordinary
// topic assets and are never considered metadata.
func ReadTopicMetadata(topicPath string) (TopicMetadata, error) {
	canonical := filepath.Join(topicPath, MetaFilename)
	info, err := os.Lstat(canonical)
	if err != nil {
		return TopicMetadata{}, fmt.Errorf("%s: read: %w", canonical, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return TopicMetadata{}, fmt.Errorf("%w: metadata symlink %s", domain.ErrConflict, canonical)
	}
	if info.IsDir() {
		return TopicMetadata{}, fmt.Errorf("%s: metadata is a directory", canonical)
	}
	if !info.Mode().IsRegular() {
		return TopicMetadata{}, fmt.Errorf("%w: metadata is not a regular file", domain.ErrConflict)
	}
	return ParseTopicMetadataFile(canonical)
}
func atomicWrite(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("%s: create parent: %w", path, err)
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".historic-meta-*.tmp")
	if err != nil {
		return fmt.Errorf("%s: create temporary file: %w", path, err)
	}
	temporary := file.Name()
	defer func() { _ = os.Remove(temporary) }()
	if err := file.Chmod(0o644); err == nil {
		_, err = file.Write(data)
	}
	if err == nil {
		err = file.Sync()
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("%s: write temporary file: %w", path, err)
	}
	if err := os.Rename(temporary, path); err != nil {
		return fmt.Errorf("%s: replace file: %w", path, err)
	}
	return nil
}
