package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSyncMetaBatchCommandJSONAndEmptyWorkspace(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	command := NewRootCommand()
	var output bytes.Buffer
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"sync-meta", "--json"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	var response struct {
		Command string `json:"command"`
		OK      bool   `json:"ok"`
		Data    struct {
			Topics []any `json:"topics"`
			Total  int   `json:"total"`
			Errors int   `json:"errors"`
		} `json:"data"`
	}
	if err := json.Unmarshal(output.Bytes(), &response); err != nil || response.Command != "sync-meta" || !response.OK || response.Data.Total != 0 || response.Data.Errors != 0 || len(response.Data.Topics) != 0 {
		t.Fatalf("empty batch = %q, %v", output.String(), err)
	}
}

func TestSyncMetaBatchCommandDoesNotProcessArchive(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Active", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(root, ".historic", ".database", "00002-archived")
	if err := os.MkdirAll(archive, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(archive, "_meta.yaml"), []byte("id: 00002\ntitle: Archived\nstatus: complete\ncreated: 2026-09-19\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "sync-meta", "--json"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(archive); err != nil {
		t.Fatal(err)
	}
}
