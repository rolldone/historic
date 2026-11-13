package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCloseRejectsNewerMirrorAndNamesThePath(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	writeGuardCopy(t, root, "", "1794526185679", "2026-11-12T10:00:00Z", "local edit")
	writeGuardCopy(t, root, ".database", "1794526185679", "2026-11-12T18:00:00Z", "pulled edit")

	output, err := executeCommand(t, "close", "1794526185679")
	if err == nil {
		t.Fatalf("close accepted a newer mirror: %s", output)
	}
	// Single-topic close reports through the returned error (SilenceErrors is
	// on; main prints it once), so the message must carry the offending path
	// and the --force guidance.
	if !strings.Contains(err.Error(), "wos/01-task.md") {
		t.Fatalf("close error does not name the offending path: %v", err)
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Fatalf("close error does not point at --force: %v", err)
	}

	output, err = executeCommand(t, "close", "1794526185679", "--force")
	if err != nil {
		t.Fatalf("--force close failed: %v (%s)", err, output)
	}
}

func TestCloseAllWithForceWaivesTheGuard(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	writeGuardCopy(t, root, "", "1794526185679", "2026-11-12T10:00:00Z", "local edit")
	writeGuardCopy(t, root, ".database", "1794526185679", "2026-11-12T18:00:00Z", "pulled edit")

	output, err := executeCommand(t, "close", "--all", "--json")
	if err == nil {
		t.Fatalf("close --all accepted a newer mirror: %s", output)
	}
	var response struct {
		OK   bool `json:"ok"`
		Data struct {
			FailedCount int `json:"failed_count"`
		} `json:"data"`
	}
	if unmarshalErr := json.Unmarshal([]byte(output), &response); unmarshalErr != nil {
		t.Fatalf("decode: %v (%q)", unmarshalErr, output)
	}
	if response.OK || response.Data.FailedCount != 1 {
		t.Fatalf("close --all did not report the per-topic failure: %s", output)
	}

	output, err = executeCommand(t, "close", "--all", "--force")
	if err != nil {
		t.Fatalf("--force close --all failed: %v (%s)", err, output)
	}
	if !strings.Contains(output, "closed 1794526185679") {
		t.Fatalf("--force close --all did not close the topic: %s", output)
	}
}
