package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateCommandCreatesTopicAndJSON(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	command := NewRootCommand()
	var output bytes.Buffer
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"init"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	output.Reset()
	command = NewRootCommand()
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"create", "Admin Dashboard", "--id", "00014", "--json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("execute create: %v", err)
	}
	var response struct {
		Command string `json:"command"`
		OK      bool   `json:"ok"`
		Data    struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatalf("decode output: %v (%q)", err, output.String())
	}
	if !response.OK || response.Command != "create" || response.Data.ID != "00014" {
		t.Fatalf("response = %#v", response)
	}
	if _, err := os.Stat(filepath.Join(root, ".historic", "00014-admin-dashboard", "_meta.yaml")); err != nil {
		t.Fatalf("metadata: %v", err)
	}
}

func TestCreateCommandRejectsDuplicateWithoutPartialDirectory(t *testing.T) {
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
	command.SetArgs([]string{"create", "First", "--id", "00001"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	command = NewRootCommand()
	output.Reset()
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"create", "Second", "--id", "00001"})
	if err := command.Execute(); err == nil {
		t.Fatal("duplicate create succeeded")
	}
	if _, err := os.Stat(filepath.Join(root, ".historic", "00001-second")); !os.IsNotExist(err) {
		t.Fatalf("partial duplicate directory exists: %v", err)
	}
}
