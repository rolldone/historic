package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOpenWithoutIDJSONReturnsEnvelopeError(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	output, err := executeCommand(t, "open", "--json")
	if err == nil {
		t.Fatalf("expected error, output=%q", output)
	}
	var response struct {
		Command string `json:"command"`
		OK      bool   `json:"ok"`
		Error   string `json:"error"`
	}
	if json.Unmarshal([]byte(output), &response) != nil || response.Command != "open" || response.OK || !strings.Contains(response.Error, "topic ID is required") {
		t.Fatalf("response=%q", output)
	}
}

func TestOpenExplicitIDStillWorks(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Open me", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "close", "00001"); err != nil {
		t.Fatal(err)
	}
	output, err := executeCommand(t, "open", "00001", "--json")
	if err != nil || !strings.Contains(output, `"command":"open"`) {
		t.Fatalf("output=%q err=%v", output, err)
	}
}
