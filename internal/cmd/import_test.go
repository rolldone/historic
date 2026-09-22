package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestImportCommandCopiesArchivedTopic(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	command := NewRootCommand()
	var output bytes.Buffer
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"init"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(root, ".historic", ".database", "00001-topic")
	if err := os.MkdirAll(archive, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(archive, "_meta.yaml"), []byte("id: 00001\ntitle: Topic\nstatus: complete\ncreated: 2026-09-18\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	command = NewRootCommand()
	output.Reset()
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"import", "00001", "--json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("import: %v", err)
	}
	var response struct {
		OK   bool `json:"ok"`
		Data struct {
			Storage string `json:"storage"`
		} `json:"data"`
	}
	if err := json.Unmarshal(output.Bytes(), &response); err != nil || !response.OK || response.Data.Storage != "open" {
		t.Fatalf("response = %q, %v", output.String(), err)
	}
	if _, err := os.Stat(filepath.Join(root, ".historic", "00001-topic", "_meta.yaml")); err != nil {
		t.Fatal(err)
	}
}
