package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveMirrorsAllOpenTopicsWithoutClosingThem(t *testing.T) {
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
	command.SetArgs([]string{"create", "Topic", "--id", "00001"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	openTopic := filepath.Join(root, ".historic", "00001-topic")
	if err := os.WriteFile(filepath.Join(openTopic, "note.md"), []byte("snapshot data\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	command = NewRootCommand()
	output.Reset()
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"save", "--all", "--json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("save: %v", err)
	}
	var response struct {
		OK   bool `json:"ok"`
		Data struct {
			Saved       int `json:"saved"`
			FailedCount int `json:"failed_count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(output.Bytes(), &response); err != nil || !response.OK || response.Data.Saved != 1 || response.Data.FailedCount != 0 {
		t.Fatalf("response = %q, err=%v", output.String(), err)
	}
	closedNote := filepath.Join(root, ".historic", ".database", "00001-topic", "note.md")
	if content, err := os.ReadFile(closedNote); err != nil || string(content) != "snapshot data\n" {
		t.Fatalf("snapshot = %q, err=%v", content, err)
	}
	if _, err := os.Stat(openTopic); err != nil {
		t.Fatalf("open topic removed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(openTopic, "note.md"), []byte("updated data\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	command = NewRootCommand()
	output.Reset()
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"save", "--all"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if content, err := os.ReadFile(closedNote); err != nil || string(content) != "updated data\n" {
		t.Fatalf("updated snapshot = %q, err=%v", content, err)
	}
}

func TestSaveRejectsAllWithTopicID(t *testing.T) {
	_, err := executeCommand(t, "save", "--all", "00001")
	if err == nil {
		t.Fatal("expected --all and topic ID to be rejected")
	}
}

func TestSaveJSONWithoutIDRequiresAll(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	output, err := executeCommand(t, "save", "--json")
	if err == nil || !strings.Contains(output, "topic ID or --all is required") {
		t.Fatalf("output=%q err=%v", output, err)
	}
}

func TestSaveRejectsMessageFlag(t *testing.T) {
	command := NewRootCommand()
	command.SetArgs([]string{"save", "-m", "message"})
	if err := command.Execute(); err == nil {
		t.Fatal("expected unknown flag error")
	}
}

func TestSaveSingleTopicMirrorsOnlyThatTopic(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "First", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Second", "--id", "00002"); err != nil {
		t.Fatal(err)
	}
	output, err := executeCommand(t, "save", "00001", "--json")
	if err != nil {
		t.Fatalf("save 00001: %v", err)
	}
	var response struct {
		OK   bool `json:"ok"`
		Data struct {
			Total int `json:"total"`
			Saved int `json:"saved"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil || !response.OK || response.Data.Total != 1 || response.Data.Saved != 1 {
		t.Fatalf("response = %q err=%v", output, err)
	}
	if _, err := os.Stat(filepath.Join(root, ".historic", ".database", "00001-first")); err != nil {
		t.Fatalf("saved topic missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".historic", ".database", "00002-second")); !os.IsNotExist(err) {
		t.Fatalf("unsaved topic was mirrored: %v", err)
	}
}

func TestSaveSingleTopicRejectsMissingTopic(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	output, err := executeCommand(t, "save", "00099", "--json")
	if err == nil {
		t.Fatal("expected save of missing topic to fail")
	}
	var response struct {
		OK   bool `json:"ok"`
		Data struct {
			FailedCount int `json:"failed_count"`
		} `json:"data"`
	}
	if unmarshalErr := json.Unmarshal([]byte(output), &response); unmarshalErr != nil || response.OK || response.Data.FailedCount != 1 {
		t.Fatalf("response = %q err=%v", output, unmarshalErr)
	}
}
