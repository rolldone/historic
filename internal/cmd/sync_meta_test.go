package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncMetaCommandClassifiesFilesAndAssets(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Topic", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	topic := filepath.Join(root, ".historic", "00001-topic")
	if err := os.WriteFile(filepath.Join(topic, "manual.md"), []byte("---\nid: 00001\ntitle: Manual\nstatus: progress\ncreated: 2026-09-19\n---\nmanaged\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(topic, "plain.md"), []byte("plain asset"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(topic, "image.png"), []byte("binary-ish"), 0o644); err != nil {
		t.Fatal(err)
	}
	command := NewRootCommand()
	var output bytes.Buffer
	ConfigureOutput(command, &output, &output)
	command.SetArgs([]string{"sync-meta", "00001", "--json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("sync-meta: %v", err)
	}
	var response struct {
		Command string `json:"command"`
		OK      bool   `json:"ok"`
		Data    struct {
			Files  int `json:"files"`
			Assets int `json:"assets"`
		} `json:"data"`
	}
	if err := json.Unmarshal(output.Bytes(), &response); err != nil || !response.OK || response.Command != "sync-meta" || response.Data.Files != 1 || response.Data.Assets != 2 {
		t.Fatalf("response=%q err=%v", output.String(), err)
	}
	meta, err := os.ReadFile(filepath.Join(topic, "_meta.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(meta)
	for _, expected := range []string{"- [manual.md](./manual.md)", "- [plain.md](./plain.md)", "- [image.png](./image.png)"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("metadata missing %q: %s", expected, text)
		}
	}
	if _, err := command.ExecuteC(); err != nil {
		t.Fatalf("repeated sync-meta: %v", err)
	}
	meta, err = os.ReadFile(filepath.Join(topic, "_meta.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(meta), "./manual.md") != 1 || strings.Count(string(meta), "./plain.md") != 1 || strings.Count(string(meta), "./image.png") != 1 {
		t.Fatalf("repeated sync-meta duplicated links: %s", meta)
	}
}

func TestSyncMetaAndStatusRejectAsset(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Topic", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	asset := filepath.Join(root, ".historic", "00001-topic", "plain.txt")
	if err := os.WriteFile(asset, []byte("asset"), 0o644); err != nil {
		t.Fatal(err)
	}
	command := NewRootCommand()
	var output, stderr bytes.Buffer
	ConfigureOutput(command, &output, &stderr)
	command.SetArgs([]string{"status", ".historic/00001-topic/plain.txt", "complete", "--json"})
	if err := command.Execute(); err == nil || !strings.Contains(output.String(), "file status unsupported") || stderr.Len() != 0 {
		t.Fatalf("asset status output=%q stderr=%q err=%v", output.String(), stderr.String(), err)
	}
}
