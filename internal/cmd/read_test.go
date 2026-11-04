package cmd

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadCommandOpenFile(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Read Topic", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	content := "---\nid: 00001\ntitle: Read Note\ndescription: read me\nstatus: progress\ncreated: 2026-11-01\n---\nfirst line\nsecond line\n"
	path := filepath.Join(root, ".historic", "00001-read-topic", "note.md")
	writeManagedNote(t, path, content)
	if _, err := executeCommand(t, "rebuild"); err != nil {
		t.Fatal(err)
	}

	output, err := executeCommand(t, "read", ".historic/00001-read-topic/note.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if output != content {
		t.Fatalf("read output = %q, want %q", output, content)
	}
}

func TestReadCommandHeadAndJSON(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Read JSON", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	content := "---\nid: 00001\ntitle: Read JSON Note\ndescription: json body\nstatus: draft\ncreated: 2026-11-01\ntags:\n  - alpha\n---\nline one\nline two\nline three\n"
	writeManagedNote(t, filepath.Join(root, ".historic", "00001-read-json", "note.md"), content)
	if _, err := executeCommand(t, "rebuild"); err != nil {
		t.Fatal(err)
	}

	head, err := executeCommand(t, "read", ".historic/00001-read-json/note.md", "--head", "8")
	if err != nil {
		t.Fatal(err)
	}
	if want := "---\nid: 00001\ntitle: Read JSON Note\ndescription: json body\nstatus: draft\ncreated: 2026-11-01\ntags:\n  - alpha\n"; head != want {
		t.Fatalf("head output = %q, want %q", head, want)
	}

	output, err := executeCommand(t, "read", ".historic/00001-read-json/note.md", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var response struct {
		Command string `json:"command"`
		OK      bool   `json:"ok"`
		Data    struct {
			Path        string `json:"path"`
			Frontmatter struct {
				ID          string   `json:"id"`
				Title       string   `json:"title"`
				Description string   `json:"description"`
				Status      string   `json:"status"`
				Created     string   `json:"created"`
				Tags        []string `json:"tags"`
			} `json:"frontmatter"`
			Body string `json:"body"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("json output = %q: %v", output, err)
	}
	if !response.OK || response.Command != "read" || response.Data.Path != ".historic/00001-read-json/note.md" {
		t.Fatalf("response = %+v", response)
	}
	if response.Data.Frontmatter.ID != "00001" || response.Data.Frontmatter.Title != "Read JSON Note" || response.Data.Frontmatter.Status != "draft" || response.Data.Frontmatter.Created != "2026-11-01" || len(response.Data.Frontmatter.Tags) != 1 {
		t.Fatalf("frontmatter = %+v", response.Data.Frontmatter)
	}
	if response.Data.Body != "line one\nline two\nline three\n" {
		t.Fatalf("body = %q", response.Data.Body)
	}
}

func TestReadCommandClosedStorageAndAmbiguity(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Closed Read", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	closedContent := "---\nid: 00001\ntitle: Closed Note\nstatus: complete\ncreated: 2026-11-01\n---\nclosed body\n"
	writeManagedNote(t, filepath.Join(root, ".historic", "00001-closed-read", "note.md"), closedContent)
	if _, err := executeCommand(t, "close", "00001"); err != nil {
		t.Fatal(err)
	}

	output, err := executeCommand(t, "read", ".historic/.database/00001-closed-read/note.md")
	if err != nil {
		t.Fatalf("read closed: %v", err)
	}
	if output != closedContent {
		t.Fatalf("closed read = %q", output)
	}

	// A second note with the same basename in open storage makes the short
	// path genuinely ambiguous across both storage roots.
	if _, err := executeCommand(t, "create", "Open Twin", "--id", "00002"); err != nil {
		t.Fatal(err)
	}
	writeManagedNote(t, filepath.Join(root, ".historic", "00002-open-twin", "note.md"), "---\nid: 00002\ntitle: Open Twin Note\nstatus: draft\ncreated: 2026-11-01\n---\nopen body\n")
	if _, err := executeCommand(t, "rebuild"); err != nil {
		t.Fatal(err)
	}

	if _, err := executeCommand(t, "read", "note.md"); err == nil || !strings.Contains(err.Error(), "matches multiple") {
		t.Fatalf("ambiguous resolution error = %v", err)
	}
	if _, err := executeCommand(t, "read", "../outside.md"); err == nil {
		t.Fatal("path outside the workspace was accepted")
	}
	if _, err := executeCommand(t, "read", ".historic/00001-closed-read/note.md", "--head", "-1"); err == nil {
		t.Fatal("negative --head was accepted")
	}
}
