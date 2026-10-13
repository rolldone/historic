package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestListAndShowCommandsJSON(t *testing.T) {
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
		Data    []struct {
			Storage string `json:"storage"`
		} `json:"data"`
	}
	if err := json.Unmarshal(output.Bytes(), &show); err != nil || !show.OK || show.Command != "show" || len(show.Data) != 1 || show.Data[0].Storage != "open" {
		t.Fatalf("show response = %q, %v", output.String(), err)
	}
	if _, err := executeCommand(t, "close", "00001"); err != nil {
		t.Fatal(err)
	}
	closedOutput, err := executeCommand(t, "show", "00001")
	if err != nil || !strings.Contains(closedOutput, "[CLOSED]") {
		t.Fatalf("show closed fallback output=%q err=%v", closedOutput, err)
	}
	jsonOutput, err := executeCommand(t, "show", "00001", "--json")
	var closedShow struct {
		Data []struct {
			Storage string `json:"storage"`
		} `json:"data"`
	}
	if err != nil || json.Unmarshal([]byte(jsonOutput), &closedShow) != nil || len(closedShow.Data) != 1 || closedShow.Data[0].Storage != "closed" {
		t.Fatalf("closed show JSON=%q err=%v", jsonOutput, err)
	}
}

func TestShowDisplaysBothMirroredVersions(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Topic", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "save", "--all"); err != nil {
		t.Fatal(err)
	}
	output, err := executeCommand(t, "show", "00001")
	if err != nil || !strings.Contains(output, "[OPEN]") || !strings.Contains(output, "[CLOSED]") {
		t.Fatalf("show output=%q err=%v", output, err)
	}
	jsonOutput, err := executeCommand(t, "show", "00001", "--json")
	var response struct {
		Data []struct {
			Storage string `json:"storage"`
		} `json:"data"`
	}
	if err != nil || json.Unmarshal([]byte(jsonOutput), &response) != nil || len(response.Data) != 2 || response.Data[0].Storage != "open" || response.Data[1].Storage != "closed" {
		t.Fatalf("show JSON=%q err=%v", jsonOutput, err)
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
