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
	command.SetArgs([]string{"find", "needle", "--active", "--json"})
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
