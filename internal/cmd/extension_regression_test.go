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
	regParentA = "1791851149892"
	regParentB = "1791851149900"
	regChildC  = "1791851149901"
	regOther   = "1791851149902"
)

type envelope struct {
	Command string          `json:"command"`
	OK      bool            `json:"ok"`
	Data    json.RawMessage `json:"data"`
	Error   json.RawMessage `json:"error"`
}

func decodeEnvelope(t *testing.T, output string) envelope {
	t.Helper()
	var result envelope
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode envelope: %v (%q)", err, output)
	}
	return result
}

func assertErrorEnvelope(t *testing.T, output string, command string) {
	t.Helper()
	result := decodeEnvelope(t, output)
	if result.OK {
		t.Fatalf("expected ok=false, got %q", output)
	}
	if result.Command != command {
		t.Fatalf("command = %q, want %q", result.Command, command)
	}
	if string(result.Error) == "null" || len(result.Error) == 0 {
		t.Fatalf("error envelope missing error value: %q", output)
	}
}

func TestRegressionStatusOnClosedTopicSyncsManifestAndReviewFilter(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Closed Topic", "--id", regParentA); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "add", "Review Me", "--id", regParentA); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "close", regParentA); err != nil {
		t.Fatal(err)
	}
	closedTopic := filepath.Join(root, ".historic", ".database", regParentA+"-closed-topic")
	closedFile := ".historic/.database/" + regParentA + "-closed-topic/review-me.md"

	if _, err := executeCommand(t, "status", closedFile, "review"); err != nil {
		t.Fatalf("status on closed topic: %v", err)
	}
	meta, err := markdown.ReadTopicMetadata(closedTopic)
	if err != nil {
		t.Fatal(err)
	}
	if len(meta.Files) != 1 || meta.Files[0].Status != domain.StatusReview {
		t.Fatalf("closed manifest = %#v", meta.Files)
	}
	document, err := markdown.ParseFile(filepath.Join(closedTopic, "review-me.md"))
	if err != nil {
		t.Fatal(err)
	}
	if document.Frontmatter.Status != domain.StatusReview {
		t.Fatalf("frontmatter = %#v", document.Frontmatter)
	}

	output, err := executeCommand(t, "find", "review", "--status", "review", "--json")
	if err != nil {
		t.Fatalf("find --status review: %v", err)
	}
	if !strings.Contains(output, "review-me.md") {
		t.Fatalf("review filter output = %q", output)
	}
}

func TestRegressionAmbiguousPathAcrossOpenAndClosedCopies(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Dual", "--id", regParentA); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "add", "Shared File", "--id", regParentA); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "save", regParentA); err != nil {
		t.Fatal(err)
	}
	openFile := filepath.Join(root, ".historic", regParentA+"-dual", "shared-file.md")
	closedFile := filepath.Join(root, ".historic", ".database", regParentA+"-dual", "shared-file.md")
	openBefore, err := os.ReadFile(openFile)
	if err != nil {
		t.Fatal(err)
	}
	closedBefore, err := os.ReadFile(closedFile)
	if err != nil {
		t.Fatal(err)
	}

	output, err := executeCommand(t, "status", "shared-file.md", "complete", "--json")
	if err == nil {
		t.Fatal("ambiguous status succeeded")
	}
	assertErrorEnvelope(t, output, "status")
	for path, before := range map[string][]byte{openFile: openBefore, closedFile: closedBefore} {
		after, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if string(before) != string(after) {
			t.Fatalf("ambiguous rejection mutated %s", path)
		}
	}

	if _, err := executeCommand(t, "status", ".historic/.database/"+regParentA+"-dual/shared-file.md", "complete"); err != nil {
		t.Fatalf("explicit closed path rejected: %v", err)
	}
}

func TestRegressionAddOnClosedTopicMatchesLegacyBehavior(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Add Closed", "--id", regParentA); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "close", regParentA); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadDir(filepath.Join(root, ".historic", ".database", regParentA+"-add-closed"))
	if err != nil {
		t.Fatal(err)
	}

	output, err := executeCommand(t, "add", "Late Note", "--id", regParentA, "--json")
	if err == nil {
		t.Fatal("add on closed topic succeeded")
	}
	assertErrorEnvelope(t, output, "add")
	after, err := os.ReadDir(filepath.Join(root, ".historic", ".database", regParentA+"-add-closed"))
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != len(after) {
		t.Fatalf("closed topic mutated: before=%d after=%d entries", len(before), len(after))
	}

	if _, err := executeCommand(t, "open", regParentA); err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := executeCommand(t, "add", "Late Note", "--id", regParentA); err != nil {
		t.Fatalf("add after open: %v", err)
	}
}

