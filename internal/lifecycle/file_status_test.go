package lifecycle

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/indexer"
	"historic/internal/markdown"
)

func TestChangeFileStatusUpdatesOnlyTargetAndRebuilds(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00001-topic")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	metadata := domain.Frontmatter{ID: "00001", Title: "Topic", Status: domain.StatusProgress, Created: "2026-09-19"}
	if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename), markdown.TopicMetadataFromFrontmatter(metadata, "")); err != nil {
		t.Fatal(err)
	}
	target, err := markdown.NewDocument(metadata, "target body")
	if err != nil {
		t.Fatal(err)
	}
	other, err := markdown.NewDocument(metadata, "other body")
	if err != nil {
		t.Fatal(err)
	}
	targetPath := filepath.Join(topic, "19-fts5-index.md")
	otherPath := filepath.Join(topic, "20-query.md")
	if err := markdown.WriteFile(targetPath, target); err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(otherPath, other); err != nil {
		t.Fatal(err)
	}
	beforeOther, err := os.ReadFile(otherPath)
	if err != nil {
		t.Fatal(err)
	}

	change, err := ChangeFileStatus(workspace, "19-fts5-index.md", domain.StatusComplete)
	if err != nil {
		t.Fatal(err)
	}
	if change.Previous != domain.StatusProgress || change.Current != domain.StatusComplete || change.Path != ".historic/00001-topic/19-fts5-index.md" {
		t.Fatalf("change = %#v", change)
	}
	if change.Updated == "" {
		t.Fatal("updated date is empty")
	}
	updated, err := markdown.ParseFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Frontmatter.Status != domain.StatusComplete || updated.Frontmatter.Updated != change.Updated {
		t.Fatalf("updated frontmatter = %#v", updated.Frontmatter)
	}
	afterOther, err := os.ReadFile(otherPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(afterOther) != string(beforeOther) {
		t.Fatal("non-target file changed")
	}
	if indexed, err := indexer.Count(workspace); err != nil || indexed != 3 {
		t.Fatalf("index count = %d, %v", indexed, err)
	}
}

func TestChangeFileStatusSyncsManifestWithoutRebuild(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00002-topic")
	sub := filepath.Join(topic, "wos")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	metadata := markdown.TopicMetadataFromFrontmatter(
		domain.Frontmatter{ID: "00002", Title: "Topic", Status: domain.StatusProgress, Created: "2026-09-19"}, "")
	metadata.Tags = []string{"lifecycle"}
	if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename), metadata); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(topic, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(topic, "assets", "logo.png"), []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	document, err := markdown.NewDocument(
		domain.Frontmatter{ID: "00002", Title: "Work Order", Status: domain.StatusProgress, Created: "2026-09-19"}, "body")
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(sub, "01-work-order.md")
	if err := markdown.WriteFile(target, document); err != nil {
		t.Fatal(err)
	}

	change, err := ChangeFileStatus(workspace, "wos/01-work-order.md", domain.StatusDraft)
	if err != nil {
		t.Fatal(err)
	}
	if change.Current != domain.StatusDraft {
		t.Fatalf("change = %#v", change)
	}
	synced, err := markdown.ReadTopicMetadata(topic)
	if err != nil {
		t.Fatal(err)
	}
	if len(synced.Files) != 1 {
		t.Fatalf("manifest files = %#v", synced.Files)
	}
	entry := synced.Files[0]
	if entry.Path != "wos/01-work-order.md" || entry.Status != domain.StatusDraft {
		t.Fatalf("manifest entry = %#v", entry)
	}
	if entry.Type != markdown.FileType(entry.Path) || entry.Type != "task" {
		t.Fatalf("manifest type = %q, want task", entry.Type)
	}
	if len(synced.Assets) != 1 || synced.Assets[0].Path != "assets/logo.png" || synced.Assets[0].Type != "image" {
		t.Fatalf("manifest assets = %#v", synced.Assets)
	}
	if synced.ID != "00002" || synced.Title != "Topic" || synced.Created != "2026-09-19" || len(synced.Tags) != 1 {
		t.Fatalf("metadata mutated = %#v", synced)
	}

	metaPath := filepath.Join(topic, markdown.MetaFilename)
	first, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ChangeFileStatus(workspace, "wos/01-work-order.md", domain.StatusDraft); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("repeated status change rewrote an unchanged manifest")
	}
}

func TestChangeFileStatusSyncsManifestForClosedTopic(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Database, "00003-closed")
	sub := filepath.Join(topic, "wos")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	metadata := markdown.TopicMetadataFromFrontmatter(
		domain.Frontmatter{ID: "00003", Title: "Closed", Status: domain.StatusProgress, Created: "2026-09-19"}, "")
	if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename), metadata); err != nil {
		t.Fatal(err)
	}
	document, err := markdown.NewDocument(
		domain.Frontmatter{ID: "00003", Title: "Task", Status: domain.StatusProgress, Created: "2026-09-19"}, "body")
	if err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(filepath.Join(sub, "02-task.md"), document); err != nil {
		t.Fatal(err)
	}

	relative := ".historic/.database/00003-closed/wos/02-task.md"
	if _, err := ChangeFileStatus(workspace, relative, domain.StatusReview); err != nil {
		t.Fatal(err)
	}
	synced, err := markdown.ReadTopicMetadata(topic)
	if err != nil {
		t.Fatal(err)
	}
	if len(synced.Files) != 1 || synced.Files[0].Status != domain.StatusReview {
		t.Fatalf("closed manifest = %#v", synced.Files)
	}
}

func TestChangeFileStatusReportsManifestFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission-based failure simulation requires a non-root user")
	}
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00004-topic")
	sub := filepath.Join(topic, "wos")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	metadata := markdown.TopicMetadataFromFrontmatter(
		domain.Frontmatter{ID: "00004", Title: "Topic", Status: domain.StatusProgress, Created: "2026-09-19"}, "")
	if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename), metadata); err != nil {
		t.Fatal(err)
	}
	document, err := markdown.NewDocument(
		domain.Frontmatter{ID: "00004", Title: "Task", Status: domain.StatusProgress, Created: "2026-09-19"}, "body")
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(sub, "01-task.md")
	if err := markdown.WriteFile(target, document); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(topic, 0o555); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(topic, 0o755) }()

	_, err = ChangeFileStatus(workspace, "wos/01-task.md", domain.StatusDraft)
	if err == nil || !strings.Contains(err.Error(), "manifest sync failed") || !strings.Contains(err.Error(), "historic rebuild") {
		t.Fatalf("manifest failure error = %v", err)
	}
	updated, err := markdown.ParseFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Frontmatter.Status != domain.StatusDraft {
		t.Fatalf("frontmatter not applied: %#v", updated.Frontmatter)
	}
	if indexed, err := indexer.Count(workspace); err != nil || indexed != 2 {
		t.Fatalf("index count = %d, %v; index must stay consistent with Markdown", indexed, err)
	}
}

func TestChangeFileStatusAmbiguousOpenAndClosedCopies(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	metadata := markdown.TopicMetadataFromFrontmatter(
		domain.Frontmatter{ID: "00005", Title: "Dual", Status: domain.StatusProgress, Created: "2026-09-19"}, "")
	document, err := markdown.NewDocument(
		domain.Frontmatter{ID: "00005", Title: "Task", Status: domain.StatusProgress, Created: "2026-09-19"}, "body")
	if err != nil {
		t.Fatal(err)
	}
	for _, topic := range []string{
		filepath.Join(workspace.Histories, "00005-dual"),
		filepath.Join(workspace.Database, "00005-dual"),
	} {
		if err := os.MkdirAll(filepath.Join(topic, "wos"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename), metadata); err != nil {
			t.Fatal(err)
		}
		if err := markdown.WriteFile(filepath.Join(topic, "wos", "01-task.md"), document); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := ChangeFileStatus(workspace, "wos/01-task.md", domain.StatusComplete); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("ambiguous path error = %v, want ErrConflict", err)
	}
	closed := ".historic/.database/00005-dual/wos/01-task.md"
	if _, err := ChangeFileStatus(workspace, closed, domain.StatusReview); err != nil {
		t.Fatalf("explicit closed path rejected: %v", err)
	}
	synced, err := markdown.ReadTopicMetadata(filepath.Join(workspace.Database, "00005-dual"))
	if err != nil {
		t.Fatal(err)
	}
	if len(synced.Files) != 1 || synced.Files[0].Status != domain.StatusReview {
		t.Fatalf("closed manifest = %#v", synced.Files)
	}
}

func TestChangeFileStatusClearsForwardedTo(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "1792889892200-child")
	sub := filepath.Join(topic, "wos")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename), markdown.TopicMetadata{
		ID: "1792889892200", Title: "Child", Created: "2026-10-25", Extends: "1791851149892",
	}); err != nil {
		t.Fatal(err)
	}
	document, err := markdown.NewDocument(domain.Frontmatter{
		ID: "1792889892200", Title: "Task", Status: domain.StatusForwarded, Created: "2026-10-25",
		ForwardedTo: "1792889892200",
	}, "body")
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(sub, "01-task.md")
	if err := markdown.WriteFile(target, document); err != nil {
		t.Fatal(err)
	}
	if _, err := ChangeFileStatus(workspace, "wos/01-task.md", domain.StatusForwarded); err != nil {
		t.Fatal(err)
	}
	before, err := markdown.ReadTopicMetadata(topic)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Files) != 1 || before.Files[0].ForwardedTo != "1792889892200" {
		t.Fatalf("manifest forwarded_to = %#v", before.Files)
	}

	if _, err := ChangeFileStatus(workspace, "wos/01-task.md", domain.StatusProgress); err != nil {
		t.Fatal(err)
	}
	updated, err := markdown.ParseFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Frontmatter.Status != domain.StatusProgress || updated.Frontmatter.ForwardedTo != "" {
		t.Fatalf("frontmatter after unforward = %#v", updated.Frontmatter)
	}
	after, err := markdown.ReadTopicMetadata(topic)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Files) != 1 || after.Files[0].ForwardedTo != "" || after.Files[0].Status != domain.StatusProgress {
		t.Fatalf("manifest after unforward = %#v", after.Files)
	}
}

func TestChangeFileStatusRejectsUnsafeAndInvalidTargets(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"../escape.md", "/tmp/absolute.md", "missing.txt"} {
		if _, err := ChangeFileStatus(workspace, path, domain.StatusComplete); err == nil {
			t.Fatalf("unsafe target %q accepted", path)
		}
	}
	topic := filepath.Join(workspace.Histories, "00001-topic")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(topic, "broken.md"), []byte("not frontmatter"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ChangeFileStatus(workspace, "broken.md", domain.StatusComplete); err == nil || !strings.Contains(err.Error(), "frontmatter") {
		t.Fatalf("invalid Markdown error = %v", err)
	}
}
