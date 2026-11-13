package lifecycle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/identifier"
	"historic/internal/markdown"
)

// stampTopicFixture builds one open topic with two managed files so tests can
// prove a stamp advances only on the file that actually changed.
func stampTopicFixture(t *testing.T, topicID, created string) (config.Workspace, string, string, string) {
	t.Helper()
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, topicID+"-stamps")
	if err := os.MkdirAll(filepath.Join(topic, "wos"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename),
		markdown.TopicMetadata{ID: domain.ID(topicID), Title: "Stamps", Created: created}); err != nil {
		t.Fatal(err)
	}
	first := filepath.Join(topic, "wos", "01-first.md")
	second := filepath.Join(topic, "wos", "02-second.md")
	for path, title := range map[string]string{first: "First", second: "Second"} {
		document, docErr := markdown.NewDocument(domain.Frontmatter{
			ID: domain.ID(topicID), Title: title, Status: domain.StatusProgress, Created: created, Updated: created,
		}, "body")
		if docErr != nil {
			t.Fatal(docErr)
		}
		if err := markdown.WriteFile(path, document); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := RebuildMetadata(workspace); err != nil {
		t.Fatal(err)
	}
	return workspace, topic, first, second
}

func withFrozenStampClock(t *testing.T, value time.Time) {
	t.Helper()
	previous := stampClock
	stampClock = func() time.Time { return value }
	t.Cleanup(func() { stampClock = previous })
}

func stampFor(t *testing.T, topic, relative string) string {
	t.Helper()
	metadata, err := markdown.ReadTopicMetadata(topic)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range metadata.Files {
		if file.Path == relative {
			return file.UpdatedAt
		}
	}
	return ""
}

func TestStampTopicFilesAdvancesOnlyListedFile(t *testing.T) {
	workspace, topic, _, _ := stampTopicFixture(t, "1794526185679", "2026-11-01")
	at := time.Date(2026, 11, 13, 10, 45, 30, 0, time.UTC)

	changed, err := StampTopicFiles(workspace, topic, []string{"wos/01-first.md"}, at)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected the manifest to change")
	}

	first := stampFor(t, topic, "wos/01-first.md")
	second := stampFor(t, topic, "wos/02-second.md")
	if first != "2026-11-13T10:45:30Z" {
		t.Fatalf("listed file stamp = %q", first)
	}
	if second == first {
		t.Fatalf("unlisted file stamp advanced to %q; siblings must stay untouched", second)
	}

	metadata, err := markdown.ReadTopicMetadata(topic)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.UpdatedAt != "2026-11-13T10:45:30Z" {
		t.Fatalf("topic stamp = %q", metadata.UpdatedAt)
	}
}

func TestStampTopicFilesIsMonotonicAndIdempotent(t *testing.T) {
	workspace, topic, _, _ := stampTopicFixture(t, "1794526185679", "2026-11-01")
	newer := time.Date(2026, 11, 13, 10, 45, 30, 0, time.UTC)
	older := time.Date(2026, 11, 10, 8, 0, 0, 0, time.UTC)

	if _, err := StampTopicFiles(workspace, topic, []string{"wos/01-first.md"}, newer); err != nil {
		t.Fatal(err)
	}
	// A second call with the same instant must report no change: re-stamping
	// must not dirty the manifest.
	changed, err := StampTopicFiles(workspace, topic, []string{"wos/01-first.md"}, newer)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("re-stamping with the same instant must be a no-op")
	}
	// An older instant must never move the stamp backwards.
	if _, err := StampTopicFiles(workspace, topic, []string{"wos/01-first.md"}, older); err != nil {
		t.Fatal(err)
	}
	if got := stampFor(t, topic, "wos/01-first.md"); got != "2026-11-13T10:45:30Z" {
		t.Fatalf("stamp moved backwards: %q", got)
	}
}

func TestBumpChangedFilesStampsOnlyTheEditedFile(t *testing.T) {
	workspace, topic, _, second := stampTopicFixture(t, "1794526185679", "2026-11-01")
	withFrozenStampClock(t, time.Date(2026, 11, 13, 21, 30, 15, 0, time.UTC))
	withFrozenStatusNow(t, time.Date(2026, 11, 13, 21, 30, 15, 0, time.UTC))

	writeBody(t, second, "body edited later the same day")

	result := BumpChangedFiles(workspace, []domain.ID{"1794526185679"})
	if result.Count() != 1 {
		t.Fatalf("bumped = %v, want only the edited file", result.BumpedFiles)
	}

	firstStamp := stampFor(t, topic, "wos/01-first.md")
	secondStamp := stampFor(t, topic, "wos/02-second.md")
	if secondStamp != "2026-11-13T21:30:15Z" {
		t.Fatalf("edited file stamp = %q", secondStamp)
	}
	if firstStamp == secondStamp {
		t.Fatalf("untouched file shares the edited stamp %q; isolation is broken", firstStamp)
	}
}

