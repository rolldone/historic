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
// Managed Markdown frontmatter remains authoritative for member status.
type TopicMetadata struct {
	ID          domain.ID       `yaml:"id"`
	Title       string          `yaml:"title"`
	Description string          `yaml:"description,omitempty"`
	Created     string          `yaml:"created"`
	Updated     string          `yaml:"updated,omitempty"`
	Tags        []string        `yaml:"tags,omitempty"`
	Related     []domain.ID     `yaml:"related,omitempty"`
	Files       []ManifestFile  `yaml:"files"`
	Assets      []ManifestAsset `yaml:"assets"`
}

// ManifestFile describes a managed Markdown file discovered below a topic.
type ManifestFile struct {
	ID     domain.FileID `yaml:"id"`
	Path   string        `yaml:"path"`
	Type   string        `yaml:"type"`
	Status domain.Status `yaml:"status"`
}

// ManifestAsset describes a non-managed file discovered below a topic.
type ManifestAsset struct {
	Path string `yaml:"path"`
	Type string `yaml:"type"`
}

// TopicMetadataFromFrontmatter converts legacy topic frontmatter without
// carrying generated Markdown sections into the YAML representation.
func TopicMetadataFromFrontmatter(frontmatter domain.Frontmatter, _ string) TopicMetadata {
	return TopicMetadata{
		ID: frontmatter.ID, Title: frontmatter.Title, Description: frontmatter.Description,
		Created: frontmatter.Created, Updated: frontmatter.Updated,
		Tags: append([]string(nil), frontmatter.Tags...), Related: append([]domain.ID(nil), frontmatter.Related...),
	}
}

// Frontmatter converts topic metadata to the legacy-compatible representation
// used by lifecycle responses and callers that still expose Status.
func (metadata TopicMetadata) Frontmatter() domain.Frontmatter {
	return domain.Frontmatter{
		ID: metadata.ID, Title: metadata.Title, Description: metadata.Description,
		Created: metadata.Created, Updated: metadata.Updated,
		Tags: append([]string(nil), metadata.Tags...), Related: append([]domain.ID(nil), metadata.Related...),
	}
}

// ValidateTopicMetadata validates the canonical YAML fields.
func ValidateTopicMetadata(metadata TopicMetadata) error {
	if !metadata.ID.Valid() {
		return fmt.Errorf("%w: field %q must be a valid topic ID", ErrInvalidFrontmatter, "id")
	}
	if strings.TrimSpace(metadata.Title) == "" {
		return fmt.Errorf("%w: field %q is required", ErrInvalidFrontmatter, "title")
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
		if !validRelatedReference(string(related)) {
			return fmt.Errorf("%w: field %q item %d must be a topic ID or relative path", ErrInvalidFrontmatter, "related", index)
		}
	}
	if err := validateManifestFiles(metadata.Files); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidFrontmatter, err)
	}
	if err := validateManifestAssets(metadata.Assets); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidFrontmatter, err)
	}
	return nil
}

func validateManifestFiles(files []ManifestFile) error {
	seenPaths := make(map[string]struct{}, len(files))
	seenIDs := make(map[domain.FileID]struct{}, len(files))
	for index, file := range files {
		if !file.ID.Valid() {
			return fmt.Errorf("field %q item %d has invalid file ID %q", "files", index, file.ID)
		}
		if _, exists := seenIDs[file.ID]; exists {
			return fmt.Errorf("field %q contains duplicate file ID %q", "files", file.ID)
		}
		seenIDs[file.ID] = struct{}{}
		if err := validateManifestPath(file.Path); err != nil {
			return fmt.Errorf("field %q item %d: %v", "files", index, err)
		}
		if strings.TrimSpace(file.Type) == "" {
			return fmt.Errorf("field %q item %d type is required", "files", index)
		}
		if !file.Status.IsValid() {
			return fmt.Errorf("field %q item %d has invalid status %q", "files", index, file.Status)
		}
		if _, exists := seenPaths[file.Path]; exists {
			return fmt.Errorf("field %q contains duplicate path %q", "files", file.Path)
		}
		seenPaths[file.Path] = struct{}{}
	}
	return nil
}

// ParseTopicMetadataLenient parses _meta.yaml without requiring valid file IDs.
// Used by the FileID migration command to read legacy metadata.
func ParseTopicMetadataLenient(path string, input []byte) (TopicMetadata, error) {
	if !utf8Valid(input) {
		return TopicMetadata{}, documentError(path, "metadata", errors.New("file is not valid UTF-8"))
	}
	var metadata TopicMetadata
	decoder := yaml.NewDecoder(bytes.NewReader(input))
	decoder.KnownFields(false)
	if err := decoder.Decode(&metadata); err != nil {
		return TopicMetadata{}, documentError(path, "metadata", fmt.Errorf("%w: %v", ErrInvalidFrontmatter, err))
	}
	if err := ValidateTopicMetadataLenient(metadata); err != nil {
		return TopicMetadata{}, documentError(path, "metadata", err)
	}
	return metadata, nil
}

