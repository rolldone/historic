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

	"github.com/yuin/goldmark"
	"gopkg.in/yaml.v3"
)

var (
	ErrMissingFrontmatter = errors.New("missing frontmatter")
	ErrInvalidFrontmatter = errors.New("invalid frontmatter")
)

const dateLayout = "2006-01-02"

// Document is a Markdown document with validated frontmatter and body.
type Document struct {
	Frontmatter domain.Frontmatter
	Body        string
}

// LooksLikeHistoricFile reports whether the Markdown bytes contain frontmatter
// with at least one canonical Historic field. It does not validate values.
func LooksLikeHistoricFile(input []byte) bool {
	frontmatter, _, err := splitFrontmatter(input)
	if err != nil {
		return false
	}
	text := string(frontmatter)
	return strings.Contains(text, "title:") || strings.Contains(text, "status:") || strings.Contains(text, "created:")
}

// Parse decodes a Markdown document from UTF-8 bytes.
func Parse(path string, input []byte) (Document, error) {
	if !utf8Valid(input) {
		return Document{}, documentError(path, "encoding", errors.New("file is not valid UTF-8"))
	}
	frontmatter, body, err := splitFrontmatter(input)
	if err != nil {
		return Document{}, documentError(path, "frontmatter", err)
	}

	var metadata domain.Frontmatter
	if err := validateDescriptionField(frontmatter); err != nil {
		return Document{}, documentError(path, "frontmatter", err)
	}
	if err := yaml.Unmarshal(frontmatter, &metadata); err != nil {
		return Document{}, documentError(path, "frontmatter", fmt.Errorf("%w: %v", ErrInvalidFrontmatter, err))
	}
	if err := ValidateFrontmatter(metadata); err != nil {
		return Document{}, documentError(path, "metadata", err)
	}
	return Document{Frontmatter: metadata, Body: body}, nil
}

// ParseFile reads and parses a Markdown document from disk.
func ParseFile(path string) (Document, error) {
	input, err := os.ReadFile(path)
	if err != nil {
		return Document{}, fmt.Errorf("%s: read: %w", path, err)
	}
	if filepath.Base(path) == MetaFilename {
		metadata, err := ParseTopicMetadata(path, input)
		if err != nil {
			return Document{}, err
		}
		return Document{Frontmatter: metadata.Frontmatter()}, nil
	}
	return Parse(path, input)
}

// ValidateFrontmatter checks all required and optional frontmatter fields.
func ValidateFrontmatter(metadata domain.Frontmatter) error {
	if metadata.ID != "" && !metadata.ID.Valid() {
		return fmt.Errorf("%w: field %q must be a valid topic ID when provided", ErrInvalidFrontmatter, "id")
	}
	if strings.TrimSpace(metadata.Title) == "" {
		return fmt.Errorf("%w: field %q is required", ErrInvalidFrontmatter, "title")
	}
	if !metadata.Status.IsValid() {
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
		if !validRelatedReference(string(related)) {
			return fmt.Errorf("%w: field %q item %d must be a five-digit ID or relative path", ErrInvalidFrontmatter, "related", index)
		}
	}
	return nil
}

func validRelatedReference(value string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}
	if domain.ID(value).Valid() {
		return true
	}
	if _, err := domain.ParseTopicID(value); err == nil {
		return true
	}
	if _, err := domain.ParseTopicID(value); err == nil {
		return true
	}
	if filepath.IsAbs(value) || strings.HasPrefix(value, "/") || strings.Contains(value, "\\") || (len(value) > 1 && value[1] == ':') {
		return false
	}
	parts := strings.Split(value, "/")
	hasTarget := false
	for _, part := range parts {
		if part == "" {
			return false
		}
		if part != "." && part != ".." {
			hasTarget = true
		}
	}
	return hasTarget
}

// Write serializes a document to UTF-8 Markdown bytes.
func Write(document Document) ([]byte, error) {
	if err := ValidateFrontmatter(document.Frontmatter); err != nil {
		return nil, err
	}
	metadata, err := yaml.Marshal(document.Frontmatter)
	if err != nil {
		return nil, fmt.Errorf("marshal frontmatter: %w", err)
	}
	body := strings.TrimPrefix(document.Body, "\n")
	var output bytes.Buffer
	output.WriteString("---\n")
	output.Write(metadata)
	output.WriteString("---\n")
	if body != "" {
		output.WriteString(body)
		if !strings.HasSuffix(body, "\n") {
			output.WriteByte('\n')
		}
	}
	return output.Bytes(), nil
}

// WriteFile writes a document atomically, preserving the destination on errors.
func WriteFile(path string, document Document) error {
	if filepath.Base(path) == MetaFilename {
		return WriteTopicMetadata(path, TopicMetadataFromFrontmatter(document.Frontmatter, ""))
	}
	data, err := Write(document)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("%s: create parent: %w", path, err)
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".historic-*.tmp")
	if err != nil {
		return fmt.Errorf("%s: create temporary file: %w", path, err)
	}
	temporary := file.Name()
	defer func() {
		_ = os.Remove(temporary)
	}()
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

// RenderMarkdown validates body syntax using Goldmark before writing it.
func RenderMarkdown(body string) (string, error) {
	var rendered bytes.Buffer
	if err := goldmark.New().Convert([]byte(body), &rendered); err != nil {
		return "", fmt.Errorf("parse Markdown: %w", err)
	}
	return rendered.String(), nil
}

func splitFrontmatter(input []byte) ([]byte, string, error) {
	text := string(input)
	if !strings.HasPrefix(text, "---\n") && !strings.HasPrefix(text, "---\r\n") {
		return nil, "", ErrMissingFrontmatter
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	end := strings.Index(text[4:], "\n---\n")
	if end < 0 {
		return nil, "", ErrInvalidFrontmatter
	}
	end += 4
	return []byte(text[4:end]), text[end+5:], nil
}

func validateDescriptionField(input []byte) error {
	var document yaml.Node
	if err := yaml.Unmarshal(input, &document); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidFrontmatter, err)
	}
	if len(document.Content) == 0 || document.Content[0].Kind != yaml.MappingNode {
		return nil
	}
	mapping := document.Content[0]
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		key, value := mapping.Content[index], mapping.Content[index+1]
		if key.Value != "description" {
			continue
		}
		if value.Kind != yaml.ScalarNode || (value.Tag != "!!str" && value.Tag != "!!null") {
			return fmt.Errorf("%w: field %q must be a string or null", ErrInvalidFrontmatter, "description")
		}
		return nil
	}
	return nil
}

func documentError(path, field string, err error) error {
	if path == "" {
		return fmt.Errorf("field %s: %w", field, err)
	}
	return fmt.Errorf("%s: field %s: %w", path, field, err)
}

func utf8Valid(input []byte) bool {
	return strings.ToValidUTF8(string(input), "") == string(input)
}
