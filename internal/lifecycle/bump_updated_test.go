package lifecycle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/indexer"
	"historic/internal/markdown"
)

// bumpFixture builds one open topic with one managed task file and indexes it,
// returning the workspace, the topic folder, and the managed file path.
func bumpFixture(t *testing.T, topicID, created string) (config.Workspace, string, string) {
	t.Helper()
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, topicID+"-bump")
	if err := os.MkdirAll(filepath.Join(topic, "wos"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename),
		markdown.TopicMetadata{ID: domain.ID(topicID), Title: "Bump", Created: created}); err != nil {
		t.Fatal(err)
	}
	document, err := markdown.NewDocument(domain.Frontmatter{
		ID: domain.ID(topicID), Title: "Task", Status: domain.StatusProgress, Created: created,
	}, "original body")
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(topic, "wos", "01-task.md")
	if err := markdown.WriteFile(target, document); err != nil {
		t.Fatal(err)
	}
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	return workspace, topic, target
}

func writeBody(t *testing.T, path, body string) {
	t.Helper()
	document, err := markdown.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	document.Body = body
	if err := markdown.WriteFile(path, document); err != nil {
		t.Fatal(err)
	}
}

func TestBumpChangedFilesAdvancesUpdatedForChangedContent(t *testing.T) {
	workspace, _, target := bumpFixture(t, "00031", "2026-10-01")
	withFrozenStatusNow(t, time.Date(2026, 11, 9, 3, 0, 0, 0, time.UTC))

	writeBody(t, target, "edited body")

	result := BumpChangedFiles(workspace, []domain.ID{"00031"})
	if result.Count() != 1 {
		t.Fatalf("bumped = %v, want 1 file", result.BumpedFiles)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("warnings = %v, want none", result.Warnings)
	}
	document, err := markdown.ParseFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if document.Frontmatter.Updated != "2026-11-09" {
		t.Fatalf("updated = %q, want 2026-11-09", document.Frontmatter.Updated)
	}
	if document.Frontmatter.Updated < document.Frontmatter.Created {
		t.Fatalf("updated %q precedes created %q", document.Frontmatter.Updated, document.Frontmatter.Created)
	}
}

func TestBumpChangedFilesIsIdempotent(t *testing.T) {
	workspace, _, target := bumpFixture(t, "00032", "2026-10-01")
	withFrozenStatusNow(t, time.Date(2026, 11, 9, 3, 0, 0, 0, time.UTC))

	writeBody(t, target, "edited body")
	if result := BumpChangedFiles(workspace, []domain.ID{"00032"}); result.Count() != 1 {
		t.Fatalf("first pass bumped = %v, want 1", result.BumpedFiles)
	}
	before, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	second := BumpChangedFiles(workspace, []domain.ID{"00032"})
	if second.Count() != 0 {
		t.Fatalf("second pass bumped = %v, want 0 (idempotent)", second.BumpedFiles)
	}
	after, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatalf("second pass rewrote the file:\nbefore=%q\nafter=%q", before, after)
	}
}

func TestBumpChangedFilesNeverMovesUpdatedBeforeCreated(t *testing.T) {
	// created is ahead of the frozen UTC date: the created floor must win.
	workspace, _, target := bumpFixture(t, "00033", "2026-12-31")
	withFrozenStatusNow(t, time.Date(2026, 11, 9, 3, 0, 0, 0, time.UTC))

	writeBody(t, target, "edited body")

	result := BumpChangedFiles(workspace, []domain.ID{"00033"})
	if result.Count() != 1 {
		t.Fatalf("bumped = %v, want 1", result.BumpedFiles)
	}
	document, err := markdown.ParseFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if document.Frontmatter.Updated != "2026-12-31" {
		t.Fatalf("updated = %q, want created floor 2026-12-31", document.Frontmatter.Updated)
	}
}

