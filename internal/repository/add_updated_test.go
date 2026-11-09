package repository

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"historic/internal/domain"
	"historic/internal/lifecycle"
	"historic/internal/markdown"
)

func TestAddEntryStampsUpdatedEqualToCreated(t *testing.T) {
	store := newTestStore(t)
	topic, err := store.CreateTopic("Fresh", "00041", "")
	if err != nil {
		t.Fatal(err)
	}
	// 2026-10-28 03:00 in UTC+7 is still 2026-10-27 in UTC.
	withFrozenNow(t, time.Date(2026, 10, 28, 3, 0, 0, 0, time.FixedZone("WIB", 7*60*60)))

	entry, err := store.AddEntry(topic.ID, "note", false)
	if err != nil {
		t.Fatal(err)
	}
	document, err := markdown.ParseFile(entry.Path)
	if err != nil {
		t.Fatal(err)
	}
	if document.Frontmatter.Created != "2026-10-27" {
		t.Fatalf("created = %q, want 2026-10-27", document.Frontmatter.Created)
	}
	if document.Frontmatter.Updated != document.Frontmatter.Created {
		t.Fatalf("updated = %q, want it to equal created %q", document.Frontmatter.Updated, document.Frontmatter.Created)
	}
	if document.Frontmatter.Updated < document.Frontmatter.Created {
		t.Fatalf("updated %q precedes created %q", document.Frontmatter.Updated, document.Frontmatter.Created)
	}
}

func TestAddEntryForceRestampsUpdatedToToday(t *testing.T) {
	store := newTestStore(t)
	topic, err := store.CreateTopic("Restamp", "00042", "")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(topic.Path, "note.md")
	stale := "---\ntitle: stale\nstatus: complete\ncreated: 2026-01-01\nupdated: 2026-01-01\n---\nold body\n"
	if err := os.WriteFile(path, []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddEntry(topic.ID, "note", true); err != nil {
		t.Fatalf("add --force: %v", err)
	}
	document, err := markdown.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	today := time.Now().UTC().Format("2006-01-02")
	if document.Frontmatter.Updated != today {
		t.Fatalf("updated = %q, want %q after --force", document.Frontmatter.Updated, today)
	}
	if document.Frontmatter.Updated != document.Frontmatter.Created {
		t.Fatalf("updated %q != created %q after --force", document.Frontmatter.Updated, document.Frontmatter.Created)
	}
}

// TestAddEntryNeedsNoBumpPass proves the design decision in SPEC §8.1: because
// the new file is stamped at write time and indexed by AddEntry itself, the
// hash-diff bump pass has nothing to detect and must not rewrite the file.
func TestAddEntryNeedsNoBumpPass(t *testing.T) {
	store := newTestStore(t)
	topic, err := store.CreateTopic("No bump", "00043", "")
	if err != nil {
		t.Fatal(err)
	}
	withFrozenNow(t, time.Now().AddDate(0, 0, -30))

	entry, err := store.AddEntry(topic.ID, "note", false)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(entry.Path)
	if err != nil {
		t.Fatal(err)
	}
	// The bump pass compares against the hash AddEntry itself indexed; with an
	// untouched file it must report zero bumps and leave the bytes identical.
	result := lifecycle.BumpChangedFiles(store.Workspace, []domain.ID{topic.ID})
	if result.Count() != 0 {
		t.Fatalf("bumped = %v, want 0 for a freshly added file", result.BumpedFiles)
	}
	after, err := os.ReadFile(entry.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatalf("freshly added file was rewritten by the bump pass")
	}
}
