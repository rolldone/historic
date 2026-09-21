package cmd

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"historic/internal/domain"
)

func TestDeleteCommandRequiresExplicitConfirmationAndReportsJSON(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Delete CLI", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "add", "wos/task", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "delete", "wos/01-task.md"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("unconfirmed delete error = %v", err)
	}
	output, err := executeCommand(t, "delete", "wos/01-task.md", "--yes", "--json")
	if err != nil {
		t.Fatalf("confirmed delete: %v", err)
	}
	var response struct {
		Command string `json:"command"`
		OK      bool   `json:"ok"`
		Data    struct {
			Storage string `json:"storage"`
			Target  string `json:"target"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil || !response.OK || response.Command != "delete" || response.Data.Storage != "open" || response.Data.Target != "file" {
		t.Fatalf("delete response = %q err=%v", output, err)
	}
	if _, err := os.Stat(filepath.Join(root, ".historic", "00001-delete-cli", "wos", "01-task.md")); !os.IsNotExist(err) {
		t.Fatalf("deleted file exists: %v", err)
	}
}

func TestPurgeCommandRequiresIDConfirmation(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Purge CLI", "--id", "00002"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "purge", "00002", "--yes"); err == nil || !strings.Contains(err.Error(), "--confirm 00002") {
		t.Fatalf("missing purge confirmation error = %v", err)
	}
	output, err := executeCommand(t, "purge", "00002", "--yes", "--confirm", "00002", "--json")
	if err != nil {
		t.Fatalf("confirmed purge: %v", err)
	}
	if !jsonHasOK(output) {
		t.Fatalf("purge response = %q", output)
	}
	if _, err := os.Stat(filepath.Join(root, ".historic", "00002-purge-cli")); !os.IsNotExist(err) {
		t.Fatalf("purged topic exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".historic", ".database", ".git")); err != nil {
		t.Fatalf("internal git was affected by purge: %v", err)
	}
}
