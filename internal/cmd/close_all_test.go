package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCloseAllCommandJSONAndHumanOutput(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "First", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Second", "--id", "00002"); err != nil {
		t.Fatal(err)
	}

	output, err := executeCommand(t, "close", "all", "--json")
	if err != nil {
		t.Fatalf("close all: %v output=%s", err, output)
	}
	var response struct {
		Command string `json:"command"`
		OK      bool   `json:"ok"`
		Data    struct {
			Succeeded []struct {
				ID string `json:"id"`
			} `json:"succeeded"`
			Failed      []any `json:"failed"`
			Total       int   `json:"total"`
			Closed      int   `json:"closed"`
			FailedCount int   `json:"failed_count"`
		} `json:"data"`
		Error any `json:"error"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("invalid JSON output %q: %v", output, err)
	}
	if response.Command != "close" || !response.OK || response.Data.Total != 2 || response.Data.Closed != 2 || response.Data.FailedCount != 0 || len(response.Data.Succeeded) != 2 || len(response.Data.Failed) != 0 || response.Error != nil {
		t.Fatalf("response = %#v", response)
	}
	if response.Data.Succeeded[0].ID != "00001" || response.Data.Succeeded[1].ID != "00002" {
		t.Fatalf("success order = %#v", response.Data.Succeeded)
	}

	output, err = executeCommand(t, "close", "all")
	if err != nil || !strings.Contains(output, "close all: total 0, closed 0, failed 0") {
		t.Fatalf("empty output=%q err=%v", output, err)
	}
	if _, err := executeCommand(t, "create", "Human output", "--id", "00003"); err != nil {
		t.Fatal(err)
	}
	output, err = executeCommand(t, "close", "all")
	if err != nil || !strings.Contains(output, "closed 00003 Human output: .historic/.database/00003-human-output") || !strings.Contains(output, "close all: total 1, closed 1, failed 0") {
		t.Fatalf("human output=%q err=%v", output, err)
	}
}

func TestCloseAllPartialFailureEmitsOneJSONEnvelopeAndNonZero(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Bad", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Good", "--id", "00002"); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".historic", ".database", ".staging-close-00001-bad"), 0o755); err != nil {
		t.Fatal(err)
	}
	output, err := executeCommand(t, "close", "all", "--json")
	if err == nil {
		t.Fatalf("partial close succeeded: %s", output)
	}
	if strings.Count(strings.TrimSpace(output), "\n") != 0 {
		t.Fatalf("expected one JSON envelope, got %q", output)
	}
	var response struct {
		OK   bool `json:"ok"`
		Data struct {
			Total       int `json:"total"`
			Closed      int `json:"closed"`
			FailedCount int `json:"failed_count"`
		} `json:"data"`
	}
	if decodeErr := json.Unmarshal([]byte(output), &response); decodeErr != nil {
		t.Fatalf("invalid JSON output %q: %v", output, decodeErr)
	}
	if response.OK || response.Data.Total != 2 || response.Data.Closed != 1 || response.Data.FailedCount != 1 {
		t.Fatalf("response = %#v", response)
	}
}
