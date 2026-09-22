package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"historic/internal/domain"
)

func executeCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()
	command := NewRootCommand()
	var output bytes.Buffer
	ConfigureOutput(command, &output, &output)
	command.SetArgs(args)
	err := command.Execute()
	return output.String(), err
}

func TestPhase1EndToEndWorkflow(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	if output, err := executeCommand(t, "init"); err != nil || !strings.Contains(output, "Initialized Historic") {
		t.Fatalf("init output=%q err=%v", output, err)
	}
	if output, err := executeCommand(t, "init"); err != nil || !strings.Contains(output, "Initialized Historic") {
		t.Fatalf("repeated init output=%q err=%v", output, err)
	}
	if _, err := executeCommand(t, "create", "Phase One", "--id", "00014"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := executeCommand(t, "add", "prd", "--id", "00014"); err != nil {
		t.Fatalf("add prd: %v", err)
	}
	if _, err := executeCommand(t, "add", "wos/scaffold", "--id", "00014"); err != nil {
		t.Fatalf("add work order: %v", err)
	}
	listOutput, err := executeCommand(t, "list", "--json")
	if err != nil || !jsonHasOK(listOutput) || !strings.Contains(listOutput, "Phase One") {
		t.Fatalf("list output=%q err=%v", listOutput, err)
	}
	showOutput, err := executeCommand(t, "show", "00014", "--json")
	if err != nil || !jsonHasOK(showOutput) || !strings.Contains(showOutput, "prd.md") || !strings.Contains(showOutput, "01-scaffold.md") {
		t.Fatalf("show output=%q err=%v", showOutput, err)
	}
	rebuildOutput, err := executeCommand(t, "rebuild", "--json")
	if err != nil || !jsonHasOK(rebuildOutput) || !strings.Contains(rebuildOutput, `"records":3`) {
		t.Fatalf("rebuild output=%q err=%v", rebuildOutput, err)
	}
	findOutput, err := executeCommand(t, "find", "scaffold", "--open", "--json")
	if err != nil || !jsonHasOK(findOutput) || !strings.Contains(findOutput, "01-scaffold.md") {
		t.Fatalf("find output=%q err=%v", findOutput, err)
	}
}

func TestPhase1RejectsInvalidInputAndCorruptFrontmatter(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "", "--id", "00001"); err == nil {
		t.Fatal("empty title accepted")
	}
	if _, err := executeCommand(t, "create", "Topic", "--id", "14"); !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("invalid ID error=%v, want ErrInvalidID", err)
	}
	broken := filepath.Join(root, ".historic", "00001-broken.md")
	if err := os.WriteFile(broken, []byte("not frontmatter\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "rebuild"); err != nil {
		t.Fatalf("rebuild with regular Markdown asset failed: %v", err)
	}
}

func TestPhase1EmptyResultsAreSuccessful(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "rebuild", "--json"); err != nil {
		t.Fatal(err)
	}
	output, err := executeCommand(t, "list", "--json")
	if err != nil || !jsonHasOK(output) || !strings.Contains(output, `"data":[]`) {
		t.Fatalf("list empty output=%q err=%v", output, err)
	}
	output, err = executeCommand(t, "find", "absent", "--json")
	if err != nil || !jsonHasOK(output) || !strings.Contains(output, `"items":[]`) || !strings.Contains(output, `"pagination"`) {
		t.Fatalf("find empty output=%q err=%v", output, err)
	}
}

func jsonHasOK(output string) bool {
	var response struct {
		OK bool `json:"ok"`
	}
	return json.Unmarshal([]byte(output), &response) == nil && response.OK
}