// ValidateTopicMetadataLenient validates canonical YAML fields without requiring
// valid file IDs. File path, type, and status are still validated.
func ValidateTopicMetadataLenient(metadata TopicMetadata) error {
	if !metadata.ID.Valid() {
		return fmt.Errorf("%w: field %q must be a valid topic ID", ErrInvalidFrontmatter, "id")
	}
	if strings.TrimSpace(metadata.Title) == "" {
		return fmt.Errorf("%w: field %q is required", ErrInvalidFrontmatter, "title")
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
		if !validRelatedReference(string(related)) {
			return fmt.Errorf("%w: field %q item %d must be a topic ID or relative path", ErrInvalidFrontmatter, "related", index)
		}
	}
	if err := validateManifestFilesLenient(metadata.Files); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidFrontmatter, err)
	}
	if err := validateManifestAssets(metadata.Assets); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidFrontmatter, err)
	}
	return nil
}

func validateManifestFilesLenient(files []ManifestFile) error {
	seenPaths := make(map[string]struct{}, len(files))
	for index, file := range files {
		if err := validateManifestPath(file.Path); err != nil {
			return fmt.Errorf("field %q item %d: %v", "files", index, err)
		}
		if strings.TrimSpace(file.Type) == "" {
			return fmt.Errorf("field %q item %d type is required", "files", index)
		}
		if !file.Status.IsValid() {
			return fmt.Errorf("field %q item %d has invalid status %q", "files", index, file.Status)
		}
		if _, exists := seenPaths[file.Path]; exists {
			return fmt.Errorf("field %q contains duplicate path %q", "files", file.Path)
		}
		seenPaths[file.Path] = struct{}{}
	}
	return nil
}

func validateManifestAssets(assets []ManifestAsset) error {
	seen := make(map[string]struct{}, len(assets))
	for index, asset := range assets {
		if err := validateManifestPath(asset.Path); err != nil {
			return fmt.Errorf("field %q item %d: %v", "assets", index, err)
		}
		if strings.TrimSpace(asset.Type) == "" {
			return fmt.Errorf("field %q item %d type is required", "assets", index)
		}
		if _, exists := seen[asset.Path]; exists {
			return fmt.Errorf("field %q contains duplicate path %q", "assets", asset.Path)
		}
		seen[asset.Path] = struct{}{}
	}
	return nil
}

func validateManifestPath(value string) error {
	if value == "" || filepath.IsAbs(value) || filepath.ToSlash(value) != value || strings.HasPrefix(value, "/") {
		return fmt.Errorf("path %q must be a logical POSIX relative path", value)
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("path %q must be a logical POSIX relative path", value)
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
	decoder.KnownFields(false)
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
	if err := writeTopicMetadata(path, metadata); err != nil {
		return err
	}
	return nil
}

// RestoreTopicMetadata restores previously captured canonical metadata bytes
// atomically after validating them. It is used by rebuild rollback.
func RestoreTopicMetadata(path string, input []byte) error {
	if _, err := ParseTopicMetadata(path, input); err != nil {
		return err
	}
	return atomicWrite(path, input)
}

// WriteTopicMetadataManifest writes metadata only when the filesystem-derived
// manifest differs from the current canonical metadata. The rendered result
// is validated before and after the atomic replacement.
func WriteTopicMetadataManifest(path string, metadata TopicMetadata, files []ManifestFile, assets []ManifestAsset) (bool, error) {
	if manifestsEqual(metadata.Files, files) && assetsEqual(metadata.Assets, assets) {
		return false, nil
	}
	metadata.Files = append([]ManifestFile(nil), files...)
	metadata.Assets = append([]ManifestAsset(nil), assets...)
	if err := writeTopicMetadata(path, metadata); err != nil {
		return false, err
	}
	return true, nil
}

func writeTopicMetadata(path string, metadata TopicMetadata) error {
	if err := ValidateTopicMetadata(metadata); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	data, err := yaml.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("%s: marshal metadata: %w", path, err)
	}
	if _, err := ParseTopicMetadata(path, data); err != nil {
		return fmt.Errorf("%s: validate metadata before write: %w", path, err)
	}
	if err := atomicWrite(path, data); err != nil {
		return err
	}
	if _, err := ParseTopicMetadataFile(path); err != nil {
		return fmt.Errorf("%s: validate metadata after write: %w", path, err)
	}
	return nil
}

func manifestsEqual(left, right []ManifestFile) bool {
	if (left == nil) != (right == nil) || len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func assetsEqual(left, right []ManifestAsset) bool {
	if (left == nil) != (right == nil) || len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
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
