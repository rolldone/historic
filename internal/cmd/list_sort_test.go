package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type listSortItem struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Storage string `json:"storage"`
	Path    string `json:"path"`
}

func writeListSortTopic(t *testing.T, root, folder, id, title, created, updated string) {
	t.Helper()
	path := filepath.Join(root, folder)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "id: " + id + "\ntitle: " + title + "\ncreated: " + created + "\n"
	if updated != "" {
		content += "updated: " + updated + "\n"
	}
	if err := os.WriteFile(filepath.Join(path, "_meta.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func seedListSortWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	histories := filepath.Join(root, ".historic")
	writeListSortTopic(t, histories, "00001-older", "00001", "Older", "2026-01-01", "2026-01-05")
	writeListSortTopic(t, histories, "00002-newest", "00002", "Newest", "2026-01-01", "2026-03-01")
	writeListSortTopic(t, histories, "00003-fallback", "00003", "Fallback", "2026-02-15", "")
	writeListSortTopic(t, filepath.Join(histories, ".database"), "00010-closed-old", "00010", "Closed Old", "2026-01-01", "2026-01-10")
	writeListSortTopic(t, filepath.Join(histories, ".database"), "00011-closed-new", "00011", "Closed New", "2026-01-01", "2026-05-10")
	return root
}

func decodeListOutput(t *testing.T, output string) []listSortItem {
	t.Helper()
	var response struct {
		Command string         `json:"command"`
		OK      bool           `json:"ok"`
		Data    []listSortItem `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("decode list: %v (%q)", err, output)
	}
	if !response.OK || response.Command != "list" {
		t.Fatalf("unexpected response: %q", output)
	}
	return response.Data
}

func listIDs(items []listSortItem) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func TestListDefaultSortIsNewestFirst(t *testing.T) {
	seedListSortWorkspace(t)
	output, err := executeCommand(t, "list", "--json")
	if err != nil {
		t.Fatalf("list: %v (%s)", err, output)
	}
	items := decodeListOutput(t, output)
	// Both storage copies are listed: the mirror rows of 00010/00011 sort in
	// with the open rows by their own freshness stamp.
	want := []string{"00011", "00002", "00003", "00010", "00001"}
	if got := listIDs(items); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("default order = %v, want %v", got, want)
	}
	storages := make(map[string]string, len(items))
	for _, item := range items {
		storages[item.ID] = item.Storage
	}
	for _, mirrored := range []string{"00010", "00011"} {
		if storages[mirrored] != "closed" {
			t.Fatalf("mirror row %s storage = %q, want closed (%v)", mirrored, storages[mirrored], items)
		}
	}
	for _, open := range []string{"00001", "00002", "00003"} {
		if storages[open] != "open" {
			t.Fatalf("open row %s storage = %q, want open (%v)", open, storages[open], items)
		}
	}

	explicit, err := executeCommand(t, "list", "--sort", "updated", "--json")
	if err != nil {
		t.Fatalf("list --sort updated: %v", err)
	}
	if got := listIDs(decodeListOutput(t, explicit)); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("explicit updated order = %v, want %v", got, want)
	}
}

func TestListSortCreatedTitleAndLegacyID(t *testing.T) {
	seedListSortWorkspace(t)

	created, err := executeCommand(t, "list", "--sort", "created", "--json")
	if err != nil {
		t.Fatalf("list --sort created: %v", err)
	}
	if got := listIDs(decodeListOutput(t, created)); strings.Join(got, ",") != "00003,00011,00010,00002,00001" {
		t.Fatalf("created order = %v", got)
	}

	title, err := executeCommand(t, "list", "--sort", "title", "--json")
	if err != nil {
		t.Fatalf("list --sort title: %v", err)
	}
	if got := listIDs(decodeListOutput(t, title)); strings.Join(got, ",") != "00011,00010,00003,00002,00001" {
		t.Fatalf("title order = %v", got)
	}

	legacy, err := executeCommand(t, "list", "--sort", "id", "--json")
	if err != nil {
		t.Fatalf("list --sort id: %v", err)
	}
	if got := listIDs(decodeListOutput(t, legacy)); strings.Join(got, ",") != "00001,00002,00003,00010,00011" {
		t.Fatalf("legacy id order = %v", got)
	}
}

func TestListSortWorksOnClosedScope(t *testing.T) {
	seedListSortWorkspace(t)
	output, err := executeCommand(t, "list", "--closed", "--sort", "updated", "--json")
	if err != nil {
		t.Fatalf("list --closed: %v (%s)", err, output)
	}
	items := decodeListOutput(t, output)
	if got := listIDs(items); strings.Join(got, ",") != "00011,00010" {
		t.Fatalf("closed order = %v", got)
	}
	for _, item := range items {
		if item.Storage != "closed" {
			t.Fatalf("closed scope leaked open item: %#v", item)
		}
	}
}

func TestListRejectsUnknownSortValue(t *testing.T) {
	seedListSortWorkspace(t)
	output, err := executeCommand(t, "list", "--sort", "invalid_value")
	if err == nil || !strings.Contains(err.Error(), `invalid --sort value "invalid_value"`) || !strings.Contains(err.Error(), "use updated, created, title, or id") {
		t.Fatalf("human error = %v (output=%q)", err, output)
	}

	jsonOutput, err := executeCommand(t, "list", "--sort", "invalid_value", "--json")
	if err == nil {
		t.Fatalf("json invalid sort unexpectedly succeeded: %q", jsonOutput)
	}
	var response struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if decodeErr := json.Unmarshal([]byte(jsonOutput), &response); decodeErr != nil {
		t.Fatalf("decode: %v (%q)", decodeErr, jsonOutput)
	}
	if response.OK || !strings.Contains(response.Error, "use updated, created, title, or id") {
		t.Fatalf("json error = %q", jsonOutput)
	}
}
