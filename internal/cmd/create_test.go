package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"historic/internal/domain"
	"historic/internal/markdown"
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

func TestCreateCommandExtendsParentJSON(t *testing.T) {
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
	command.SetArgs([]string{"create", "Parent", "--id", "1791851149892"})
	if err := command.Execute(); err != nil {
		t.Fatalf("create parent: %v", err)
	}

	command = NewRootCommand()
	output.Reset()
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"create", "Lanjutan", "--id", "1792889892200", "--extends", "1791851149892", "--json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("create child: %v", err)
	}
	var response struct {
		OK   bool `json:"ok"`
		Data struct {
			ID      string `json:"id"`
			Extends string `json:"extends"`
		} `json:"data"`
	}
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatalf("decode output: %v (%q)", err, output.String())
	}
	if !response.OK || response.Data.ID != "1792889892200" || response.Data.Extends != "1791851149892" {
		t.Fatalf("response = %#v", response)
	}
	metadata, err := markdown.ParseTopicMetadataFile(filepath.Join(root, ".historic", "1792889892200-lanjutan", "_meta.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Extends != domain.TopicID("1791851149892") {
		t.Fatalf("metadata extends = %q", metadata.Extends)
	}
}

func TestCreateCommandRejectsRepeatedOrEmptyExtends(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	command := NewRootCommand()
	var output bytes.Buffer
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"init"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{
		{"create", "Repeated", "--extends", "1791851149892", "--extends", "1792889892200"},
		{"create", "Empty", "--extends", ""},
	} {
		command = NewRootCommand()
		output.Reset()
		ConfigureOutput(command, &output, &output)
		command.SetArgs(args)
		if err := command.Execute(); err == nil {
			t.Fatalf("%v accepted", args)
		}
	}
	if entries, err := os.ReadDir(filepath.Join(root, ".historic")); err != nil {
		t.Fatal(err)
	} else {
		for _, entry := range entries {
			if entry.Name() == "repeated" || entry.Name() == "empty" {
				t.Fatalf("partial topic directory %q created", entry.Name())
			}
		}
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
