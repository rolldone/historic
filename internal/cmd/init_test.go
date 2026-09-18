package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCommandCreatesWorkspace(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	command := NewRootCommand()
	var output bytes.Buffer
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"init"})

	if err := command.Execute(); err != nil {
		t.Fatalf("execute init: %v", err)
	}
	for _, path := range []string{
		filepath.Join(root, ".historic"),
		filepath.Join(root, ".historic", ".database"),
		filepath.Join(root, ".historic", ".index.sqlite"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("stat %s: %v", path, err)
		}
	}
	if !strings.Contains(output.String(), "Initialized Historic") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestInitCommandIsIdempotent(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	for i := 0; i < 2; i++ {
		command := NewRootCommand()
		var output bytes.Buffer
		ConfigureOutput(command, &output, &output)
		command.SetArgs([]string{"init"})
		if err := command.Execute(); err != nil {
			t.Fatalf("execute init %d: %v", i+1, err)
		}
	}
}
