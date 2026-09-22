package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommandHelp(t *testing.T) {
	root := NewRootCommand()
	var out bytes.Buffer
	ConfigureOutput(root, &out, &out)
	root.SetArgs([]string{"--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("execute help: %v", err)
	}
	if !strings.Contains(out.String(), "Usage:") {
		t.Fatalf("help output does not contain usage: %q", out.String())
	}
}

func TestVersionCommand(t *testing.T) {
	root := NewRootCommand()
	var out bytes.Buffer
	ConfigureOutput(root, &out, &out)
	root.SetArgs([]string{"version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("execute version: %v", err)
	}
	if got := strings.TrimSpace(out.String()); !strings.Contains(got, Version) || !strings.Contains(got, "workspace format") {
		t.Fatalf("version output = %q, want version with workspace info", got)
	}
}
