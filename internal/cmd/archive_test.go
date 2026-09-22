package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCompleteCommandOnlyChangesWorkStatus(t *testing.T) {
	t.Skip("topic status is not authoritative; use historic status on a member file")
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
	command.SetArgs([]string{"complete", "00001", "--json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("complete: %v", err)
	}
	var response struct {
		OK   bool `json:"ok"`
		Data struct {
			Current  string `json:"current"`
			Storage  string `json:"storage"`
			Path     string `json:"path"`
			Archived bool   `json:"archived"`
		} `json:"data"`
	}
	if err := json.Unmarshal(output.Bytes(), &response); err != nil || !response.OK || response.Data.Current != "complete" || response.Data.Storage != "open" || response.Data.Archived || response.Data.Path != ".historic/00001-topic" {
		t.Fatalf("response = %q, %v", output.String(), err)
	}
	if _, err := os.Stat(filepath.Join(root, ".historic", "00001-topic")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".historic", ".database", "00001-topic")); !os.IsNotExist(err) {
		t.Fatalf("closed topic exists: %v", err)
	}
}
