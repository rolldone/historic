package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"historic/internal/domain"
	"historic/internal/markdown"
)

// writeManagedStaleFile writes one managed task file with an old created/updated
// date so a later content edit must be bumped to today.
func writeManagedStaleFile(t *testing.T, path string) {
	t.Helper()
	document, err := markdown.NewDocument(domain.Frontmatter{
		ID: "00001", Title: "Work", Status: domain.StatusProgress, Created: "2026-01-01", Updated: "2026-01-01",
	}, "original body")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(path, document); err != nil {
		t.Fatal(err)
	}
}

func editManagedBody(t *testing.T, path, body string) {
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

func TestSaveBumpsChangedContentBeforeMirrorAndCommit(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Topic", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	openFile := filepath.Join(root, ".historic", "00001-topic", "wos", "01-work.md")
	writeManagedStaleFile(t, openFile)
	// Baseline: index the new file without bumping (no baseline exists yet).
	if _, err := executeCommand(t, "rebuild"); err != nil {
		t.Fatal(err)
	}
	editManagedBody(t, openFile, "edited body")

	output, err := executeCommand(t, "save", "00001", "--json")
	if err != nil {
		t.Fatalf("save: %v output=%s", err, output)
	}
	var response struct {
		OK   bool `json:"ok"`
		Data struct {
			BumpedFiles int `json:"bumped_files"`
			Commit      *struct {
				Committed bool   `json:"committed"`
				Message   string `json:"message"`
			} `json:"commit"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("decode %q: %v", output, err)
	}
	today := time.Now().UTC().Format("2006-01-02")
	if response.Data.BumpedFiles != 1 {
		t.Fatalf("bumped_files = %d, want 1 (%s)", response.Data.BumpedFiles, output)
	}
	document, err := markdown.ParseFile(openFile)
	if err != nil {
		t.Fatal(err)
	}
	if document.Frontmatter.Updated != today {
		t.Fatalf("open updated = %q, want %q", document.Frontmatter.Updated, today)
	}
	// The mirrored copy committed by save must already carry the bumped date.
	closedFile := filepath.Join(root, ".historic", ".database", "00001-topic", "wos", "01-work.md")
	closed, err := markdown.ParseFile(closedFile)
	if err != nil {
		t.Fatal(err)
	}
	if closed.Frontmatter.Updated != today {
		t.Fatalf("closed updated = %q, want %q (bump must precede mirror)", closed.Frontmatter.Updated, today)
	}
	if response.Data.Commit == nil || !response.Data.Commit.Committed {
		t.Fatalf("commit = %#v, want a committed mirror including the bump", response.Data.Commit)
	}
	if !strings.Contains(response.Data.Commit.Message, "auto: save 00001") {
		t.Fatalf("commit message = %q", response.Data.Commit.Message)
	}

	// A second save with unchanged content must not bump again.
	output, err = executeCommand(t, "save", "00001", "--json")
	if err != nil {
		t.Fatalf("second save: %v", err)
	}
	var second struct {
		Data struct {
			BumpedFiles int `json:"bumped_files"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &second); err != nil {
		t.Fatal(err)
	}
	if second.Data.BumpedFiles != 0 {
		t.Fatalf("second save bumped_files = %d, want 0", second.Data.BumpedFiles)
	}
}

func TestRebuildBumpsContentChangedOutsideHistoric(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Topic", "--id", "00002"); err != nil {
		t.Fatal(err)
	}
	openFile := filepath.Join(root, ".historic", "00002-topic", "wos", "01-work.md")
	writeManagedStaleFile(t, openFile)
	if _, err := executeCommand(t, "rebuild"); err != nil {
		t.Fatal(err)
	}
	editManagedBody(t, openFile, "edited outside historic")

	output, err := executeCommand(t, "rebuild", "--json")
	if err != nil {
		t.Fatalf("rebuild: %v output=%s", err, output)
	}
	var response struct {
		OK   bool `json:"ok"`
		Data struct {
			BumpedFiles int `json:"bumped_files"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("decode %q: %v", output, err)
	}
	today := time.Now().UTC().Format("2006-01-02")
	if response.Data.BumpedFiles != 1 {
		t.Fatalf("rebuild bumped_files = %d, want 1 (%s)", response.Data.BumpedFiles, output)
	}
	document, err := markdown.ParseFile(openFile)
	if err != nil {
		t.Fatal(err)
	}
	if document.Frontmatter.Updated != today {
		t.Fatalf("updated = %q, want %q", document.Frontmatter.Updated, today)
	}
	// rebuild --json must still carry the legacy fields.
	var legacy struct {
		Data struct {
			Topics  int    `json:"topics"`
			Records int    `json:"records"`
			Index   string `json:"index"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &legacy); err != nil {
		t.Fatal(err)
	}
	if legacy.Data.Topics == 0 || legacy.Data.Records == 0 || legacy.Data.Index == "" {
		t.Fatalf("legacy rebuild fields regressed: %s", output)
	}
}
