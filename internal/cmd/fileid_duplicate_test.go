package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"historic/internal/domain"
	"historic/internal/markdown"
)

const duplicateFileID = "01a1a8a1-6258-757e-95f0-fd5228110e62"

// seedDuplicateFileIDTopic adds shared.md to an already-created topic and
// records duplicateFileID for it in the topic manifest, reproducing the legacy
// topic-ID migration state from issue #1.
func seedDuplicateFileIDTopic(t *testing.T, root, folder string, topicID domain.ID) {
	t.Helper()
	path := filepath.Join(root, ".historic", folder)
	document, err := markdown.NewDocument(domain.Frontmatter{ID: topicID, Title: "Shared File", Status: domain.StatusProgress, Created: "2026-10-01"}, "duplicate fileid body\n")
	if err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(filepath.Join(path, "shared.md"), document); err != nil {
		t.Fatal(err)
	}
	metaPath := filepath.Join(path, markdown.MetaFilename)
	metadata, err := markdown.ReadTopicMetadata(path)
	if err != nil {
		t.Fatal(err)
	}
	metadata.Files = []markdown.ManifestFile{{ID: domain.FileID(duplicateFileID), Path: "shared.md", Type: "file", Status: domain.StatusProgress}}
	if err := markdown.WriteTopicMetadata(metaPath, metadata); err != nil {
		t.Fatal(err)
	}
}

func setupDuplicateFileIDWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Alpha", "--id", "1793830502504"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Beta", "--id", "1793830502505"); err != nil {
		t.Fatal(err)
	}
	seedDuplicateFileIDTopic(t, root, "1793830502504-alpha", "1793830502504")
	seedDuplicateFileIDTopic(t, root, "1793830502505-beta", "1793830502505")
	return root
}

// TestSaveAllSucceedsWithDuplicateFileIDs is the issue #1 regression: one
// FileID recorded in two topics must not abort the save with SQLite 1555.
func TestSaveAllSucceedsWithDuplicateFileIDs(t *testing.T) {
	root := setupDuplicateFileIDWorkspace(t)

	output, err := executeCommand(t, "save", "--all", "--json")
	if err != nil {
		t.Fatalf("save --all: %v output=%s", err, output)
	}
	var response struct {
		OK   bool `json:"ok"`
		Data struct {
			Saved       int `json:"saved"`
			FailedCount int `json:"failed_count"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("invalid JSON %q: %v", output, err)
	}
	if !response.OK || response.Data.Saved != 2 || response.Data.FailedCount != 0 {
		t.Fatalf("response = %#v output=%s", response, output)
	}
	// The mirror copies the shared FileID into closed storage too: an
	// open/closed and closed/closed pair must both stay indexable.
	if _, err := executeCommand(t, "rebuild"); err != nil {
		t.Fatalf("rebuild after duplicate-FileID save: %v", err)
	}
	for _, storage := range []string{"open", "closed"} {
		closed := filepath.Join(root, ".historic", ".database", "1793830502504-alpha", "shared.md")
		if storage == "closed" {
			if _, err := os.Stat(closed); err != nil {
				t.Fatalf("closed snapshot missing: %v", err)
			}
		}
	}
}

// TestSaveIsIdempotentForDuplicateFileIDs asserts that saving the same topic
// twice is a no-op on the second call instead of a constraint failure.
func TestSaveIsIdempotentForDuplicateFileIDs(t *testing.T) {
	root := setupDuplicateFileIDWorkspace(t)

	if output, err := executeCommand(t, "save", "1793830502504", "--json"); err != nil {
		t.Fatalf("first save: %v output=%s", err, output)
	}
	snapshot := filepath.Join(root, ".historic", ".database", "1793830502504-alpha")
	before, err := os.ReadFile(filepath.Join(snapshot, markdown.MetaFilename))
	if err != nil {
		t.Fatal(err)
	}
	if output, err := executeCommand(t, "save", "1793830502504", "--json"); err != nil {
		t.Fatalf("second save: %v output=%s", err, output)
	}
	after, err := os.ReadFile(filepath.Join(snapshot, markdown.MetaFilename))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatalf("second save changed the snapshot:\nbefore=%s\nafter=%s", before, after)
	}
}

// TestFindShowsSharedFileInBothTopics asserts the read model keeps one row per
// topic instead of letting the duplicate FileID overwrite the other topic.
func TestFindShowsSharedFileInBothTopics(t *testing.T) {
	setupDuplicateFileIDWorkspace(t)
	if output, err := executeCommand(t, "rebuild"); err != nil {
		t.Fatalf("rebuild: %v output=%s", err, output)
	}

	output, err := executeCommand(t, "find", "duplicate", "--json")
	if err != nil {
		t.Fatalf("find: %v output=%s", err, output)
	}
	var response struct {
		OK   bool `json:"ok"`
		Data struct {
			Items []struct {
				ID      string `json:"id"`
				TopicID string `json:"topic_id"`
				Path    string `json:"path"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("invalid JSON %q: %v", output, err)
	}
	topics := make(map[string]bool)
	for _, item := range response.Data.Items {
		if item.TopicID != "" && strings.Contains(item.Path, "shared.md") {
			topics[item.TopicID] = true
		}
	}
	if !topics["1793830502504"] || !topics["1793830502505"] {
		t.Fatalf("find results missing a topic: %#v (output=%s)", topics, output)
	}
}

// TestDoctorReportsDuplicateFileIDs covers WO 03: doctor detects the duplicate
// read-only, in both human and JSON output.
func TestDoctorReportsDuplicateFileIDs(t *testing.T) {
	setupDuplicateFileIDWorkspace(t)

	output, err := executeCommand(t, "doctor", "--json")
	if err != nil {
		t.Fatalf("doctor --json: %v output=%s", err, output)
	}
	var response struct {
		Data struct {
			DuplicateFileIDs []struct {
				FileID  string `json:"file_id"`
				Storage string `json:"storage"`
				Topics  []struct {
					TopicID string `json:"topic_id"`
					Path    string `json:"path"`
				} `json:"topics"`
			} `json:"duplicate_file_ids"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("invalid JSON %q: %v", output, err)
	}
	found := false
	for _, duplicate := range response.Data.DuplicateFileIDs {
		if duplicate.FileID != duplicateFileID {
			continue
		}
		found = true
		if len(duplicate.Topics) != 2 {
			t.Fatalf("duplicate topics = %#v, want 2", duplicate.Topics)
		}
	}
	if !found {
		t.Fatalf("doctor did not report duplicate FileID: %s", output)
	}

	human, err := executeCommand(t, "doctor")
	if err != nil {
		t.Fatalf("doctor: %v output=%s", err, human)
	}
	if !strings.Contains(human, "Duplicate FileIDs: 1") || !strings.Contains(human, duplicateFileID) {
		t.Fatalf("doctor human output missing duplicate report:\n%s", human)
	}
}

// TestDoctorReportsNoDuplicateFileIDs guards against false positives.
func TestDoctorReportsNoDuplicateFileIDs(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Alpha", "--id", "1793830502504"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Beta", "--id", "1793830502505"); err != nil {
		t.Fatal(err)
	}
	output, err := executeCommand(t, "doctor")
	if err != nil {
		t.Fatalf("doctor: %v output=%s", err, output)
	}
	if !strings.Contains(output, "Duplicate FileIDs: none") {
		t.Fatalf("doctor reported a false positive:\n%s", output)
	}
}
