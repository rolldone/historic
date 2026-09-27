package cmd

import (
	"strings"
	"testing"
)

func TestListClosedShowsOnlyClosedTopics(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Open Topic", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Closed Topic", "--id", "00002"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "close", "00002"); err != nil {
		t.Fatal(err)
	}

	output, err := executeCommand(t, "list", "--closed")
	if err != nil {
		t.Fatalf("list --closed: %v", err)
	}
	if !strings.Contains(output, "00002") || !strings.Contains(output, "Closed Topic") {
		t.Fatalf("closed topic missing from output: %q", output)
	}
	if strings.Contains(output, "00001") || strings.Contains(output, "Open Topic") {
		t.Fatalf("open topic leaked into --closed output: %q", output)
	}
}
