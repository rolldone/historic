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
	command.SetArgs([]string{"add", "wos/Remove stale Cost Dashboard breakdowns", "--id", "00001", "--json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("execute add: %v", err)
	}
	var result struct {
		OK   bool `json:"ok"`
		Data struct {
			ID    string `json:"id"`
			File  string `json:"file"`
			Title string `json:"title"`
		} `json:"data"`
	}
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	wantFile := ".historic/00001-topic/wos/01-remove-stale-cost-dashboard-breakdowns.md"
	if !result.OK || result.Data.ID != "00001" || result.Data.File != wantFile || result.Data.Title != "01 remove stale cost dashboard breakdowns" {
		t.Fatalf("response = %#v", result)
	}
	if _, err := os.Stat(filepath.Join(root, result.Data.File)); err != nil {
		t.Fatal(err)
	}
	meta, err := os.ReadFile(filepath.Join(root, ".historic/00001-topic/_meta.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(meta, []byte("./wos/01-remove-stale-cost-dashboard-breakdowns.md")) {
		t.Fatalf("metadata missing canonical path: %s", meta)
	}
}

func TestAddCommandCreatesPlainSluggedEntry(t *testing.T) {
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
	command.SetArgs([]string{"add", "Remove stale Cost Dashboard breakdowns", "--id", "00001", "--json"})
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
	wantFile := ".historic/00001-topic/remove-stale-cost-dashboard-breakdowns.md"
	if !result.OK || result.Data.File != wantFile {
		t.Fatalf("response = %#v", result)
	}
	if _, err := os.Stat(filepath.Join(root, wantFile)); err != nil {
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
