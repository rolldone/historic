package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLifecycleCommandUpdatesStatus(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	command := NewRootCommand()
	var output bytes.Buffer
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"create", "Topic", "--id", "00001"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	command = NewRootCommand()
	output.Reset()
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"progress", "00001", "--json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("progress: %v", err)
	}
	var response struct {
		OK   bool `json:"ok"`
		Data struct {
			Previous string `json:"previous"`
			Current  string `json:"current"`
		} `json:"data"`
	}
	if err := json.Unmarshal(output.Bytes(), &response); err != nil || !response.OK || response.Data.Previous != "create" || response.Data.Current != "progress" {
		t.Fatalf("response = %q, %v", output.String(), err)
	}
	metadata, err := os.ReadFile(filepath.Join(root, ".historic", "00001-topic", "_meta.md"))
	if err != nil || !bytes.Contains(metadata, []byte("status: progress")) {
		t.Fatalf("metadata = %q err=%v", metadata, err)
	}
}

func TestLifecycleCommandRejectsInvalidID(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	command := NewRootCommand()
	command.SetArgs([]string{"init"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	command = NewRootCommand()
	command.SetArgs([]string{"progress", "14"})
	if err := command.Execute(); err == nil {
		t.Fatal("invalid ID accepted")
	}
}
