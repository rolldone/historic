package markdown

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"historic/internal/domain"
)

func testFrontmatter() domain.Frontmatter {
	return domain.Frontmatter{
		ID: "00014", Title: "Admin Dashboard", Status: domain.StatusProgress,
		Created: "2026-09-16", Updated: "2026-09-18",
		Tags: []string{"frontend", "admin"}, Related: []domain.ID{"00015", "00021"},
	}
}

func TestParseAndWriteRoundTrip(t *testing.T) {
	original := Document{Frontmatter: testFrontmatter(), Body: "# Admin Dashboard\n\nProgress notes.\n"}
	encoded, err := Write(original)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	parsed, err := Parse("topic/prd.md", encoded)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if parsed.Frontmatter.ID != original.Frontmatter.ID || parsed.Frontmatter.Title != original.Frontmatter.Title {
		t.Fatalf("metadata changed: %#v", parsed.Frontmatter)
	}
	if parsed.Body != original.Body {
		t.Fatalf("body = %q, want %q", parsed.Body, original.Body)
	}
}

func TestParseAllowsEmptyOptionalFields(t *testing.T) {
	input := "---\nid: 00014\ntitle: Note\nstatus: create\ncreated: 2026-09-18\n---\nBody\n"
	document, err := Parse("note.md", []byte(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(document.Frontmatter.Tags) != 0 || len(document.Frontmatter.Related) != 0 {
		t.Fatalf("optional fields should be empty: %#v", document.Frontmatter)
	}
}

func TestParseReportsMissingFieldAndPath(t *testing.T) {
	input := "---\nid: 00014\ntitle: Note\ncreated: 2026-09-18\n---\n"
	_, err := Parse("topics/00014-note/prd.md", []byte(input))
	if err == nil || !strings.Contains(err.Error(), "topics/00014-note/prd.md") || !strings.Contains(err.Error(), "status") {
		t.Fatalf("error = %v, want path and status", err)
	}
}

func TestParseRejectsMissingFrontmatter(t *testing.T) {
	_, err := Parse("note.md", []byte("# no metadata\n"))
	if !errors.Is(err, ErrMissingFrontmatter) {
		t.Fatalf("error = %v, want ErrMissingFrontmatter", err)
	}
}

func TestWriteFileIsAtomicOnInvalidDocument(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	invalid := Document{Frontmatter: domain.Frontmatter{ID: "00014", Title: "", Status: domain.StatusProgress, Created: "2026-09-18"}}
	if err := WriteFile(path, invalid); err == nil {
		t.Fatal("WriteFile accepted invalid document")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "original\n" {
		t.Fatalf("source overwritten: %q", content)
	}
}

func TestWriteFileCreatesUTF8Document(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "note.md")
	document := Document{Frontmatter: testFrontmatter(), Body: "Catatan: café\n"}
	if err := WriteFile(path, document); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	parsed, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if !strings.Contains(parsed.Body, "café") {
		t.Fatalf("UTF-8 body lost: %q", parsed.Body)
	}
}

func TestNewMetaDocument(t *testing.T) {
	document, err := NewMetaDocument(testFrontmatter(), "Topic description.", []string{"prd.md", "wos/01-scaffold.md"}, "2/3 selesai")
	if err != nil {
		t.Fatalf("NewMetaDocument: %v", err)
	}
	if !strings.Contains(document.Body, "## Deskripsi") || !strings.Contains(document.Body, "./wos/01-scaffold.md") || !strings.Contains(document.Body, "2/3 selesai") {
		t.Fatalf("metadata body missing sections: %q", document.Body)
	}
}