func TestChangeFileStatusStampsTheChangedFileOnly(t *testing.T) {
	workspace, topic, _, _ := stampTopicFixture(t, "1794526185679", "2026-11-01")
	withFrozenStatusNow(t, time.Date(2026, 11, 13, 9, 12, 3, 0, time.UTC))

	if _, err := ChangeFileStatus(workspace, "wos/01-first.md", domain.StatusComplete); err != nil {
		t.Fatal(err)
	}

	if got := stampFor(t, topic, "wos/01-first.md"); got != "2026-11-13T09:12:03Z" {
		t.Fatalf("status-changed file stamp = %q", got)
	}
	if got := stampFor(t, topic, "wos/02-second.md"); got == "2026-11-13T09:12:03Z" {
		t.Fatal("sibling file stamp advanced on an unrelated status change")
	}
}

func TestBackfillTopicStampsFromIdentities(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topicID := domain.ID("1794526185679")
	fileID, err := identifier.Default.New()
	if err != nil {
		t.Fatal(err)
	}
	instant, ok := identifier.FileIDTime(fileID)
	if !ok {
		t.Fatal("expected a UUIDv7 file identity")
	}
	topicIDInstant, ok := identifier.TopicIDTime(topicID)
	if !ok {
		t.Fatal("expected an epoch-millisecond topic identity")
	}

	topic := filepath.Join(workspace.Histories, topicID.String()+"-backfill")
	if err := os.MkdirAll(filepath.Join(topic, "wos"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Metadata without any stamp: this is the legacy workspace shape a rebuild
	// must repair.
	if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename), markdown.TopicMetadata{
		ID: topicID, Title: "Backfill", Created: "2026-11-01",
		Files: []markdown.ManifestFile{{ID: fileID, Path: "wos/01-legacy.md", Type: "task", Status: domain.StatusProgress}},
	}); err != nil {
		t.Fatal(err)
	}
	document, err := markdown.NewDocument(domain.Frontmatter{
		ID: topicID, Title: "Legacy", Status: domain.StatusProgress, Created: "2026-11-01", Updated: "2026-11-02",
	}, "body")
	if err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(filepath.Join(topic, "wos", "01-legacy.md"), document); err != nil {
		t.Fatal(err)
	}

	written, err := BackfillTopicStamps(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if written != 1 {
		t.Fatalf("backfilled manifests = %d, want 1", written)
	}

	metadata, err := markdown.ReadTopicMetadata(topic)
	if err != nil {
		t.Fatal(err)
	}
	if want := markdown.FormatTimestamp(topicIDInstant); metadata.UpdatedAt != want {
		t.Fatalf("topic stamp = %q, want %q", metadata.UpdatedAt, want)
	}
	if want := markdown.FormatTimestamp(instant); metadata.Files[0].UpdatedAt != want {
		t.Fatalf("file stamp = %q, want %q", metadata.Files[0].UpdatedAt, want)
	}
}

func TestBackfillTopicStampsLeavesLegacyFiveDigitTopicsEmpty(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "00042-legacy")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename),
		markdown.TopicMetadata{ID: "00042", Title: "Legacy", Created: "2026-09-01"}); err != nil {
		t.Fatal(err)
	}

	if _, err := BackfillTopicStamps(workspace); err != nil {
		t.Fatal(err)
	}

	metadata, err := markdown.ReadTopicMetadata(topic)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.UpdatedAt != "" {
		t.Fatalf("five-digit topic stamp = %q; it carries no instant to derive from", metadata.UpdatedAt)
	}
}

