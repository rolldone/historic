package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"historic/internal/cmd"
)

func TestJSONCommandErrorDoesNotWriteStderr(t *testing.T) {
	root := t.TempDir()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	command := cmd.NewRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.ConfigureOutput(command, &stdout, &stderr)
	command.SetArgs([]string{"init"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	command = cmd.NewRootCommand()
	cmd.ConfigureOutput(command, &stdout, &stderr)
	command.SetArgs([]string{"show", "00001", "--json"})
	err := command.Execute()
	if err == nil {
		t.Fatal("missing topic unexpectedly succeeded")
	}
	var response struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if decodeErr := json.Unmarshal(stdout.Bytes(), &response); decodeErr != nil {
		t.Fatalf("stdout is not JSON: %v (%q)", decodeErr, stdout.String())
	}
	if response.OK || !strings.Contains(response.Error, "topic not found") {
		t.Fatalf("response = %#v", response)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	var silent interface{ Silent() bool }
	if !errors.As(err, &silent) || !silent.Silent() {
		t.Fatalf("error = %T %v, want SilentError", err, err)
	}
}