func TestRegressionExtensionChainAndParentImmutability(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Topic A", "--id", regParentA); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "close", regParentA); err != nil {
		t.Fatal(err)
	}
	closedA := filepath.Join(root, ".historic", ".database", regParentA+"-topic-a")
	closedABefore, err := os.ReadFile(filepath.Join(closedA, markdown.MetaFilename))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := executeCommand(t, "create", "Topic B", "--id", regParentB, "--extends", regParentA); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "close", regParentB); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Topic C", "--id", regChildC, "--extends", regParentB); err != nil {
		t.Fatal(err)
	}

	metaB, err := markdown.ReadTopicMetadata(filepath.Join(root, ".historic", ".database", regParentB+"-topic-b"))
	if err != nil {
		t.Fatal(err)
	}
	if metaB.Extends != domain.TopicID(regParentA) {
		t.Fatalf("B extends = %q, want %q", metaB.Extends, regParentA)
	}
	metaC, err := markdown.ReadTopicMetadata(filepath.Join(root, ".historic", regChildC+"-topic-c"))
	if err != nil {
		t.Fatal(err)
	}
	if metaC.Extends != domain.TopicID(regParentB) {
		t.Fatalf("C extends = %q, want %q", metaC.Extends, regParentB)
	}
	closedAAfter, err := os.ReadFile(filepath.Join(closedA, markdown.MetaFilename))
	if err != nil {
		t.Fatal(err)
	}
	if string(closedABefore) != string(closedAAfter) {
		t.Fatal("parent A metadata changed by chain extension")
	}

	childJSON, err := executeCommand(t, "show", regChildC, "--json")
	if err != nil {
		t.Fatalf("show child: %v", err)
	}
	if !strings.Contains(childJSON, `"extends":"`+regParentB+`"`) {
		t.Fatalf("child show JSON = %q", childJSON)
	}
	parentJSON, err := executeCommand(t, "show", regParentA, "--json")
	if err != nil {
		t.Fatalf("show parent: %v", err)
	}
	if strings.Contains(parentJSON, "extends") {
		t.Fatalf("parent show leaked extends: %q", parentJSON)
	}
}

func TestRegressionForwardLifecycleAndFilters(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Source", "--id", regParentA); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "add", "Forwardable", "--id", regParentA); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Child", "--id", regChildC, "--extends", regParentA); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Unrelated", "--id", regOther); err != nil {
		t.Fatal(err)
	}

	output, err := executeCommand(t, "forward", "forwardable.md", "--to", regOther, "--json")
	if err == nil {
		t.Fatal("forward to unrelated topic succeeded")
	}
	assertErrorEnvelope(t, output, "forward")

	if _, err := executeCommand(t, "forward", "forwardable.md", "--to", regChildC); err != nil {
		t.Fatalf("forward to child: %v", err)
	}
	if _, err := executeCommand(t, "forward", "forwardable.md", "--to", regChildC); err != nil {
		t.Fatalf("repeat forward: %v", err)
	}

	forwardedJSON, err := executeCommand(t, "find", "forwardable", "--status", "forwarded", "--json")
	if err != nil {
		t.Fatalf("find forwarded: %v", err)
	}
	if !strings.Contains(forwardedJSON, "forwardable.md") {
		t.Fatalf("forwarded filter output = %q", forwardedJSON)
	}

	if _, err := executeCommand(t, "status", "forwardable.md", "progress"); err != nil {
		t.Fatalf("unforward: %v", err)
	}
	sourceTopic := filepath.Join(root, ".historic", regParentA+"-source")
	meta, err := markdown.ReadTopicMetadata(sourceTopic)
	if err != nil {
		t.Fatal(err)
	}
	if len(meta.Files) != 1 || meta.Files[0].ForwardedTo != "" || meta.Files[0].Status != domain.StatusProgress {
		t.Fatalf("manifest after unforward = %#v", meta.Files)
	}
	document, err := markdown.ParseFile(filepath.Join(sourceTopic, "forwardable.md"))
	if err != nil {
		t.Fatal(err)
	}
	if document.Frontmatter.ForwardedTo != "" {
		t.Fatalf("frontmatter after unforward = %#v", document.Frontmatter)
	}
}

func TestRegressionRebuildIsIdempotentWithExtensions(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Parent", "--id", regParentA); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Child", "--id", regChildC, "--extends", regParentA); err != nil {
		t.Fatal(err)
	}

	type rebuildData struct {
		Topics  int `json:"topics"`
		Errors  int `json:"errors"`
		Records int `json:"records"`
	}
	first, err := executeCommand(t, "rebuild", "--json")
	if err != nil {
		t.Fatalf("first rebuild: %v", err)
	}
	var firstResult struct {
		OK   bool        `json:"ok"`
		Data rebuildData `json:"data"`
	}
	if err := json.Unmarshal([]byte(first), &firstResult); err != nil {
		t.Fatalf("decode rebuild: %v (%q)", err, first)
	}
	if !firstResult.OK || firstResult.Data.Errors != 0 {
		t.Fatalf("first rebuild = %#v", firstResult)
	}
	second, err := executeCommand(t, "rebuild", "--json")
	if err != nil {
		t.Fatalf("second rebuild: %v", err)
	}
	var secondResult struct {
		OK   bool        `json:"ok"`
		Data rebuildData `json:"data"`
	}
	if err := json.Unmarshal([]byte(second), &secondResult); err != nil {
		t.Fatalf("decode rebuild: %v (%q)", err, second)
	}
	if !secondResult.OK || secondResult.Data.Errors != 0 {
		t.Fatalf("second rebuild = %#v", secondResult)
	}
	if firstResult.Data.Topics != secondResult.Data.Topics || firstResult.Data.Records != secondResult.Data.Records {
		t.Fatalf("rebuild not idempotent: %#v vs %#v", firstResult.Data, secondResult.Data)
	}
}
