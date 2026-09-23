package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"historic/internal/config"
)

func TestFindRejectsBeforeRebuildWithStableJSONError(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if err := os.MkdirAll(filepath.Join(root, ".historic", ".database", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	before := filesystemSnapshot(t, root)
	command := NewRootCommand()
	var output, stderr bytes.Buffer
	ConfigureOutput(command, &output, &stderr)
	command.SetArgs([]string{"find", "needle", "--json"})
	err := command.Execute()
	if err == nil {
		t.Fatal("find unexpectedly succeeded before rebuild")
	}
	var response struct {
		Command string `json:"command"`
		OK      bool   `json:"ok"`
		Data    any    `json:"data"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if decodeErr := json.Unmarshal(output.Bytes(), &response); decodeErr != nil {
		t.Fatalf("invalid JSON: %v (%q)", decodeErr, output.String())
	}
	if response.Command != "find" || response.OK || response.Data != nil || response.Error.Code != string(config.ReadinessMissingIndex) || response.Error.Message != "Jalankan historic rebuild terlebih dahulu." {
		t.Fatalf("response = %+v", response)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	var silent interface{ Silent() bool }
	if !errors.As(err, &silent) || !silent.Silent() {
		t.Fatalf("error = %T, want SilentError", err)
	}
	if after := filesystemSnapshot(t, root); before != after {
		t.Fatalf("find readiness changed filesystem: before=%q after=%q", before, after)
	}
}

func filesystemSnapshot(t *testing.T, root string) string {
	t.Helper()
	entries := make([]string, 0)
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		entries = append(entries, path+":"+info.Mode().String())
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	result := ""
	for _, entry := range entries {
		result += entry + "\n"
	}
	return result
}
