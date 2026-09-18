package cmd

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestListAndShowCommandsJSON(t *testing.T) {
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
	command.SetArgs([]string{"list", "--json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("list: %v", err)
	}
	var list struct {
		Command string `json:"command"`
		OK      bool   `json:"ok"`
		Data    []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(output.Bytes(), &list); err != nil || !list.OK || list.Command != "list" || len(list.Data) != 1 {
		t.Fatalf("list response = %q, %v", output.String(), err)
	}
	command = NewRootCommand()
	output.Reset()
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"show", "00001", "--json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("show: %v", err)
	}
	var show struct {
		Command string `json:"command"`
		OK      bool   `json:"ok"`
	}
	if err := json.Unmarshal(output.Bytes(), &show); err != nil || !show.OK || show.Command != "show" {
		t.Fatalf("show response = %q, %v", output.String(), err)
	}
}

func TestShowCommandJSONMissingTopic(t *testing.T) {
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
	command.SetArgs([]string{"show", "00099", "--json"})
	if err := command.Execute(); err == nil {
		t.Fatal("missing topic succeeded")
	}
	var response map[string]any
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatalf("invalid JSON: %v; output=%q", err, output.String())
	}
	if response["ok"] != false {
		t.Fatalf("response = %#v", response)
	}
}
