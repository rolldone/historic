package cmd

import (
	"bytes"
	"testing"
)

func TestLifecycleCommandUpdatesStatus(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	command := NewRootCommand()
	var output bytes.Buffer
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"create", "Topic", "--id", "00001"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestLifecycleCommandRejectsInvalidID(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	command := NewRootCommand()
	command.SetArgs([]string{"init"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	command = NewRootCommand()
	command.SetArgs([]string{"progress", "14"})
	if err := command.Execute(); err == nil {
		t.Fatal("invalid ID accepted")
	}
}
