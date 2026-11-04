package lifecycle

import (
	"fmt"
	"os"
	"strings"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"
)

// FileRead is a read-only snapshot of one managed Markdown file. Raw keeps the
// exact bytes on disk so human output can stay byte-faithful; Document is the
// parsed view used by JSON callers.
type FileRead struct {
	Path     string
	Storage  domain.StorageState
	Raw      string
	Document markdown.Document
	Head     int
}

// ReadFile resolves and reads one managed Markdown file from open or closed
// storage. It never writes, rebuilds the index, or triggers auto-commit.
// A head value greater than zero limits the returned content to that many
// lines; zero returns the whole file.
func ReadFile(workspace config.Workspace, inputPath string, head int) (FileRead, error) {
	if head < 0 {
		return FileRead{}, fmt.Errorf("%w: --head must be zero or greater", domain.ErrConflict)
	}
	path, err := resolveMarkdownPath(workspace, inputPath)
	if err != nil {
		return FileRead{}, err
	}
	document, err := markdown.ParseFile(path)
	if err != nil {
		return FileRead{}, fmt.Errorf("read unsupported: target is not valid Historic Markdown: %w", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return FileRead{}, fmt.Errorf("read %s: %w", workspace.RelativePath(path), err)
	}
	storage := domain.StorageOpen
	if isInside(workspace.Database, path) {
		storage = domain.StorageClosed
	}
	content := string(raw)
	if head > 0 {
		content = headLines(content, head)
		document.Body = headLines(document.Body, head)
	}
	return FileRead{Path: workspace.RelativePath(path), Storage: storage, Raw: content, Document: document, Head: head}, nil
}

// headLines keeps the first limit lines, preserving the trailing newline of
// the source file when the limit falls on a line boundary.
func headLines(content string, limit int) string {
	if limit <= 0 || content == "" {
		return content
	}
	lines := strings.Split(content, "\n")
	if len(lines) <= limit {
		return content
	}
	return strings.Join(lines[:limit], "\n") + "\n"
}
