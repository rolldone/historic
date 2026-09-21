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
	// LegacyMetaFilename is the Markdown topic metadata filename supported for migration.
	LegacyMetaFilename = "_meta.md"
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

// ReadTopicMetadata reads canonical metadata and migrates one legacy topic if
// needed. A topic containing both formats is rejected instead of overwritten.
func ReadTopicMetadata(topicPath string) (TopicMetadata, error) {
	canonical := filepath.Join(topicPath, MetaFilename)
	legacy := filepath.Join(topicPath, LegacyMetaFilename)
	canonicalInfo, canonicalErr := os.Stat(canonical)
	legacyInfo, legacyErr := os.Stat(legacy)
	if canonicalErr != nil && !errors.Is(canonicalErr, os.ErrNotExist) {
		return TopicMetadata{}, fmt.Errorf("inspect topic metadata %s: %w", canonical, canonicalErr)
	}
	if legacyErr != nil && !errors.Is(legacyErr, os.ErrNotExist) {
		return TopicMetadata{}, fmt.Errorf("inspect legacy metadata %s: %w", legacy, legacyErr)
	}
	if canonicalErr == nil && canonicalInfo.IsDir() {
		return TopicMetadata{}, fmt.Errorf("%s: metadata is a directory", canonical)
	}
	if legacyErr == nil && legacyInfo.IsDir() {
		return TopicMetadata{}, fmt.Errorf("%s: legacy metadata is a directory", legacy)
	}
	if canonicalErr == nil && legacyErr == nil {
		return TopicMetadata{}, fmt.Errorf("%w: both %s and %s exist", domain.ErrConflict, canonical, legacy)
	}
	if canonicalErr == nil {
		return ParseTopicMetadataFile(canonical)
	}
	if legacyErr == nil {
		metadata, err := migrateLegacyTopic(topicPath, legacy, canonical)
		if err != nil {
			return TopicMetadata{}, err
		}
		return metadata, nil
	}
	return TopicMetadata{}, fmt.Errorf("%s: read: %w", canonical, os.ErrNotExist)
}

// MigrateWorkspaceMetadata migrates all topic roots. The returned rollback
// function restores legacy files if a later rebuild/upgrade step fails.
func MigrateWorkspaceMetadata(roots ...string) (func(), error) {
	migrations := make([]metadataMigration, 0)
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("scan metadata root %s: %w", root, err)
		}
		for _, entry := range entries {
			if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			topicPath := filepath.Join(root, entry.Name())
			canonical := filepath.Join(topicPath, MetaFilename)
			legacy := filepath.Join(topicPath, LegacyMetaFilename)
			_, canonicalErr := os.Stat(canonical)
			_, legacyErr := os.Stat(legacy)
			if errors.Is(canonicalErr, os.ErrNotExist) && legacyErr == nil {
				old, err := os.ReadFile(legacy)
				if err != nil {
					rollbackMetadata(migrations)
					return nil, fmt.Errorf("save legacy metadata %s: %w", legacy, err)
				}
				if _, err := migrateLegacyTopic(topicPath, legacy, canonical); err != nil {
					rollbackMetadata(migrations)
					return nil, err
				}
				migrations = append(migrations, metadataMigration{canonical: canonical, legacy: legacy, legacyBytes: old})
				continue
			}
			if canonicalErr != nil && !errors.Is(canonicalErr, os.ErrNotExist) {
				rollbackMetadata(migrations)
				return nil, fmt.Errorf("inspect metadata %s: %w", canonical, canonicalErr)
			}
			if legacyErr != nil && !errors.Is(legacyErr, os.ErrNotExist) {
				rollbackMetadata(migrations)
				return nil, fmt.Errorf("inspect legacy metadata %s: %w", legacy, legacyErr)
			}
			if canonicalErr == nil && legacyErr == nil {
				rollbackMetadata(migrations)
				return nil, fmt.Errorf("%w: both %s and %s exist", domain.ErrConflict, canonical, legacy)
			}
		}
	}
	return func() { rollbackMetadata(migrations) }, nil
}

type metadataMigration struct {
	canonical   string
	legacy      string
	legacyBytes []byte
}

func migrateLegacyTopic(topicPath, legacy, canonical string) (TopicMetadata, error) {
	document, err := ParseFile(legacy)
	if err != nil {
		return TopicMetadata{}, fmt.Errorf("invalid legacy topic metadata %s: %w", legacy, err)
	}
	metadata := TopicMetadataFromFrontmatter(document.Frontmatter, "")
	if err := WriteTopicMetadata(canonical, metadata); err != nil {
		return TopicMetadata{}, fmt.Errorf("migrate legacy metadata %s: %w", legacy, err)
	}
	validated, err := ParseTopicMetadataFile(canonical)
	if err != nil {
		_ = os.Remove(canonical)
		return TopicMetadata{}, fmt.Errorf("validate migrated metadata %s: %w", canonical, err)
	}
	if err := os.Remove(legacy); err != nil {
		_ = os.Remove(canonical)
		return TopicMetadata{}, fmt.Errorf("remove legacy metadata %s after validation: %w", legacy, err)
	}
	return validated, nil
}

func rollbackMetadata(migrations []metadataMigration) {
	for index := len(migrations) - 1; index >= 0; index-- {
		migration := migrations[index]
		_ = os.Remove(migration.canonical)
		_ = os.WriteFile(migration.legacy, migration.legacyBytes, 0o644)
	}
}

func legacyDescription(body string) string {
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	for index, line := range lines {
		if strings.TrimSpace(line) != "## Deskripsi" {
			continue
		}
		end := index + 1
		for end < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[end]), "## ") {
			end++
		}
		return strings.TrimSpace(strings.Join(lines[index+1:end], "\n"))
	}
	return ""
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
