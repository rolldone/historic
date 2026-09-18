package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCompleteCommandArchivesTopic(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	command := NewRootCommand()
	var output bytes.Buffer
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"create", "Archive Topic", "--id", "00001"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	command = NewRootCommand()
	output.Reset()
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"complete", "00001", "--json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("complete: %v", err)
	}
	var response struct {
		OK   bool `json:"ok"`
		Data struct {
			Archived bool   `json:"archived"`
			Path     string `json:"path"`
		} `json:"data"`
	}
	if err := json.Unmarshal(output.Bytes(), &response); err != nil || !response.OK || !response.Data.Archived || response.Data.Path != ".histories/.database/00001-archive-topic" {
		t.Fatalf("response = %q, %v", output.String(), err)
	}
	if _, err := os.Stat(filepath.Join(root, ".histories", ".database", "00001-archive-topic")); err != nil {
		t.Fatal(err)
	}
}
