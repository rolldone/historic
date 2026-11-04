package cmd

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// setupFindCompactFixture creates a topic with a multi-match file and returns
// the topic ID. Frontmatter dates are fixed so the JSON output is
// deterministic apart from the temp workspace path and file mtime.
func setupFindCompactFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	output, err := executeCommand(t, "create", "Compact Fixture")
	if err != nil {
		t.Fatal(err)
	}
	topicID := createdIDFromOutput(t, output)
	body := strings.Join([]string{
		"needle first location",
		"filler filler filler filler filler filler filler filler filler filler filler filler needle second location",
		"more more more more more more more more more more more more needle third location",
		"extra extra extra extra extra extra extra extra extra extra extra extra needle fourth location",
		"last last last last last last last last last last last last needle fifth location",
	}, "\n")
	writeManagedNote(t, filepath.Join(root, ".historic", topicID+"-compact-fixture", "note.md"), `---
id: `+topicID+`
title: Compact Note
description: "compact description"
status: progress
created: 2026-11-01
updated: 2026-11-02
tags:
  - contract
forwarded_to: "`+topicID+`"
---
`+body+"\n")
	if _, err := executeCommand(t, "rebuild"); err != nil {
		t.Fatal(err)
	}
	return topicID
}

func decodeFindItems(t *testing.T, output string) []map[string]any {
	t.Helper()
	var response struct {
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("json = %q: %v", output, err)
	}
	return response.Data.Items
}

func sortedKeys(item map[string]any) []string {
	keys := make([]string, 0, len(item))
	for key := range item {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// TestFindJSONCompactShape verifies the compact item matches SPEC §6.1: the
// kept fields, the dropped fields, and the size win over full output.
func TestFindJSONCompactShape(t *testing.T) {
	topicID := setupFindCompactFixture(t)
	if _, err := executeCommand(t, "create", "Compact Child", "--extends", topicID); err != nil {
		t.Fatal(err)
	}

	fullOutput, err := executeCommand(t, "find", "needle", "--type", "file", "--json")
	if err != nil {
		t.Fatal(err)
	}
	compactOutput, err := executeCommand(t, "find", "needle", "--type", "file", "--json", "--compact")
	if err != nil {
		t.Fatal(err)
	}
	if len(compactOutput) >= len(fullOutput) {
		t.Fatalf("compact bytes %d not smaller than full bytes %d", len(compactOutput), len(fullOutput))
	}

	fullItems := decodeFindItems(t, fullOutput)
	compactItems := decodeFindItems(t, compactOutput)
	if len(fullItems) != 1 || len(compactItems) != 1 {
		t.Fatalf("full items = %d, compact items = %d", len(fullItems), len(compactItems))
	}
	fullItem, compactItem := fullItems[0], compactItems[0]

	goldenFullKeys := []string{"created_at", "description", "forwarded_to", "hash", "match_count", "match_previews", "matched_in", "mtime", "path", "score", "size", "snippet", "status", "storage", "tags", "title", "topic", "topic_id", "type", "updated_at"}
	if got := sortedKeys(fullItem); !reflect.DeepEqual(got, goldenFullKeys) {
		t.Fatalf("full item keys = %v, want %v", got, goldenFullKeys)
	}
	goldenCompactKeys := []string{"created_at", "description", "forwarded_to", "match_count", "match_previews", "path", "status", "storage", "tags", "title", "topic_id", "type", "updated_at"}
	if got := sortedKeys(compactItem); !reflect.DeepEqual(got, goldenCompactKeys) {
		t.Fatalf("compact item keys = %v, want %v", got, goldenCompactKeys)
	}
	for _, dropped := range []string{"topic", "hash", "size", "mtime", "score", "matched_in", "snippet"} {
		if _, present := compactItem[dropped]; present {
			t.Fatalf("dropped field %q present in compact item %v", dropped, compactItem)
		}
	}
	for _, kept := range []string{"type", "topic_id", "title", "description", "status", "storage", "tags", "path", "created_at", "updated_at", "match_count", "match_previews", "forwarded_to"} {
		if _, present := compactItem[kept]; !present {
			t.Fatalf("kept field %q missing from compact item %v", kept, compactItem)
		}
	}
	if compactItem["match_count"] != float64(5) {
		t.Fatalf("match_count = %v", compactItem["match_count"])
	}
	previews, ok := compactItem["match_previews"].([]any)
	if !ok || len(previews) != 3 {
		t.Fatalf("match_previews = %v", compactItem["match_previews"])
	}
	if previews[0] != fullItem["snippet"] {
		t.Fatalf("match_previews[0] = %v, want snippet %v", previews[0], fullItem["snippet"])
	}
	for _, preview := range previews {
		if !strings.Contains(strings.ToLower(preview.(string)), "needle") {
			t.Fatalf("preview missing keyword: %v", preview)
		}
	}
}

// TestFindJSONFullOutputByteIdentical guards the non-breaking contract: the
// full output produced twice in one workspace must be byte-identical.
func TestFindJSONFullOutputByteIdentical(t *testing.T) {
	setupFindCompactFixture(t)
	first, err := executeCommand(t, "find", "needle", "--type", "file", "--json")
	if err != nil {
		t.Fatal(err)
	}
	second, err := executeCommand(t, "find", "needle", "--type", "file", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("full output changed between runs:\nfirst:  %q\nsecond: %q", first, second)
	}
}

// TestFindCompactWithoutJSONRejected ensures --compact is rejected without
// --json with an actionable message.
func TestFindCompactWithoutJSONRejected(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	_, err := executeCommand(t, "find", "needle", "--compact")
	if err == nil {
		t.Fatal("expected error for --compact without --json")
	}
	if !strings.Contains(err.Error(), "--compact requires --json") {
		t.Fatalf("error = %v", err)
	}
}

// TestFindCompactTopicRow ensures topic rows carry the same compact field set
// with zero-valued match metadata, and that relation fields stay omitted when
// absent.
func TestFindCompactTopicRow(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Compact Topic Row", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "rebuild"); err != nil {
		t.Fatal(err)
	}
	output, err := executeCommand(t, "find", "compact", "--type", "topic", "--json", "--compact")
	if err != nil {
		t.Fatal(err)
	}
	items := decodeFindItems(t, output)
	if len(items) != 1 {
		t.Fatalf("items = %v", items)
	}
	item := items[0]
	if item["type"] != "topic" {
		t.Fatalf("type = %v", item["type"])
	}
	if item["match_count"] != float64(0) {
		t.Fatalf("match_count = %v", item["match_count"])
	}
	previews, ok := item["match_previews"].([]any)
	if !ok || len(previews) != 0 {
		t.Fatalf("match_previews = %v", item["match_previews"])
	}
	for _, absent := range []string{"topic", "hash", "size", "mtime", "score", "matched_in", "snippet", "extends", "forwarded_to"} {
		if _, present := item[absent]; present {
			t.Fatalf("field %q present in topic row = %v", absent, item)
		}
	}
}
