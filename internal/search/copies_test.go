package search

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/indexer"
	"historic/internal/markdown"
)

// dualFileIDs are canonical UUIDv7 values used so fixtures stay reproducible
// instead of depending on wall-clock entropy. Each fixture topic needs its own
// FileID: the read model scopes a FileID per topic, so reusing one would fan
// the file join out across every topic.
var dualFileIDs = []string{
	"01a1d218-a752-7f9c-8416-fa6044bd1ec9",
	"01a1d24a-8b31-77ae-bee5-33f3312b6072",
	"01a1d218-a938-7ec2-bb7a-72eaff1d2145",
	"01a1d24a-8d07-7461-b886-95f5823ba4b6",
}

// mustFileID returns a deterministic UUIDv7 for the given fixture index.
func mustFileID(t *testing.T, index int) string {
	t.Helper()
	if index >= len(dualFileIDs) {
		t.Fatalf("fixture needs more file IDs (index %d)", index)
	}
	return dualFileIDs[index]
}

// writeDualTopic creates one topic with one managed file in both storages. The
// mirror shares the FileID and relative path with the open copy — it is the
// same logical content — while the stamps are caller-supplied so the freshness
// direction under test is explicit.
func writeDualTopic(t *testing.T, workspace config.Workspace, id domain.ID, slug, title, created, fileID, openStamp, closedStamp string) (string, string) {
	t.Helper()
	openTopic := filepath.Join(workspace.Histories, id.String()+"-"+slug)
	closedTopic := filepath.Join(workspace.Database, id.String()+"-"+slug)
	for topic, stamp := range map[string]string{openTopic: openStamp, closedTopic: closedStamp} {
		if err := os.MkdirAll(topic, 0o755); err != nil {
			t.Fatal(err)
		}
		metadata := markdown.TopicMetadata{
			ID: id, Title: title, Description: "dedupe-token topic", Created: created, UpdatedAt: stamp,
			Files: []markdown.ManifestFile{{
				ID: domain.FileID(fileID), Path: "note.md", Type: markdown.FileType("note.md"),
				Status: domain.StatusProgress, UpdatedAt: stamp,
			}},
		}
		if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename), metadata); err != nil {
			t.Fatal(err)
		}
		document, err := markdown.NewDocument(domain.Frontmatter{ID: id, Title: title, Status: domain.StatusProgress, Created: created}, "dedupe-token body")
		if err != nil {
			t.Fatal(err)
		}
		if err := markdown.WriteFile(filepath.Join(topic, "note.md"), document); err != nil {
			t.Fatal(err)
		}
	}
	return openTopic, closedTopic
}

// findRows returns every row matching the keyword, keyed by storage and type.
func findRows(t *testing.T, workspace config.Workspace, options Options) []Result {
	t.Helper()
	page, err := FindPage(workspace, options)
	if err != nil {
		t.Fatal(err)
	}
	return page.Items
}

func TestFindListsBothCopiesAndMarksTheNewerOne(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// The mirror is NEWER: a collaborator pushed. Both copies must still be
	// listed, and the closed row is the one that carries the notice.
	writeDualTopic(t, workspace, "00001", "dual", "Dual Topic", "2026-10-01", mustFileID(t, 0), "2026-11-13T10:45:30Z", "2026-11-13T16:02:11Z")
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}

	results := findRows(t, workspace, Options{Keyword: "dedupe-token"})
	if len(results) != 4 {
		t.Fatalf("results = %#v, want two rows per copy (topic + file)", results)
	}
	notices := make(map[string]string, len(results))
	for _, result := range results {
		notices[result.Storage+"/"+result.Type] = result.NewerCopy
	}
	if got := notices["closed/historic_file"]; got != domain.CopyRegionDatabase {
		t.Fatalf("closed file notice = %q, want %q (%#v)", got, domain.CopyRegionDatabase, notices)
	}
	if got := notices["closed/topic"]; got != domain.CopyRegionDatabase {
		t.Fatalf("closed topic notice = %q, want %q (%#v)", got, domain.CopyRegionDatabase, notices)
	}
	if got := notices["open/historic_file"]; got != "" {
		t.Fatalf("open file notice = %q, want none (%#v)", got, notices)
	}
	if got := notices["open/topic"]; got != "" {
		t.Fatalf("open topic notice = %q, want none (%#v)", got, notices)
	}
}

func TestFindMarksNewerLocallyWhenTheOpenCopyHasTheNewerStamp(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeDualTopic(t, workspace, "00001", "dual", "Dual Topic", "2026-10-01", mustFileID(t, 0), "2026-11-13T16:02:11Z", "2026-11-13T10:45:30Z")
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}

	results := findRows(t, workspace, Options{Keyword: "dedupe-token"})
	notices := make(map[string]string, len(results))
	for _, result := range results {
		notices[result.Storage+"/"+result.Type] = result.NewerCopy
	}
	if got := notices["open/historic_file"]; got != domain.CopyRegionLocal {
		t.Fatalf("open file notice = %q, want %q (%#v)", got, domain.CopyRegionLocal, notices)
	}
	if got := notices["closed/historic_file"]; got != "" {
		t.Fatalf("closed file notice = %q, want none (%#v)", got, notices)
	}
}

