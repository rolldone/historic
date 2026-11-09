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

func decodeBumpedFiles(t *testing.T, output string) int {
	t.Helper()
	var response struct {
		Data struct {
			BumpedFiles int `json:"bumped_files"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("decode %q: %v", output, err)
	}
	return response.Data.BumpedFiles
}

func TestSaveNoBumpSkipsDateButStillMirrorsAndCommits(t *testing.T) {
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
	if _, err := executeCommand(t, "rebuild"); err != nil {
		t.Fatal(err)
	}
	editManagedBody(t, openFile, "edited body")

	output, err := executeCommand(t, "save", "00001", "--no-bump", "--json")
	if err != nil {
		t.Fatalf("save --no-bump: %v output=%s", err, output)
	}
	if bumped := decodeBumpedFiles(t, output); bumped != 0 {
		t.Fatalf("bumped_files = %d, want 0 with --no-bump (%s)", bumped, output)
	}
	open, err := markdown.ParseFile(openFile)
	if err != nil {
		t.Fatal(err)
	}
	if open.Frontmatter.Updated != "2026-01-01" {
		t.Fatalf("open updated = %q, want the unchanged 2026-01-01", open.Frontmatter.Updated)
	}
	// The mirror and the auto-commit must still happen: --no-bump only disables
	// the frontmatter date advance.
	closedFile := filepath.Join(root, ".historic", ".database", "00001-topic", "wos", "01-work.md")
	closed, err := markdown.ParseFile(closedFile)
	if err != nil {
		t.Fatal(err)
	}
	if closed.Body != open.Body {
		t.Fatalf("mirror body = %q, want the edited open body %q", closed.Body, open.Body)
	}
	if closed.Frontmatter.Updated != "2026-01-01" {
		t.Fatalf("mirror updated = %q, want the unbumped 2026-01-01", closed.Frontmatter.Updated)
	}
	var commit struct {
		Data struct {
			Commit *struct {
				Committed bool `json:"committed"`
			} `json:"commit"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &commit); err != nil {
		t.Fatal(err)
	}
	if commit.Data.Commit == nil || !commit.Data.Commit.Committed {
		t.Fatalf("auto-commit did not run with --no-bump: %s", output)
	}

	// Documented semantics: the operation still rebuilds the index, so the
	// current content becomes the new hash baseline. The skipped change is
	// therefore absorbed and never bumps later — opting out is per change.
	output, err = executeCommand(t, "save", "00001", "--json")
	if err != nil {
		t.Fatalf("second save: %v", err)
	}
	if bumped := decodeBumpedFiles(t, output); bumped != 0 {
		t.Fatalf("bumped_files = %d, want 0 (change already absorbed by --no-bump)", bumped)
	}
	absorbed, err := markdown.ParseFile(openFile)
	if err != nil {
		t.Fatal(err)
	}
	if absorbed.Frontmatter.Updated != "2026-01-01" {
		t.Fatalf("updated = %q, want 2026-01-01 to stay unbumped", absorbed.Frontmatter.Updated)
	}
}

func TestSaveNoBumpHumanOutputExplainsTheSkip(t *testing.T) {
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
	editManagedBody(t, openFile, "edited body")

	output, err := executeCommand(t, "save", "00002", "--no-bump")
	if err != nil {
		t.Fatalf("save --no-bump: %v output=%s", err, output)
	}
	if !strings.Contains(output, "bump updated: skipped (--no-bump)") {
		t.Fatalf("human output does not explain the skip: %q", output)
	}
}

func TestSaveAllNoBumpSkipsEveryTopic(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Topic", "--id", "00003"); err != nil {
		t.Fatal(err)
	}
	openFile := filepath.Join(root, ".historic", "00003-topic", "wos", "01-work.md")
	writeManagedStaleFile(t, openFile)
	if _, err := executeCommand(t, "rebuild"); err != nil {
		t.Fatal(err)
	}
	editManagedBody(t, openFile, "edited body")

	output, err := executeCommand(t, "save", "--all", "--no-bump", "--json")
	if err != nil {
		t.Fatalf("save --all --no-bump: %v output=%s", err, output)
	}
	if bumped := decodeBumpedFiles(t, output); bumped != 0 {
		t.Fatalf("bumped_files = %d, want 0", bumped)
	}
	open, err := markdown.ParseFile(openFile)
	if err != nil {
		t.Fatal(err)
	}
	if open.Frontmatter.Updated != "2026-01-01" {
		t.Fatalf("open updated = %q, want 2026-01-01", open.Frontmatter.Updated)
	}
}

func TestRebuildNoBumpSkipsDateButStillReconcilesIndex(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Topic", "--id", "00004"); err != nil {
		t.Fatal(err)
	}
	openFile := filepath.Join(root, ".historic", "00004-topic", "wos", "01-work.md")
	writeManagedStaleFile(t, openFile)
	if _, err := executeCommand(t, "rebuild"); err != nil {
		t.Fatal(err)
	}
	editManagedBody(t, openFile, "edited body")
	// A second managed file is not yet indexed; rebuild must still reconcile it
	// into the manifest even though the bump pass is skipped.
	newFile := filepath.Join(root, ".historic", "00004-topic", "wos", "02-new.md")
	newDocument, err := markdown.NewDocument(domain.Frontmatter{
		ID: "00004", Title: "Second work", Status: domain.StatusProgress, Created: "2026-01-01", Updated: "2026-01-01",
	}, "second body")
	if err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(newFile, newDocument); err != nil {
		t.Fatal(err)
	}

	output, err := executeCommand(t, "rebuild", "--no-bump", "--json")
	if err != nil {
		t.Fatalf("rebuild --no-bump: %v output=%s", err, output)
	}
	if bumped := decodeBumpedFiles(t, output); bumped != 0 {
		t.Fatalf("bumped_files = %d, want 0", bumped)
	}
	open, err := markdown.ParseFile(openFile)
	if err != nil {
		t.Fatal(err)
	}
	if open.Frontmatter.Updated != "2026-01-01" {
		t.Fatalf("open updated = %q, want the unbumped 2026-01-01", open.Frontmatter.Updated)
	}
	// Legacy JSON fields stay intact.
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
	// The new file is reconciled into the topic manifest, proving the rebuild
	// itself ran while the bump pass was skipped.
	meta, err := markdown.ParseTopicMetadataFile(filepath.Join(root, ".historic", "00004-topic", markdown.MetaFilename))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, file := range meta.Files {
		if file.Path == "wos/02-new.md" {
			found = true
		}
	}
	if !found {
		t.Fatalf("manifest = %#v, want wos/02-new.md reconciled", meta.Files)
	}
	if _, err := os.Stat(newFile); err != nil {
		t.Fatal(err)
	}
}
