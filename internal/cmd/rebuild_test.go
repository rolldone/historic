package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRebuildCommandJSON(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	workspace := filepath.Join(root, ".historic", "00001-topic")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "_meta.md"), []byte("---\nid: 00001\ntitle: Topic\nstatus: progress\ncreated: 2026-09-18\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	command := NewRootCommand()
	var output bytes.Buffer
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"rebuild", "--json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	var response struct {
		Command string `json:"command"`
		OK      bool   `json:"ok"`
		Data    struct {
			Records int `json:"records"`
		} `json:"data"`
	}
	if err := json.Unmarshal(output.Bytes(), &response); err != nil || !response.OK || response.Command != "rebuild" || response.Data.Records != 1 {
		t.Fatalf("response = %q, %v", output.String(), err)
	}
}
