package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindCommandJSONAndFilters(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	command := NewRootCommand()
	var output bytes.Buffer
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"create", "Topic", "--id", "00001"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(root, ".historic", "00001-topic", "note.md")
	if err := os.WriteFile(entry, []byte("---\nid: 00001\ntitle: Search Note\nstatus: progress\ncreated: 2026-09-18\n---\nneedle in body\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	command = NewRootCommand()
	output.Reset()
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"rebuild"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	command = NewRootCommand()
	output.Reset()
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"find", "needle", "--open", "--json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("find: %v", err)
	}
	var response struct {
		Command string `json:"command"`
		OK      bool   `json:"ok"`
		Data    []struct {
			Path string `json:"path"`
		} `json:"data"`
	}
	if err := json.Unmarshal(output.Bytes(), &response); err != nil || !response.OK || response.Command != "find" || len(response.Data) != 1 {
		t.Fatalf("response = %q, %v", output.String(), err)
	}
	if response.Data[0].Path != ".historic/00001-topic/note.md" {
		t.Fatalf("path = %q", response.Data[0].Path)
	}
	if strings.Contains(output.String(), "\x1b[") {
		t.Fatalf("JSON output contains terminal formatting: %q", output.String())
	}
}

func TestFindCommandEmptyResultSucceeds(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	command := NewRootCommand()
	var output bytes.Buffer
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"init"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	command = NewRootCommand()
	output.Reset()
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"rebuild"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	command = NewRootCommand()
	output.Reset()
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"find", "missing", "--json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("empty find failed: %v", err)
	}
	var response map[string]any
	if err := json.Unmarshal(output.Bytes(), &response); err != nil || response["ok"] != true {
		t.Fatalf("response = %q, %v", output.String(), err)
	}
}

func TestFindHumanOutputHighlightsMatches(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if output, err := executeCommand(t, "init"); err != nil || output == "" {
		t.Fatalf("init output=%q err=%v", output, err)
	}
	if _, err := executeCommand(t, "create", "Highlight Topic", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "add", "note", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".historic", "00001-highlight-topic", "note.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content = append(content, []byte("\nSearch keyword\n")...)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "rebuild"); err != nil {
		t.Fatal(err)
	}
	output, err := executeCommand(t, "find", "search")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "\x1b[1;33mSearch\x1b[0m") {
		t.Fatalf("human output missing highlight: %q", output)
	}
}

func TestFindCommandFTSJSONSmokeCanonicalMetadataAndStorage(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Canonical Search Topic", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "add", "weighted-note", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".historic", "00001-canonical-search-topic", "weighted-note.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content = append(content, []byte("\nfilter-token in the body\n")...)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "close", "00001"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "rebuild"); err != nil {
		t.Fatal(err)
	}

	output, err := executeCommand(t, "find", "filter-token", "--closed", "--type", "historic_file", "--json")
	if err != nil {
		t.Fatalf("closed JSON search: %v", err)
	}
	var response struct {
		Command string `json:"command"`
		OK      bool   `json:"ok"`
		Data    []struct {
			Type      string   `json:"type"`
			TopicID   string   `json:"topic_id"`
			Storage   string   `json:"storage"`
			MatchedIn []string `json:"matched_in"`
			Snippet   string   `json:"snippet"`
			Topic     struct {
				ID      string `json:"id"`
				Storage string `json:"storage"`
			} `json:"topic"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("JSON output = %q: %v", output, err)
	}
	if !response.OK || response.Command != "find" || len(response.Data) != 1 {
		t.Fatalf("unexpected JSON response = %q", output)
	}
	result := response.Data[0]
	if result.Type != "historic_file" || result.TopicID != "00001" || result.Storage != "closed" || result.Topic.ID != "00001" || result.Topic.Storage != "closed" || result.Snippet == "" || !containsString(result.MatchedIn, "content") {
		t.Fatalf("incomplete closed result = %+v", result)
	}
	human, err := executeCommand(t, "find", "filter-token", "--closed")
	if err != nil || !strings.Contains(human, "[CLOSED]") {
		t.Fatalf("human closed label output=%q err=%v", human, err)
	}
	if _, err := executeCommand(t, "find", "filter-token", "--open", "--closed"); err == nil || !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("conflicting storage filters error = %v", err)
	}
	if _, err := executeCommand(t, "find", "filter-token", "--active"); err == nil {
		t.Fatal("deprecated --active flag was accepted")
	}
	if _, err := executeCommand(t, "find", "filter-token", "--updated-after", "not-a-date"); err == nil || !strings.Contains(err.Error(), "YYYY-MM-DD") {
		t.Fatalf("invalid date error = %v", err)
	}
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