func TestFindPageCollapsesIdenticalCopiesIntoOneRow(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	stamp := "2026-11-13T10:45:30Z"
	writeDualTopic(t, workspace, "00001", "dual", "Dual Topic", "2026-10-01", mustFileID(t, 0), stamp, stamp)
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}

	// The raw read model keeps one row per storage copy; only the listing
	// surface folds them, so callers that verify storage contents still see both.
	raw, err := Find(workspace, Options{Keyword: "dedupe-token"})
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 4 {
		t.Fatalf("raw results = %#v, want both copies of the topic and its file", raw)
	}

	results := findRows(t, workspace, Options{Keyword: "dedupe-token"})
	if len(results) != 2 {
		t.Fatalf("results = %#v, want one row per logical entity", results)
	}
	for _, result := range results {
		if result.Storage != domain.StorageOpen.String() {
			t.Fatalf("result %s/%s = %+v, want the open copy", result.Storage, result.Type, result)
		}
		if result.NewerCopy != "" {
			t.Fatalf("result %s/%s notice = %q, want none", result.Storage, result.Type, result.NewerCopy)
		}
		if !result.Mirrored {
			t.Fatalf("result %s/%s = %+v, want mirrored flagged", result.Storage, result.Type, result)
		}
	}
}

func TestRecentTopicsPageCollapsesIdenticalCopies(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	stamp := "2026-11-13T08:53:54Z"
	writeDualTopic(t, workspace, "00001", "dual", "Dual Topic", "2026-10-01", mustFileID(t, 0), stamp, stamp)
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}

	page, err := RecentTopicsPage(workspace, Options{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("recent topics = %#v, want a single collapsed row", page.Items)
	}
	item := page.Items[0]
	if item.Storage != domain.StorageOpen.String() || item.NewerCopy != "" || !item.Mirrored {
		t.Fatalf("recent topic = %+v, want the open mirrored row", item)
	}
}

func TestFindCopiesSurvivePagination(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for index := 1; index <= 3; index++ {
		id := domain.ID(fmt.Sprintf("0000%d", index))
		writeDualTopic(t, workspace, id, "topic", fmt.Sprintf("Page Topic %d", index), fmt.Sprintf("2026-09-%02d", index), mustFileID(t, index),
			fmt.Sprintf("2026-11-13T0%d:45:30Z", index), fmt.Sprintf("2026-11-13T1%d:02:11Z", index))
	}
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}

	total := 0
	notices := 0
	for page := 1; page <= 10; page++ {
		page := page
		items := findRows(t, workspace, Options{Keyword: "dedupe-token", Page: page, PageSize: 2})
		if len(items) == 0 {
			break
		}
		total += len(items)
		for _, item := range items {
			if item.NewerCopy == domain.CopyRegionDatabase {
				notices++
			}
			if item.NewerCopy == domain.CopyRegionLocal {
				t.Fatalf("unexpected local notice for %s/%s", item.Storage, item.Type)
			}
		}
	}
	if total != 12 {
		t.Fatalf("pages yielded %d rows, want 12 (two rows per copy per topic)", total)
	}
	// Each of the three topics has one newer copy (closed) with a topic row and
	// a file row, so six rows carry a notice.
	if notices != 6 {
		t.Fatalf("newer notices = %d, want 6", notices)
	}
}

func TestFindClosedScopeKeepsOnlyTheMirror(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeDualTopic(t, workspace, "00001", "dual", "Dual Topic", "2026-10-01", mustFileID(t, 0), "2026-11-13T10:45:30Z", "2026-11-13T16:02:11Z")
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}

	for _, result := range findRows(t, workspace, Options{Keyword: "dedupe-token", ClosedOnly: true}) {
		if result.Storage != domain.StorageClosed.String() {
			t.Fatalf("closed-only result = %+v", result)
		}
		// With a single storage in scope there is no counterpart to compare.
		if result.NewerCopy != "" {
			t.Fatalf("closed-only notice = %q, want none", result.NewerCopy)
		}
	}
}

func TestFindKeepsClosedOnlyFiles(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	closedTopic := filepath.Join(workspace.Database, "00002-closed-only")
	if err := os.MkdirAll(closedTopic, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteTopicMetadata(filepath.Join(closedTopic, markdown.MetaFilename), markdown.TopicMetadata{ID: "00002", Title: "Closed Only", Created: "2026-10-02"}); err != nil {
		t.Fatal(err)
	}
	document, err := markdown.NewDocument(domain.Frontmatter{ID: "00002", Title: "Archived Note", Status: domain.StatusComplete, Created: "2026-10-02"}, "closed-token body")
	if err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(filepath.Join(closedTopic, "note.md"), document); err != nil {
		t.Fatal(err)
	}
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}

	page, err := FindPage(workspace, Options{Keyword: "closed-token"})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("results = %#v, want the closed-only file", page.Items)
	}
	if item := page.Items[0]; item.Storage != domain.StorageClosed.String() || item.Path != ".historic/.database/00002-closed-only/note.md" {
		t.Fatalf("closed-only result = %+v", item)
	}
}

