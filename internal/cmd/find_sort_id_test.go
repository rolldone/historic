package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedSortIDWorkspace creates two topics with one matching file each, so the
// legacy ID ascending order can be compared between list and find.
func seedSortIDWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Second", "--id", "00002"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "First", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	write := func(topics, id, title string) {
		t.Helper()
		path := filepath.Join(root, ".historic", id+"-"+topics, "note.md")
		content := "---\nid: " + id + "\ntitle: " + title + "\nstatus: progress\ncreated: 2026-09-18\n---\nneedle body\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("second", "00002", "Second Note")
	write("first", "00001", "First Note")
	if _, err := executeCommand(t, "rebuild"); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestFindSortIDCommandOrdersTopicIDAscending(t *testing.T) {
	seedSortIDWorkspace(t)

	output, err := executeCommand(t, "find", "needle", "--sort", "id", "--json")
	if err != nil {
		t.Fatalf("find --sort id: %v output=%s", err, output)
	}
	var response struct {
		OK   bool `json:"ok"`
		Data struct {
			Items []struct {
				TopicID string `json:"topic_id"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("decode %q: %v", output, err)
	}
	if !response.OK || len(response.Data.Items) == 0 {
		t.Fatalf("response = %s", output)
	}
	previous := ""
	for _, item := range response.Data.Items {
		if previous != "" && previous > item.TopicID {
			t.Fatalf("find --sort id not ascending: %s before %s (%s)", previous, item.TopicID, output)
		}
		previous = item.TopicID
	}
	if response.Data.Items[0].TopicID != "00001" {
		t.Fatalf("first item = %s, want 00001 (%s)", response.Data.Items[0].TopicID, output)
	}
}

func TestListAndFindSortIDShareLegacyOrder(t *testing.T) {
	seedSortIDWorkspace(t)

	listOutput, err := executeCommand(t, "list", "--sort", "id", "--json")
	if err != nil {
		t.Fatalf("list --sort id: %v output=%s", err, listOutput)
	}
	var listResponse struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(listOutput), &listResponse); err != nil {
		t.Fatalf("decode list %q: %v", listOutput, err)
	}
	findOutput, err := executeCommand(t, "find", "needle", "--sort", "id", "--json")
	if err != nil {
		t.Fatalf("find --sort id: %v output=%s", err, findOutput)
	}
	var findResponse struct {
		Data struct {
			Items []struct {
				TopicID string `json:"topic_id"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(findOutput), &findResponse); err != nil {
		t.Fatalf("decode find %q: %v", findOutput, err)
	}
	seen := make(map[string]struct{}, len(findResponse.Data.Items))
	for _, item := range findResponse.Data.Items {
		seen[item.TopicID] = struct{}{}
	}
	listed := make([]string, 0, len(listResponse.Data))
	for _, topic := range listResponse.Data {
		if _, ok := seen[topic.ID]; ok {
			listed = append(listed, topic.ID)
		}
	}
	if len(listed) < 2 {
		t.Fatalf("list did not cover the find topics: %s / %s", listOutput, findOutput)
	}
	if listed[0] != "00001" || listed[1] != "00002" {
		t.Fatalf("list --sort id order = %v, want [00001 00002]", listed)
	}
}

func TestFindSortRejectsInvalidValueNamingAllOptions(t *testing.T) {
	seedSortIDWorkspace(t)

	output, err := executeCommand(t, "find", "needle", "--sort", "bogus")
	if err == nil {
		t.Fatalf("invalid --sort accepted: %s", output)
	}
	for _, value := range []string{"relevance", "updated", "created", "title", "id"} {
		if !strings.Contains(err.Error(), value) {
			t.Fatalf("error %q does not name %q", err.Error(), value)
		}
	}
}
