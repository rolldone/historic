package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeManagedNote(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFindHumanOutputIsMultiLineAndLabeled(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("COLUMNS", "80")
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Display Parent", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	writeManagedNote(t, filepath.Join(root, ".historic", "00001-display-parent", "note.md"), `---
id: 00001
title: Display Note
description: "A full reason line: with colon and detail."
status: progress
created: 2026-11-01
updated: 2026-11-02
tags:
  - lifecycle
  - git
---
alpha keyword first
second alpha keyword line
`)
	if _, err := executeCommand(t, "rebuild"); err != nil {
		t.Fatal(err)
	}

	output, err := executeCommand(t, "find", "keyword", "--open")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"00001 · Display Note",
		"[OPEN] · progress",
		"  path    .historic/00001-display-parent/note.md",
		"  time    updated 2026-11-02 · mtime ",
		"  reason  A full reason line: with colon and detail.",
		"  match   2 lokasi · terbaik di sekitar baris 1",
		"  tag     lifecycle, git",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("find output missing %q:\n%s", want, output)
		}
	}
	if !strings.Contains(output, "\n\n") {
		t.Fatalf("expected blank line separation in output:\n%s", output)
	}
}

func TestFindHumanOutputShowsRelations(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("COLUMNS", "80")
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	parentOutput, err := executeCommand(t, "create", "Parent Topic")
	if err != nil {
		t.Fatal(err)
	}
	parentID := createdIDFromOutput(t, parentOutput)
	childOutput, err := executeCommand(t, "create", "Child Topic", "--extends", parentID)
	if err != nil {
		t.Fatal(err)
	}
	childID := createdIDFromOutput(t, childOutput)
	writeManagedNote(t, filepath.Join(root, ".historic", parentID+"-parent-topic", "note.md"), `---
id: `+parentID+`
title: Forwarded Note
description: "forwarded relationship"
status: forwarded
created: 2026-11-01
forwarded_to: "`+childID+`"
---
forwarded body
`)
	writeManagedNote(t, filepath.Join(root, ".historic", childID+"-child-topic", "note.md"), `---
id: `+childID+`
title: Child Note
description: "relation keyword in description"
status: progress
created: 2026-11-01
---
body relation keyword
`)
	if _, err := executeCommand(t, "rebuild"); err != nil {
		t.Fatal(err)
	}

	child, err := executeCommand(t, "find", "relation", "--open")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(child, "  extends → "+parentID) {
		t.Fatalf("child output missing extends relation:\n%s", child)
	}
	forwarded, err := executeCommand(t, "find", "forwarded", "--open")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(forwarded, "  forwarded → "+childID) {
		t.Fatalf("forwarded output missing relation:\n%s", forwarded)
	}
}

func createdIDFromOutput(t *testing.T, output string) string {
	t.Helper()
	fields := strings.Fields(output)
	for index, field := range fields {
		if field == "topic" && index+1 < len(fields) {
			return fields[index+1]
		}
	}
	t.Fatalf("could not parse created topic ID from %q", output)
	return ""
}