func TestBumpChangedFilesSkipsAssetsAndNonHistoricMarkdown(t *testing.T) {
	workspace, topic, target := bumpFixture(t, "00034", "2026-10-01")
	withFrozenStatusNow(t, time.Date(2026, 11, 9, 3, 0, 0, 0, time.UTC))

	// A Historic asset (valid frontmatter but classified as an asset) and a
	// plain Markdown note without Historic frontmatter.
	assetPath := filepath.Join(topic, "diagram.png")
	if err := os.WriteFile(assetPath, []byte("not-a-real-png"), 0o644); err != nil {
		t.Fatal(err)
	}
	plainPath := filepath.Join(topic, "notes.md")
	if err := os.WriteFile(plainPath, []byte("# plain notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(assetPath, []byte("changed-asset-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(plainPath, []byte("# plain notes edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeBody(t, target, "edited body")

	result := BumpChangedFiles(workspace, []domain.ID{"00034"})
	if result.Count() != 1 || !strings.HasSuffix(result.BumpedFiles[0], "01-task.md") {
		t.Fatalf("bumped = %v, want only the managed task", result.BumpedFiles)
	}
	plain, err := os.ReadFile(plainPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(plain), "updated:") {
		t.Fatalf("non-Historic Markdown was rewritten: %q", plain)
	}
}

func TestBumpChangedFilesReportsPartialWarningButContinues(t *testing.T) {
	workspace, topic, target := bumpFixture(t, "00035", "2026-10-01")
	withFrozenStatusNow(t, time.Date(2026, 11, 9, 3, 0, 0, 0, time.UTC))

	// A broken Historic file: frontmatter markers but invalid YAML. It must be
	// reported as a warning without stopping the pass.
	broken := filepath.Join(topic, "broken.md")
	if err := os.WriteFile(broken, []byte("---\ntitle: broken\nstatus: [unclosed\n---\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := indexer.Rebuild(workspace); err != nil {
		// The indexer may reject invalid Historic Markdown; drop the file so the
		// fixture stays indexable, then recreate it after indexing.
		_ = os.Remove(broken)
		if _, err := indexer.Rebuild(workspace); err != nil {
			t.Fatal(err)
		}
	}
	writeBody(t, target, "edited body")
	if _, err := os.Stat(broken); err != nil {
		if err := os.WriteFile(broken, []byte("---\ntitle: broken\nstatus: [unclosed\n---\nbody\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	result := BumpChangedFiles(workspace, []domain.ID{"00035"})
	if result.Count() != 1 {
		t.Fatalf("bumped = %v, want the valid managed file", result.BumpedFiles)
	}
	found := false
	for _, warning := range result.Warnings {
		if strings.Contains(warning, "broken.md") {
			found = true
		}
	}
	if !found {
		t.Fatalf("warnings = %v, want one mentioning broken.md", result.Warnings)
	}
}

func TestBumpChangedFilesNeverWritesClosedMirror(t *testing.T) {
	workspace, _, _ := bumpFixture(t, "00036", "2026-10-01")
	withFrozenStatusNow(t, time.Date(2026, 11, 9, 3, 0, 0, 0, time.UTC))

	closedTopic := filepath.Join(workspace.Database, "00037-closed")
	if err := os.MkdirAll(filepath.Join(closedTopic, "wos"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteTopicMetadata(filepath.Join(closedTopic, markdown.MetaFilename),
		markdown.TopicMetadata{ID: "00037", Title: "Closed", Created: "2026-10-01"}); err != nil {
		t.Fatal(err)
	}
	document, err := markdown.NewDocument(domain.Frontmatter{
		ID: "00037", Title: "Closed task", Status: domain.StatusComplete, Created: "2026-10-01",
	}, "closed body")
	if err != nil {
		t.Fatal(err)
	}
	closedFile := filepath.Join(closedTopic, "wos", "01-closed.md")
	if err := markdown.WriteFile(closedFile, document); err != nil {
		t.Fatal(err)
	}
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(closedFile)
	if err != nil {
		t.Fatal(err)
	}
	writeBody(t, closedFile, "edited closed mirror")

	result := BumpChangedFiles(workspace, []domain.ID{"00037"})
	if result.Count() != 0 {
		t.Fatalf("bumped = %v, want no writes to the closed mirror", result.BumpedFiles)
	}
	after, err := os.ReadFile(closedFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) == string(after) {
		t.Fatalf("test setup did not change the closed file")
	}
	if strings.Contains(string(after), "updated: 2026-11-09") {
		t.Fatalf("closed mirror gained a bumped updated date: %q", after)
	}
}

func TestBumpChangedFilesSkipsFilesWithoutIndexedBaseline(t *testing.T) {
	workspace, topic, _ := bumpFixture(t, "00038", "2026-10-01")
	withFrozenStatusNow(t, time.Date(2026, 11, 9, 3, 0, 0, 0, time.UTC))

	// A new file created after the last index rebuild has no stored hash; the
	// pass must not guess and rewrite it.
	newFile := filepath.Join(topic, "wos", "02-new.md")
	document, err := markdown.NewDocument(domain.Frontmatter{
		ID: "00038", Title: "New", Status: domain.StatusPending, Created: "2026-11-08",
	}, "brand new")
	if err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(newFile, document); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(newFile)
	if err != nil {
		t.Fatal(err)
	}

	result := BumpChangedFiles(workspace, []domain.ID{"00038"})
	if result.Count() != 0 {
		t.Fatalf("bumped = %v, want no baseline-less writes", result.BumpedFiles)
	}
	after, err := os.ReadFile(newFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatalf("unindexed file was rewritten")
	}
}

func TestBumpAllChangedFilesCoversEveryOpenTopic(t *testing.T) {
	workspace, _, first := bumpFixture(t, "00039", "2026-10-01")
	withFrozenStatusNow(t, time.Date(2026, 11, 9, 3, 0, 0, 0, time.UTC))

	secondTopic := filepath.Join(workspace.Histories, "00040-second")
	if err := os.MkdirAll(secondTopic, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteTopicMetadata(filepath.Join(secondTopic, markdown.MetaFilename),
		markdown.TopicMetadata{ID: "00040", Title: "Second", Created: "2026-10-01"}); err != nil {
		t.Fatal(err)
	}
	document, err := markdown.NewDocument(domain.Frontmatter{
		ID: "00040", Title: "Second task", Status: domain.StatusProgress, Created: "2026-10-01",
	}, "second body")
	if err != nil {
		t.Fatal(err)
	}
	second := filepath.Join(secondTopic, "02-task.md")
	if err := markdown.WriteFile(second, document); err != nil {
		t.Fatal(err)
	}
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}
	writeBody(t, first, "first edited")
	writeBody(t, second, "second edited")

	result := BumpAllChangedFiles(workspace)
	if result.Count() != 2 {
		t.Fatalf("bumped = %v, want both topics' files", result.BumpedFiles)
	}
}
