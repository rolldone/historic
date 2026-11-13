package lifecycle

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"
)

// guardFixture is one side (open or mirror) of a topic used by the conflict
// guard tests.
type guardFixture struct {
	stamp string
	body  string
}

// writeGuardTopic materializes a topic directory holding the given managed
// files, with per-file stamps taken from the fixture. Each file is valid
// Historic Markdown so the guard can fall back to the frontmatter dates.
func writeGuardTopic(t *testing.T, topicPath string, id domain.ID, files map[string]guardFixture) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(topicPath, "wos"), 0o755); err != nil {
		t.Fatal(err)
	}
	metadata := markdown.TopicMetadata{ID: id, Title: "Guard Topic", Created: "2026-11-01", Updated: "2026-11-01"}
	for _, path := range sortedKeys(files) {
		fixture := files[path]
		target := filepath.Join(topicPath, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		document := "---\ntitle: File\nstatus: progress\ncreated: \"2026-11-01\"\nupdated: \"2026-11-01\"\n---\n" + fixture.body + "\n"
		if err := os.WriteFile(target, []byte(document), 0o644); err != nil {
			t.Fatal(err)
		}
		fileID := "01a1d3ab-90b1-7ca7-8087-c597029e9e54"
		if path != sortedKeys(files)[0] {
			fileID = "01a1d3ab-9116-7755-aabe-5efccc8b0b22"
		}
		metadata.Files = append(metadata.Files, markdown.ManifestFile{
			ID: domain.FileID(fileID), Path: path, Type: markdown.FileType(path), Status: domain.StatusProgress, UpdatedAt: fixture.stamp,
		})
	}
	if err := markdown.WriteTopicMetadata(filepath.Join(topicPath, markdown.MetaFilename), metadata); err != nil {
		t.Fatal(err)
	}
}

