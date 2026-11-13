package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeGuardCopy writes one side of a mirrored topic with a single managed file
// and an explicit per-file stamp.
func writeGuardCopy(t *testing.T, root, base, id, stamp, body string) {
	t.Helper()
	topic := filepath.Join(root, ".historic", base, id+"-guard")
	if err := os.MkdirAll(filepath.Join(topic, "wos"), 0o755); err != nil {
		t.Fatal(err)
	}
	document := "---\ntitle: Task\nstatus: progress\ncreated: \"2026-11-01\"\nupdated: \"2026-11-01\"\n---\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(topic, "wos", "01-task.md"), []byte(document), 0o644); err != nil {
		t.Fatal(err)
	}
	meta := "id: \"" + id + "\"\ntitle: Guard Topic\ncreated: \"2026-11-01\"\nupdated_at: \"2026-11-01T00:00:00Z\"\nfiles:\n" +
		"    - id: 01a1d3ab-90b1-7ca7-8087-c597029e9e54\n      path: wos/01-task.md\n      type: task\n      status: progress\n      updated_at: \"" + stamp + "\"\nassets: []\n"
	if err := os.WriteFile(filepath.Join(topic, "_meta.yaml"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSaveRejectsNewerMirrorOnceAndForceOverrides(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	writeGuardCopy(t, root, "", "1794526185679", "2026-11-12T10:00:00Z", "local edit")
	writeGuardCopy(t, root, ".database", "1794526185679", "2026-11-12T18:00:00Z", "pulled edit")

	output, err := executeCommand(t, "save", "1794526185679", "--json")
	if err == nil {
		t.Fatalf("save accepted a newer mirror: %s", output)
	}
	var response struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if unmarshalErr := json.Unmarshal([]byte(output), &response); unmarshalErr != nil {
		t.Fatalf("decode: %v (%q)", unmarshalErr, output)
	}
	if response.OK {
		t.Fatalf("ok must be false on a rejected save: %s", output)
	}
	if !strings.Contains(response.Error, "wos/01-task.md") {
		t.Fatalf("error does not name the offending path: %q", response.Error)
	}

	// Human output must guide the operator exactly once, with no duplicate
	// report on stderr.
	human, err := executeCommand(t, "save", "1794526185679")
	if err == nil {
		t.Fatalf("save accepted a newer mirror: %s", human)
	}
	if strings.Count(human, "mirror copy is newer") != 1 {
		t.Fatalf("conflict was not reported exactly once: %s", human)
	}
	if !strings.Contains(human, "--force") {
		t.Fatalf("conflict message does not point at --force: %s", human)
	}

	output, err = executeCommand(t, "save", "1794526185679", "--force")
	if err != nil {
		t.Fatalf("--force save failed: %v (%s)", err, output)
	}
	if !strings.Contains(output, "saved 1794526185679") {
		t.Fatalf("--force save did not mirror the topic: %s", output)
	}
}
