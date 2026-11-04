package cmd

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// TestFindJSONAdditiveContract locks the existing keys while asserting the
// new relation/match fields without changing any prior field name or meaning.
func TestFindJSONAdditiveContract(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	parentOutput, err := executeCommand(t, "create", "Json Parent")
	if err != nil {
		t.Fatal(err)
	}
	parentID := createdIDFromOutput(t, parentOutput)
	childOutput, err := executeCommand(t, "create", "Json Child", "--extends", parentID)
	if err != nil {
		t.Fatal(err)
	}
	childID := createdIDFromOutput(t, childOutput)
	body := strings.Join([]string{
		"needle first location",
		strings.Repeat("filler text ", 12) + "needle second location",
		strings.Repeat("more filler ", 12) + "needle third location",
		strings.Repeat("extra filler ", 12) + "needle fourth location",
		strings.Repeat("last filler ", 12) + "needle fifth location",
	}, "\n")
	writeManagedNote(t, filepath.Join(root, ".historic", childID+"-json-child", "note.md"), `---
id: `+childID+`
title: Json Note
description: "json additive description"
status: progress
created: 2026-11-01
updated: 2026-11-02
tags:
  - contract
forwarded_to: "`+parentID+`"
---
`+body+"\n")
	if _, err := executeCommand(t, "rebuild"); err != nil {
		t.Fatal(err)
	}

	output, err := executeCommand(t, "find", "needle", "--type", "file", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var response struct {
		Command string `json:"command"`
		OK      bool   `json:"ok"`
		Data    struct {
			Items      []map[string]any `json:"items"`
			Pagination map[string]any   `json:"pagination"`
		} `json:"data"`
		Error any `json:"error"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("json = %q: %v", output, err)
	}
	if !response.OK || response.Command != "find" || len(response.Data.Items) != 1 {
		t.Fatalf("unexpected envelope = %+v", response)
	}
	item := response.Data.Items[0]
	for _, existing := range []string{"type", "topic_id", "title", "description", "status", "storage", "tags", "path", "score", "matched_in", "snippet", "topic"} {
		if _, present := item[existing]; !present {
			t.Fatalf("existing field %q missing from %v", existing, item)
		}
	}
	if item["description"] != "json additive description" {
		t.Fatalf("description = %v", item["description"])
	}
	if item["extends"] != parentID {
		t.Fatalf("extends = %v", item["extends"])
	}
	if item["forwarded_to"] != parentID {
		t.Fatalf("forwarded_to = %v", item["forwarded_to"])
	}
	if item["match_count"] != float64(5) {
		t.Fatalf("match_count = %v", item["match_count"])
	}
	previews, ok := item["match_previews"].([]any)
	if !ok || len(previews) != 3 {
		t.Fatalf("match_previews = %v", item["match_previews"])
	}
	for _, preview := range previews {
		if !strings.Contains(strings.ToLower(preview.(string)), "needle") {
			t.Fatalf("preview missing keyword: %v", preview)
		}
	}
}

// TestFindJSONTopicResultKeepsMatchFields ensures topic rows stay additive and
// well typed instead of returning null for the new arrays.
func TestFindJSONTopicResultKeepsMatchFields(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Topic Match", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "rebuild"); err != nil {
		t.Fatal(err)
	}
	output, err := executeCommand(t, "find", "topic", "--type", "topic", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var response struct {
		Data struct {
			Items []struct {
				MatchCount    int      `json:"match_count"`
				MatchPreviews []string `json:"match_previews"`
				Extends       string   `json:"extends"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("json = %q: %v", output, err)
	}
	if len(response.Data.Items) != 1 {
		t.Fatalf("items = %+v", response.Data.Items)
	}
	if response.Data.Items[0].MatchPreviews == nil || len(response.Data.Items[0].MatchPreviews) != 0 {
		t.Fatalf("match_previews = %#v", response.Data.Items[0].MatchPreviews)
	}
}
