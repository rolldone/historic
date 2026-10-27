package repository

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"historic/internal/markdown"
)

func withFrozenNow(t *testing.T, value time.Time) {
	t.Helper()
	previous := timeNow
	timeNow = func() time.Time { return value }
	t.Cleanup(func() { timeNow = previous })
}

func TestCreateTopicUseUTCDateRegardlessOfHostTimezone(t *testing.T) {
	store := newTestStore(t)
	// 2026-10-28 03:00 in UTC+7 is still 2026-10-27 in UTC.
	withFrozenNow(t, time.Date(2026, 10, 28, 3, 0, 0, 0, time.FixedZone("WIB", 7*60*60)))

	topic, err := store.CreateTopic("UTC Topic", "00016", "")
	if err != nil {
		t.Fatal(err)
	}
	if got := topic.Created.Format("2006-01-02"); got != "2026-10-27" {
		t.Fatalf("topic.Created = %q, want 2026-10-27", got)
	}
	metadata, err := markdown.ParseTopicMetadataFile(filepath.Join(topic.Path, markdown.MetaFilename))
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Created != "2026-10-27" {
		t.Fatalf("metadata created = %q, want 2026-10-27", metadata.Created)
	}
}

func TestAddEntryUsesUTCDateRegardlessOfHostTimezone(t *testing.T) {
	store := newTestStore(t)
	topic, err := store.CreateTopic("UTC Topic", "00017", "")
	if err != nil {
		t.Fatal(err)
	}
	withFrozenNow(t, time.Date(2026, 10, 28, 3, 0, 0, 0, time.FixedZone("WIB", 7*60*60)))

	entry, err := store.AddEntry(topic.ID, "note", false)
	if err != nil {
		t.Fatal(err)
	}
	if got := entry.Created.Format("2006-01-02"); got != "2026-10-27" {
		t.Fatalf("entry.Created = %q, want 2026-10-27", got)
	}
	document, err := markdown.ParseFile(entry.Path)
	if err != nil {
		t.Fatal(err)
	}
	if document.Frontmatter.Created != "2026-10-27" {
		t.Fatalf("frontmatter created = %q, want 2026-10-27", document.Frontmatter.Created)
	}
	meta, err := markdown.ReadTopicMetadata(topic.Path)
	if err != nil {
		t.Fatal(err)
	}
	if len(meta.Files) != 1 || meta.Files[0].Status.String() != "create" {
		t.Fatalf("manifest = %#v", meta.Files)
	}
	if _, err := os.Stat(entry.Path); err != nil {
		t.Fatal(err)
	}
}
