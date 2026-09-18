package markdown

import (
	"fmt"
	"path/filepath"
	"strings"

	"historic/internal/domain"
)

// NewDocument constructs a validated Markdown document from metadata and body.
func NewDocument(metadata domain.Frontmatter, body string) (Document, error) {
	document := Document{Frontmatter: metadata, Body: body}
	if err := ValidateFrontmatter(metadata); err != nil {
		return Document{}, err
	}
	return document, nil
}

// NewMetaDocument constructs the canonical topic metadata document.
func NewMetaDocument(metadata domain.Frontmatter, description string, files []string, progress string) (Document, error) {
	if err := ValidateFrontmatter(metadata); err != nil {
		return Document{}, err
	}
	var body strings.Builder
	fmt.Fprintf(&body, "# %s\n\n", metadata.Title)
	body.WriteString("## Deskripsi\n\n")
	if description != "" {
		body.WriteString(strings.TrimRight(description, "\n"))
		body.WriteString("\n\n")
	}
	body.WriteString("## Files\n\n")
	for _, file := range files {
		file = filepath.ToSlash(strings.TrimSpace(file))
		if file == "" {
			continue
		}
		fmt.Fprintf(&body, "- [%s](./%s)\n", filepath.Base(file), file)
	}
	body.WriteString("\n## Progress\n\n")
	if progress != "" {
		body.WriteString(strings.TrimRight(progress, "\n"))
		body.WriteByte('\n')
	}
	return Document{Frontmatter: metadata, Body: body.String()}, nil
}

// WriteMetaFile writes the canonical topic metadata document atomically.
func WriteMetaFile(path string, metadata domain.Frontmatter, description string, files []string, progress string) error {
	document, err := NewMetaDocument(metadata, description, files, progress)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return WriteFile(path, document)
}
