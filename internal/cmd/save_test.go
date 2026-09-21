package cmd

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestSaveCommandCreatesCommitAndCleanSaveIsSafe(t *testing.T) {
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
	command.SetArgs([]string{"complete", "00001"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	command = NewRootCommand()
	output.Reset()
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"close", "00001"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	command = NewRootCommand()
	output.Reset()
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"save", "-m", "initial", "--json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("save: %v", err)
	}
	var response struct {
		OK   bool `json:"ok"`
		Data struct {
			Committed bool   `json:"committed"`
			Commit    string `json:"commit"`
		} `json:"data"`
	}
	if err := json.Unmarshal(output.Bytes(), &response); err != nil || !response.OK || !response.Data.Committed || len(response.Data.Commit) < 7 {
		t.Fatalf("response = %q, %v", output.String(), err)
	}
}
