package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFindCommandPaginationHumanAndJSON(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	for index, id := range []string{"00001", "00002", "00003"} {
		if _, err := executeCommand(t, "create", "Pagination "+string(rune('1'+index)), "--id", id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := executeCommand(t, "rebuild"); err != nil {
		t.Fatal(err)
	}

	human, err := executeCommand(t, "find", "pagination", "--type", "topic", "--page", "1", "--page-size", "2")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(human, "Page 1 · showing 2 results · more results available") || !strings.Contains(human, "Use --page 2 --page-size 2") {
		t.Fatalf("human pagination output = %q", human)
	}

	output, err := executeCommand(t, "find", "pagination", "--type", "topic", "--page", "2", "--page-size", "2", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var response struct {
		OK   bool `json:"ok"`
		Data struct {
			Items      []map[string]any `json:"items"`
			Pagination struct {
				Page     int  `json:"page"`
				PageSize int  `json:"page_size"`
				HasMore  bool `json:"has_more"`
				NextPage *int `json:"next_page"`
			} `json:"pagination"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("JSON output = %q: %v", output, err)
	}
	if !response.OK || len(response.Data.Items) != 1 || response.Data.Pagination.Page != 2 || response.Data.Pagination.PageSize != 2 || response.Data.Pagination.HasMore || response.Data.Pagination.NextPage != nil {
		t.Fatalf("pagination response = %+v", response)
	}
}

func TestFindCommandRejectsInvalidPagination(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"--page", "0"}, {"--page", "-1"}, {"--page-size", "-1"}, {"--page-size", "101"}} {
		commandArgs := append([]string{"find", ""}, args...)
		if _, err := executeCommand(t, commandArgs...); err == nil {
			t.Fatalf("invalid pagination accepted: %v", args)
		}
	}
}