// writeFileSeed writes one managed file plus its manifest entry with an explicit
// per-file stamp, which is what the file-level ordering must use.
func writeFileSeed(t *testing.T, topicPath, relative, title, created, frontmatterUpdated, stamp string) {
	t.Helper()
	path := filepath.Join(topicPath, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	document, err := markdown.NewDocument(domain.Frontmatter{Title: title, Status: domain.StatusProgress, Created: created, Updated: frontmatterUpdated}, "sundulan body")
	if err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(path, document); err != nil {
		t.Fatal(err)
	}
	meta, err := markdown.ReadTopicMetadata(topicPath)
	if err != nil {
		t.Fatal(err)
	}
	meta.Files = append(meta.Files, markdown.ManifestFile{
		ID: domain.FileID(mustFileID(t, len(meta.Files))), Path: filepath.ToSlash(relative),
		Type: markdown.FileType(relative), Status: domain.StatusProgress, UpdatedAt: stamp,
	})
	if err := markdown.WriteTopicMetadata(filepath.Join(topicPath, markdown.MetaFilename), meta); err != nil {
		t.Fatal(err)
	}
}

// TestFindSortsFilesByTheirOwnStamp reproduces issue #2: two files of one topic
// were edited on the same calendar day minutes apart. The file with the later
// stamp must come first even though both daily dates are identical.
func TestFindSortsFilesByTheirOwnStamp(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "1793876656894-feed")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename), markdown.TopicMetadata{
		ID: "1793876656894", Title: "Feed Sundulan", Created: "2026-11-12", UpdatedAt: "2026-11-12T21:30:15Z",
	}); err != nil {
		t.Fatal(err)
	}
	writeFileSeed(t, topic, "wos/01-implementasi-feed-gabungan.md", "Implementasi Feed", "2026-11-12", "2026-11-12", "2026-11-12T12:50:42Z")
	writeFileSeed(t, topic, "wos/02-auto-sambung-produk-biasa.md", "Auto Sambung", "2026-11-12", "2026-11-12", "2026-11-12T21:30:15Z")
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}

	results := findRows(t, workspace, Options{Keyword: "sundulan", Sort: "updated", Type: "task"})
	if len(results) != 2 {
		t.Fatalf("results = %#v, want both files", results)
	}
	if got := filepath.Base(results[0].Path); got != "02-auto-sambung-produk-biasa.md" {
		t.Fatalf("first file = %q, want the 21:30 stamp first (%#v)", got, results)
	}
	if results[0].UpdatedAt != "2026-11-12T21:30:15Z" {
		t.Fatalf("first file stamp = %q, want the high-resolution value", results[0].UpdatedAt)
	}
}

// TestRecentTopicsPageOrdersByHighResStampOnTheSameDay covers AC-2: two topics
// touched on the same calendar day must be ordered by the minute that separates
// them, not by ID.
func TestRecentTopicsPageOrdersByHighResStampOnTheSameDay(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, seed := range []struct {
		id    domain.ID
		title string
		stamp string
	}{
		{"1794526185679", "Later Today", "2026-11-13T10:45:30Z"},
		{"1793629449341", "Earlier Today", "2026-11-13T09:12:03Z"},
	} {
		topic := filepath.Join(workspace.Histories, seed.id.String()+"-topic")
		if err := os.MkdirAll(topic, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename), markdown.TopicMetadata{
			ID: seed.id, Title: seed.title, Created: "2026-11-13", UpdatedAt: seed.stamp,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}

	page, err := RecentTopicsPage(workspace, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 || page.Items[0].TopicID != "1794526185679" {
		t.Fatalf("recent topics = %#v, want the 10:45 topic first", page.Items)
	}
}

func TestRecentTopicsPageListsMirrorVersions(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeDualTopic(t, workspace, "00001", "dual", "Dual Topic", "2026-10-01", mustFileID(t, 0), "2026-11-13T10:45:30Z", "2026-11-13T16:02:11Z")
	if _, err := indexer.Rebuild(workspace); err != nil {
		t.Fatal(err)
	}

	page, err := RecentTopicsPage(workspace, Options{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("recent topics = %#v, want both copies", page.Items)
	}
	if page.Items[0].Storage != domain.StorageClosed.String() || page.Items[0].NewerCopy != domain.CopyRegionDatabase {
		t.Fatalf("first recent topic = %+v, want the newer database copy", page.Items[0])
	}
	if page.Items[1].Storage != domain.StorageOpen.String() || page.Items[1].NewerCopy != "" {
		t.Fatalf("second recent topic = %+v, want the plain local copy", page.Items[1])
	}
}