func sortedKeys(files map[string]guardFixture) []string {
	keys := make([]string, 0, len(files))
	for key := range files {
		keys = append(keys, key)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}

// guardWorkspace builds a workspace where the open copy and the mirror hold the
// same managed paths but different stamps and bodies.
func guardWorkspace(t *testing.T, open, mirror map[string]guardFixture) (config.Workspace, domain.ID) {
	t.Helper()
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	id := domain.ID("1794526185679")
	writeGuardTopic(t, filepath.Join(workspace.Histories, id.String()+"-guard"), id, open)
	writeGuardTopic(t, filepath.Join(workspace.Database, id.String()+"-guard"), id, mirror)
	return workspace, id
}

func TestMirrorRejectsWhenTheMirrorCopyIsNewerPerFile(t *testing.T) {
	workspace, id := guardWorkspace(t,
		map[string]guardFixture{"wos/01-task.md": {stamp: "2026-11-12T10:00:00Z", body: "local edit"}},
		map[string]guardFixture{"wos/01-task.md": {stamp: "2026-11-12T18:00:00Z", body: "pulled edit"}},
	)
	mirrorBefore, err := os.ReadFile(filepath.Join(workspace.Database, id.String()+"-guard", "_meta.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = NewService(workspace).MirrorTopic(id)
	if err == nil {
		t.Fatal("mirror accepted a newer closed copy")
	}
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("error = %v, want a conflict", err)
	}
	if !strings.Contains(err.Error(), "wos/01-task.md") {
		t.Fatalf("error does not name the offending path: %v", err)
	}
	mirrorAfter, err := os.ReadFile(filepath.Join(workspace.Database, id.String()+"-guard", "_meta.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(mirrorBefore) != string(mirrorAfter) {
		t.Fatal("rejected mirror modified the closed copy")
	}
}

func TestCloseRejectsWhenTheMirrorCopyIsNewerPerFile(t *testing.T) {
	workspace, id := guardWorkspace(t,
		map[string]guardFixture{"wos/01-task.md": {stamp: "2026-11-12T10:00:00Z", body: "local edit"}},
		map[string]guardFixture{"wos/01-task.md": {stamp: "2026-11-12T18:00:00Z", body: "pulled edit"}},
	)
	if _, err := NewService(workspace).Close(id); err == nil {
		t.Fatal("close accepted a newer closed copy")
	} else if !strings.Contains(err.Error(), "wos/01-task.md") {
		t.Fatalf("error does not name the offending path: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspace.Histories, id.String()+"-guard")); err != nil {
		t.Fatalf("rejected close removed the open copy: %v", err)
	}
}

func TestGuardAllowsIdenticalContentEvenWhenTheMirrorStampIsNewer(t *testing.T) {
	workspace, id := guardWorkspace(t,
		map[string]guardFixture{"wos/01-task.md": {stamp: "2026-11-12T10:00:00Z", body: "same body"}},
		map[string]guardFixture{"wos/01-task.md": {stamp: "2026-11-12T18:00:00Z", body: "same body"}},
	)
	if _, err := NewService(workspace).MirrorTopic(id); err != nil {
		t.Fatalf("identical content was rejected: %v", err)
	}
}

func TestGuardAllowsLocalWorkThatIsNewer(t *testing.T) {
	workspace, id := guardWorkspace(t,
		map[string]guardFixture{"wos/01-task.md": {stamp: "2026-11-12T20:00:00Z", body: "local edit"}},
		map[string]guardFixture{"wos/01-task.md": {stamp: "2026-11-12T18:00:00Z", body: "pulled edit"}},
	)
	if _, err := NewService(workspace).MirrorTopic(id); err != nil {
		t.Fatalf("newer local work was rejected: %v", err)
	}
}

func TestGuardFallsBackToTheFileDateWhenNoStampIsRecorded(t *testing.T) {
	workspace, id := guardWorkspace(t,
		map[string]guardFixture{"wos/01-task.md": {body: "local edit"}},
		map[string]guardFixture{"wos/01-task.md": {body: "pulled edit"}},
	)
	// Without any stamp the guard must not invent a conflict: the calendar
	// dates are equal, so the write proceeds instead of blocking the operator.
	if _, err := NewService(workspace).MirrorTopic(id); err != nil {
		t.Fatalf("missing stamps produced a false positive: %v", err)
	}
}

func TestForceBypassesTheGuard(t *testing.T) {
	workspace, id := guardWorkspace(t,
		map[string]guardFixture{"wos/01-task.md": {stamp: "2026-11-12T10:00:00Z", body: "local edit"}},
		map[string]guardFixture{"wos/01-task.md": {stamp: "2026-11-12T18:00:00Z", body: "pulled edit"}},
	)
	if _, err := NewService(workspace).MirrorTopicWithOptions(id, WriteOptions{Force: true}); err != nil {
		t.Fatalf("--force did not bypass the guard: %v", err)
	}
}

func TestGuardOnlyRejectsTheOffendingTopicInCloseAll(t *testing.T) {
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	conflicting := domain.ID("1794526185679")
	clean := domain.ID("1794526185680")
	writeGuardTopic(t, filepath.Join(workspace.Histories, conflicting.String()+"-guard"), conflicting,
		map[string]guardFixture{"wos/01-task.md": {stamp: "2026-11-12T10:00:00Z", body: "local edit"}})
	writeGuardTopic(t, filepath.Join(workspace.Database, conflicting.String()+"-guard"), conflicting,
		map[string]guardFixture{"wos/01-task.md": {stamp: "2026-11-12T18:00:00Z", body: "pulled edit"}})
	writeGuardTopic(t, filepath.Join(workspace.Histories, clean.String()+"-clean"), clean,
		map[string]guardFixture{"wos/01-task.md": {stamp: "2026-11-12T20:00:00Z", body: "local edit"}})

	result, err := NewService(workspace).CloseAll()
	if err == nil {
		t.Fatal("close-all reported success despite a rejected topic")
	}
	if result.Closed != 1 || result.FailedCount != 1 {
		t.Fatalf("close-all result = closed %d, failed %d; want 1/1", result.Closed, result.FailedCount)
	}
	if result.Failed[0].ID != conflicting.String() || !strings.Contains(result.Failed[0].Error, "wos/01-task.md") {
		t.Fatalf("failure does not name the offending topic and path: %+v", result.Failed[0])
	}
}

func TestGuardWorksWithoutTheSQLiteIndex(t *testing.T) {
	workspace, id := guardWorkspace(t,
		map[string]guardFixture{"wos/01-task.md": {stamp: "2026-11-12T10:00:00Z", body: "local edit"}},
		map[string]guardFixture{"wos/01-task.md": {stamp: "2026-11-12T18:00:00Z", body: "pulled edit"}},
	)
	if err := os.Remove(workspace.Index); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if _, err := NewService(workspace).MirrorTopic(id); err == nil {
		t.Fatal("guard stopped working once the index was deleted")
	}
}

func TestMirrorCopiesThePerFileStamps(t *testing.T) {
	workspace, id := guardWorkspace(t,
		map[string]guardFixture{"wos/01-task.md": {stamp: "2026-11-12T20:00:00Z", body: "local edit"}},
		map[string]guardFixture{"wos/01-task.md": {stamp: "2026-11-12T18:00:00Z", body: "pulled edit"}},
	)
	if _, err := NewService(workspace).MirrorTopicWithOptions(id, WriteOptions{Force: true}); err != nil {
		t.Fatalf("mirror: %v", err)
	}
	metadata, err := markdown.ReadTopicMetadata(filepath.Join(workspace.Database, id.String()+"-guard"))
	if err != nil {
		t.Fatal(err)
	}
	if len(metadata.Files) != 1 || metadata.Files[0].UpdatedAt != "2026-11-12T20:00:00Z" {
		t.Fatalf("mirror did not carry the open per-file stamp: %+v", metadata.Files)
	}
}
