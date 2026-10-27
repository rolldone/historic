package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"historic/internal/domain"
	"historic/internal/markdown"
)

const (
	forwardParentID = "1791851149892"
	forwardChildID  = "1792889892200"
	forwardOtherID  = "1792889899999"
)

func setupForwardWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Parent", "--id", forwardParentID); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "add", "Moved Task", "--id", forwardParentID); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestForwardToExtendingChild(t *testing.T) {
	root := setupForwardWorkspace(t)
	if _, err := executeCommand(t, "create", "Child", "--id", forwardChildID, "--extends", forwardParentID); err != nil {
		t.Fatal(err)
	}
	output, err := executeCommand(t, "forward", "moved-task.md", "--to", forwardChildID, "--json")
	if err != nil {
		t.Fatalf("forward: %v (%s)", err, output)
	}
	var response struct {
		OK   bool `json:"ok"`
		Data struct {
			Path        string `json:"path"`
			Current     string `json:"current"`
			ForwardedTo string `json:"forwarded_to"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("decode: %v (%q)", err, output)
	}
	if !response.OK || response.Data.Current != "forwarded" || response.Data.ForwardedTo != forwardChildID {
		t.Fatalf("response = %#v", response)
	}
	target := filepath.Join(root, ".historic", forwardParentID+"-parent", "moved-task.md")
	document, err := markdown.ParseFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if document.Frontmatter.Status != domain.StatusForwarded || document.Frontmatter.ForwardedTo != domain.TopicID(forwardChildID) {
		t.Fatalf("frontmatter = %#v", document.Frontmatter)
	}
	if !strings.Contains(document.Body, "moved task") && !strings.Contains(document.Body, "Moved Task") {
		t.Fatalf("file body changed: %q", document.Body)
	}
	meta, err := markdown.ReadTopicMetadata(filepath.Join(root, ".historic", forwardParentID+"-parent"))
	if err != nil {
		t.Fatal(err)
	}
	if len(meta.Files) != 1 || meta.Files[0].ForwardedTo != domain.TopicID(forwardChildID) {
		t.Fatalf("manifest = %#v", meta.Files)
	}
}

func TestForwardRejectsNonExtendingTargetWithoutChanges(t *testing.T) {
	root := setupForwardWorkspace(t)
	if _, err := executeCommand(t, "create", "Other", "--id", forwardOtherID); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, ".historic", forwardParentID+"-parent", "moved-task.md")
	before, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "forward", "moved-task.md", "--to", forwardOtherID); err == nil {
		t.Fatal("forward to unrelated topic succeeded")
	}
	after, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("file changed after rejected forward")
	}
}

func TestForwardOptionValidation(t *testing.T) {
	setupForwardWorkspace(t)
	for _, args := range [][]string{
		{"forward", "moved-task.md"},
		{"forward", "moved-task.md", "--to", forwardChildID, "--create", "Child"},
	} {
		if _, err := executeCommand(t, args...); err == nil {
			t.Fatalf("%v accepted", args)
		}
	}
}

func TestForwardRejectsClosedCopy(t *testing.T) {
	root := setupForwardWorkspace(t)
	if _, err := executeCommand(t, "create", "Child", "--id", forwardChildID, "--extends", forwardParentID); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "close", forwardParentID); err != nil {
		t.Fatal(err)
	}
	closedPath := ".historic/.database/" + forwardParentID + "-parent/moved-task.md"
	if _, err := executeCommand(t, "forward", closedPath, "--to", forwardChildID); err == nil {
		t.Fatal("forward from closed copy succeeded")
	}
	if _, err := os.Stat(filepath.Join(root, closedPath)); err != nil {
		t.Fatalf("closed file missing: %v", err)
	}
}

func TestForwardIsIdempotent(t *testing.T) {
	setupForwardWorkspace(t)
	if _, err := executeCommand(t, "create", "Child", "--id", forwardChildID, "--extends", forwardParentID); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "forward", "moved-task.md", "--to", forwardChildID); err != nil {
		t.Fatal(err)
	}
	output, err := executeCommand(t, "forward", "moved-task.md", "--to", forwardChildID, "--json")
	if err != nil {
		t.Fatalf("repeat forward: %v (%s)", err, output)
	}
	if !strings.Contains(output, forwardChildID) {
		t.Fatalf("repeat forward output = %q", output)
	}
}

func TestForwardCreateBuildsExtendingTopic(t *testing.T) {
	root := setupForwardWorkspace(t)
	output, err := executeCommand(t, "forward", "moved-task.md", "--create", "Lanjutan", "--json")
	if err != nil {
		t.Fatalf("forward --create: %v (%s)", err, output)
	}
	var response struct {
		OK   bool `json:"ok"`
		Data struct {
			ForwardedTo  string `json:"forwarded_to"`
			CreatedTopic string `json:"created_topic"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("decode: %v (%q)", err, output)
	}
	if !response.OK || response.Data.CreatedTopic == "" || response.Data.ForwardedTo != response.Data.CreatedTopic {
		t.Fatalf("response = %#v", response)
	}
	entries, err := os.ReadDir(filepath.Join(root, ".historic"))
	if err != nil {
		t.Fatal(err)
	}
	var childPath string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), response.Data.CreatedTopic+"-") {
			childPath = filepath.Join(root, ".historic", entry.Name())
		}
	}
	if childPath == "" {
		t.Fatalf("created topic directory missing for %s", response.Data.CreatedTopic)
	}
	meta, err := markdown.ReadTopicMetadata(childPath)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Extends != domain.TopicID(forwardParentID) {
		t.Fatalf("created topic extends = %q, want %s", meta.Extends, forwardParentID)
	}
}

func TestForwardCreateFailureLeavesStatusUnchanged(t *testing.T) {
	root := setupForwardWorkspace(t)
	target := filepath.Join(root, ".historic", forwardParentID+"-parent", "moved-task.md")
	before, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "forward", "moved-task.md", "--create", "   "); err == nil {
		t.Fatal("forward --create with blank title succeeded")
	}
	after, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("file changed after failed --create")
	}
}
