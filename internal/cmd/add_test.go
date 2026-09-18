package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAddCommandCreatesEntry(t *testing.T) {
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
	command.SetArgs([]string{"add", "wos/scaffold", "--id", "00001", "--json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("execute add: %v", err)
	}
	var result struct {
		OK   bool `json:"ok"`
		Data struct {
			File string `json:"file"`
		} `json:"data"`
	}
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.OK || result.Data.File != ".histories/00001-topic/wos/01-scaffold.md" {
		t.Fatalf("response = %#v", result)
	}
	if _, err := os.Stat(filepath.Join(root, result.Data.File)); err != nil {
		t.Fatal(err)
	}
}

func TestAddCommandRejectsTraversal(t *testing.T) {
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
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"add", "../escape", "--id", "00001"})
	if err := command.Execute(); err == nil {
		t.Fatal("traversal accepted")
	}
}
