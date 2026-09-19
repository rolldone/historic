package cmd

import (
	"bytes"
	"testing"
)

func TestSearchCommandRejectsNonInteractiveTerminal(t *testing.T) {
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
	command.SetArgs([]string{"search"})
	if err := command.Execute(); err == nil {
		t.Fatal("non-interactive search succeeded")
	}
}

func TestSearchCommandRejectsJSON(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	command := NewRootCommand()
	command.SetArgs([]string{"search", "--json"})
	if err := command.Execute(); err == nil {
		t.Fatal("search --json succeeded")
	}
}
