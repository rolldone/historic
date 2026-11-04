package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	extendCmdParentID = "1791851149892"
	extendCmdChildID  = "1792889892200"
)

func runCommand(t *testing.T, output *bytes.Buffer, args ...string) error {
	t.Helper()
	command := NewRootCommand()
	output.Reset()
	ConfigureOutput(command, output, output)
	command.SetArgs(args)
	return command.Execute()
}

func initWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Chdir(root)
	var output bytes.Buffer
	if err := runCommand(t, &output, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}
	return root
}

func TestExtendCommandJSONAndShow(t *testing.T) {
	initWorkspace(t)
	var output bytes.Buffer
	if err := runCommand(t, &output, "create", "Parent", "--id", extendCmdParentID); err != nil {
		t.Fatalf("create parent: %v", err)
	}
	if err := runCommand(t, &output, "create", "Child", "--id", extendCmdChildID); err != nil {
		t.Fatalf("create child: %v", err)
	}

	if err := runCommand(t, &output, "extend", extendCmdChildID, extendCmdParentID, "--json"); err != nil {
		t.Fatalf("extend: %v", err)
	}
	var response struct {
		Command string `json:"command"`
		OK      bool   `json:"ok"`
		Data    struct {
			ID      string `json:"id"`
			Extends string `json:"extends"`
			Path    string `json:"path"`
		} `json:"data"`
		Error any `json:"error"`
	}
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatalf("decode output: %v (%q)", err, output.String())
	}
	if !response.OK || response.Command != "extend" {
		t.Fatalf("response = %#v", response)
	}
	if response.Data.ID != extendCmdChildID || response.Data.Extends != extendCmdParentID || response.Data.Path == "" {
		t.Fatalf("data = %#v", response.Data)
	}

	if err := runCommand(t, &output, "show", extendCmdChildID, "--json"); err != nil {
		t.Fatalf("show: %v", err)
	}
	var shown struct {
		Data []struct {
			Extends string `json:"extends"`
		} `json:"data"`
	}
	if err := json.Unmarshal(output.Bytes(), &shown); err != nil {
		t.Fatalf("decode show: %v (%q)", err, output.String())
	}
	if len(shown.Data) != 1 || shown.Data[0].Extends != extendCmdParentID {
		t.Fatalf("show extends = %#v", shown.Data)
	}
}

func TestExtendCommandRejectedSecondExtendHasSingleError(t *testing.T) {
	initWorkspace(t)
	var output bytes.Buffer
	if err := runCommand(t, &output, "create", "Parent", "--id", extendCmdParentID); err != nil {
		t.Fatalf("create parent: %v", err)
	}
	if err := runCommand(t, &output, "create", "Child", "--id", extendCmdChildID); err != nil {
		t.Fatalf("create child: %v", err)
	}
	if err := runCommand(t, &output, "create", "Other", "--id", "1792969715909"); err != nil {
		t.Fatalf("create other: %v", err)
	}
	if err := runCommand(t, &output, "extend", extendCmdChildID, extendCmdParentID); err != nil {
		t.Fatalf("extend: %v", err)
	}
	metaPath := filepath.Join(".historic", extendCmdChildID+"-child", "_meta.yaml")
	before, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatal(err)
	}

	err = runCommand(t, &output, "extend", extendCmdChildID, "1792969715909", "--json")
	if err == nil {
		t.Fatal("second extend accepted")
	}
	if _, ok := err.(SilentError); !ok {
		t.Fatalf("error = %T %v, want SilentError", err, err)
	}
	if count := strings.Count(output.String(), "\"ok\":false"); count != 1 {
		t.Fatalf("error envelope count = %d in %q", count, output.String())
	}
	after, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("_meta.yaml changed after rejected extend")
	}
}
