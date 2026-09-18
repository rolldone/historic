package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
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
	command.SetArgs([]string{"find", "missing", "--json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("empty find failed: %v", err)
	}
	var response map[string]any
	if err := json.Unmarshal(output.Bytes(), &response); err != nil || response["ok"] != true {
		t.Fatalf("response = %q, %v", output.String(), err)
	}
}