func TestBackfillTopicStampsPreservesExistingStamps(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topicID := domain.ID("1794526185679")
	fileID, err := identifier.Default.New()
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, topicID.String()+"-kept")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	const existing = "2026-11-13T07:08:09Z"
	if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename), markdown.TopicMetadata{
		ID: topicID, Title: "Kept", Created: "2026-11-01", UpdatedAt: existing,
		Files: []markdown.ManifestFile{{ID: fileID, Path: "spec.md", Type: "spec", Status: domain.StatusProgress, UpdatedAt: existing}},
	}); err != nil {
		t.Fatal(err)
	}

	written, err := BackfillTopicStamps(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if written != 0 {
		t.Fatalf("backfilled manifests = %d, want 0 (nothing to fill)", written)
	}

	metadata, err := markdown.ReadTopicMetadata(topic)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.UpdatedAt != existing || metadata.Files[0].UpdatedAt != existing {
		t.Fatalf("existing stamps overwritten: %#v", metadata)
	}
}

func TestRebuildMetadataBackfillsStampsOnceAndIsIdempotent(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topicID := domain.ID("1794526185679")
	topic := filepath.Join(workspace.Histories, topicID.String()+"-idem")
	if err := os.MkdirAll(filepath.Join(topic, "wos"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename),
		markdown.TopicMetadata{ID: topicID, Title: "Idem", Created: "2026-11-01"}); err != nil {
		t.Fatal(err)
	}
	document, err := markdown.NewDocument(domain.Frontmatter{
		ID: topicID, Title: "Task", Status: domain.StatusProgress, Created: "2026-11-01", Updated: "2026-11-02",
	}, "body")
	if err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(filepath.Join(topic, "wos", "01-task.md"), document); err != nil {
		t.Fatal(err)
	}

	if _, err := RebuildMetadata(workspace); err != nil {
		t.Fatal(err)
	}
	metaPath := filepath.Join(topic, markdown.MetaFilename)
	before, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := markdown.ReadTopicMetadata(topic)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.UpdatedAt == "" {
		t.Fatal("rebuild did not backfill the topic stamp")
	}
	if len(metadata.Files) != 1 || metadata.Files[0].UpdatedAt == "" {
		t.Fatalf("rebuild did not backfill the file stamp: %#v", metadata.Files)
	}

	second, err := RebuildMetadata(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if second.Updated != 0 {
		t.Fatalf("second rebuild rewrote %d manifest(s); rebuild must be idempotent", second.Updated)
	}
	after, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatalf("second rebuild changed canonical metadata:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestStampsSurviveMirrorCopy(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topicID := domain.ID("1794526185679")
	source := filepath.Join(workspace.Histories, topicID.String()+"-mirror")
	if err := os.MkdirAll(filepath.Join(source, "wos"), 0o755); err != nil {
		t.Fatal(err)
	}
	const stamp = "2026-11-13T10:45:30Z"
	if err := markdown.WriteTopicMetadata(filepath.Join(source, markdown.MetaFilename), markdown.TopicMetadata{
		ID: topicID, Title: "Mirror", Created: "2026-11-01", UpdatedAt: stamp,
		Files: []markdown.ManifestFile{{
			ID:        mustFileID(t),
			Path:      "spec.md",
			Type:      "spec",
			Status:    domain.StatusProgress,
			UpdatedAt: stamp,
		}},
	}); err != nil {
		t.Fatal(err)
	}

	mirror := filepath.Join(workspace.Database, topicID.String()+"-mirror")
	if err := copyTopicTree(source, mirror); err != nil {
		t.Fatal(err)
	}

	metadata, err := markdown.ReadTopicMetadata(mirror)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.UpdatedAt != stamp {
		t.Fatalf("mirror topic stamp = %q", metadata.UpdatedAt)
	}
	if len(metadata.Files) != 1 || metadata.Files[0].UpdatedAt != stamp {
		t.Fatalf("mirror file stamp lost: %#v", metadata.Files)
	}
}

func mustFileID(t *testing.T) domain.FileID {
	t.Helper()
	id, err := identifier.Default.New()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// TestManifestStampsAreValidated guards the RFC3339 contract: a hand-edited
// value that is not a canonical UTC second-precision stamp must be rejected.
func TestManifestStampsAreValidated(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(workspace.Histories, "1794526185679-bad")
	if err := os.MkdirAll(topic, 0o755); err != nil {
		t.Fatal(err)
	}
	err = markdown.WriteTopicMetadata(filepath.Join(topic, markdown.MetaFilename),
		markdown.TopicMetadata{ID: "1794526185679", Title: "Bad", Created: "2026-11-01", UpdatedAt: "2026-11-13 10:45:30"})
	if err == nil || !strings.Contains(err.Error(), "RFC3339") {
		t.Fatalf("expected an RFC3339 validation error, got %v", err)
	}
}
