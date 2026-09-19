package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFileStatusCommandUpdatesSingleFile(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Topic", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "add", "wos/19-fts5-index", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	command := NewRootCommand()
	var output bytes.Buffer
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"status", "19-fts5-index.md", "complete", "--json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("status: %v", err)
	}
	var response struct {
		Command string `json:"command"`
		OK      bool   `json:"ok"`
		Data    struct {
			Current string `json:"current"`
			Updated string `json:"updated"`
			Path    string `json:"path"`
		} `json:"data"`
		Error any `json:"error"`
	}
	if err := json.Unmarshal(output.Bytes(), &response); err != nil || !response.OK || response.Command != "status" || response.Data.Current != "complete" || response.Data.Updated == "" || response.Error != nil {
		t.Fatalf("response = %q, %v", output.String(), err)
	}
	if response.Data.Path != ".historic/00001-topic/wos/19-fts5-index.md" {
		t.Fatalf("path = %q", response.Data.Path)
	}
	if _, err := os.Stat(filepath.Join(root, ".historic", "00001-topic")); err != nil {
		t.Fatalf("topic moved: %v", err)
	}
}

func TestFileStatusCommandRejectsTraversalAsJSON(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	command := NewRootCommand()
	var output, errorOutput bytes.Buffer
	ConfigureOutput(command, &output, &errorOutput)
	command.SetArgs([]string{"status", "../escape.md", "complete", "--json"})
	if err := command.Execute(); err == nil {
		t.Fatal("path traversal accepted")
	}
	var response map[string]any
	if err := json.Unmarshal(output.Bytes(), &response); err != nil || response["ok"] != false || response["command"] != "status" {
		t.Fatalf("response = %q, %v", output.String(), err)
	}
	if errorOutput.Len() != 0 {
		t.Fatalf("JSON error duplicated on stderr: %q", errorOutput.String())
	}
}
