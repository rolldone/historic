package cmd

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestLogAndDiffCommands(t *testing.T) {
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
	command.SetArgs([]string{"save", "-m", "topic snapshot"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	command = NewRootCommand()
	output.Reset()
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"diff", "00001", "--json"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	var diff struct {
		OK   bool `json:"ok"`
		Data struct {
			Empty bool `json:"empty"`
		} `json:"data"`
	}
	if err := json.Unmarshal(output.Bytes(), &diff); err != nil || !diff.OK || !diff.Data.Empty {
		t.Fatalf("diff = %q, %v", output.String(), err)
	}
	command = NewRootCommand()
	output.Reset()
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"log", "00001", "--json"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	var log struct {
		OK   bool `json:"ok"`
		Data []struct {
			Message string `json:"message"`
		} `json:"data"`
	}
	if err := json.Unmarshal(output.Bytes(), &log); err != nil || !log.OK || len(log.Data) == 0 || log.Data[0].Message != "topic snapshot" {
		t.Fatalf("log = %q, %v", output.String(), err)
	}
}
