package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func configureInternalIdentity(t *testing.T, root string) {
	t.Helper()
	for key, value := range map[string]string{"user.name": "Historic Test", "user.email": "historic@test.local"} {
		command := exec.Command("git", "config", key, value)
		command.Dir = filepath.Join(root, ".historic", ".database")
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git config %s: %v: %s", key, err, output)
		}
	}
}

func internalCommitCount(t *testing.T, root string) int {
	t.Helper()
	command := exec.Command("git", "rev-list", "--count", "HEAD")
	command.Dir = filepath.Join(root, ".historic", ".database")
	output, err := command.CombinedOutput()
	if err != nil {
		return 0
	}
	count := 0
	for _, character := range strings.TrimSpace(string(output)) {
		count = count*10 + int(character-'0')
	}
	return count
}

type autoCommitPayload struct {
	Committed    bool   `json:"committed"`
	Hash         string `json:"hash"`
	Message      string `json:"message"`
	ChangedPaths int    `json:"changed_paths"`
	Reason       string `json:"reason"`
}

func TestSaveReportsAutoCommitThenSkipsWhenClean(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	configureInternalIdentity(t, root)
	if _, err := executeCommand(t, "create", "Topic", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "add", "Note", "--id", "00001"); err != nil {
		t.Fatal(err)
	}

	output, err := executeCommand(t, "save", "00001", "--json")
	if err != nil {
		t.Fatalf("save: %v (%s)", err, output)
	}
	var first struct {
		OK   bool `json:"ok"`
		Data struct {
			Saved  int                `json:"saved"`
			Commit *autoCommitPayload `json:"commit"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &first); err != nil {
		t.Fatal(err)
	}
	if !first.OK || first.Data.Commit == nil || !first.Data.Commit.Committed {
		t.Fatalf("first response = %s", output)
	}
	if len(first.Data.Commit.Hash) != 7 {
		t.Fatalf("hash = %q", first.Data.Commit.Hash)
	}
	if !strings.HasPrefix(first.Data.Commit.Message, "auto: save 00001 (") {
		t.Fatalf("message = %q", first.Data.Commit.Message)
	}
	if !strings.HasSuffix(first.Data.Commit.Message, " path berubah)") {
		t.Fatalf("message = %q", first.Data.Commit.Message)
	}
	if first.Data.Commit.ChangedPaths == 0 {
		t.Fatalf("changed paths = %d", first.Data.Commit.ChangedPaths)
	}

	output, err = executeCommand(t, "save", "00001", "--json")
	if err != nil {
		t.Fatalf("second save: %v (%s)", err, output)
	}
	var second struct {
		OK   bool `json:"ok"`
		Data struct {
			Commit *autoCommitPayload `json:"commit"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &second); err != nil {
		t.Fatal(err)
	}
	if !second.OK || second.Data.Commit == nil || second.Data.Commit.Committed || second.Data.Commit.Reason != "no-changes" {
		t.Fatalf("second response = %s", output)
	}
	if count := internalCommitCount(t, root); count != 1 {
		t.Fatalf("commit count = %d", count)
	}

	human, err := executeCommand(t, "save", "00001")
	if err != nil {
		t.Fatalf("human save: %v", err)
	}
	if !strings.Contains(human, "auto-commit: skipped (no-changes)") {
		t.Fatalf("human output = %q", human)
	}
}

func TestSaveAllProducesOneAggregateCommit(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	configureInternalIdentity(t, root)
	for _, id := range []string{"00001", "00002"} {
		if _, err := executeCommand(t, "create", "Topic "+id, "--id", id); err != nil {
			t.Fatal(err)
		}
	}
	output, err := executeCommand(t, "save", "--all", "--json")
	if err != nil {
		t.Fatalf("save --all: %v (%s)", err, output)
	}
	var response struct {
		Data struct {
			Commit *autoCommitPayload `json:"commit"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.Commit == nil || !response.Data.Commit.Committed {
		t.Fatalf("response = %s", output)
	}
	if !strings.HasPrefix(response.Data.Commit.Message, "auto: save --all (") {
		t.Fatalf("message = %q", response.Data.Commit.Message)
	}
	if count := internalCommitCount(t, root); count != 1 {
		t.Fatalf("bulk save created %d commits", count)
	}
}

func TestStatusCommitsOnlyForClosedPaths(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	configureInternalIdentity(t, root)
	if _, err := executeCommand(t, "create", "Topic", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "add", "Note", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	if output, err := executeCommand(t, "save", "00001", "--json"); err != nil {
		t.Fatalf("save: %v (%s)", err, output)
	}
	baseline := internalCommitCount(t, root)

	openPath := filepath.Join(".historic", "00001-topic", "note.md")
	output, err := executeCommand(t, "status", openPath, "review", "--json")
	if err != nil {
		t.Fatalf("open status: %v (%s)", err, output)
	}
	var openResponse struct {
		Data struct {
			Commit *autoCommitPayload `json:"commit"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &openResponse); err != nil {
		t.Fatal(err)
	}
	if openResponse.Data.Commit != nil {
		t.Fatalf("open status reported commit: %s", output)
	}
	if count := internalCommitCount(t, root); count != baseline {
		t.Fatalf("open status changed commit count to %d", count)
	}

	closedPath := filepath.Join(".historic", ".database", "00001-topic", "note.md")
	output, err = executeCommand(t, "status", closedPath, "review", "--json")
	if err != nil {
		t.Fatalf("closed status: %v (%s)", err, output)
	}
	var closedResponse struct {
		Data struct {
			Commit *autoCommitPayload `json:"commit"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &closedResponse); err != nil {
		t.Fatal(err)
	}
	if closedResponse.Data.Commit == nil || !closedResponse.Data.Commit.Committed {
		t.Fatalf("closed status response = %s", output)
	}
	if !strings.HasPrefix(closedResponse.Data.Commit.Message, "auto: status 00001 (") {
		t.Fatalf("message = %q", closedResponse.Data.Commit.Message)
	}
	if count := internalCommitCount(t, root); count != baseline+1 {
		t.Fatalf("closed status commit count = %d", count)
	}
}

func TestCloseCreatesSingleCommitAndRemovesWorkdir(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	configureInternalIdentity(t, root)
	if _, err := executeCommand(t, "create", "Topic", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "add", "Note", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	output, err := executeCommand(t, "close", "00001", "--json")
	if err != nil {
		t.Fatalf("close: %v (%s)", err, output)
	}
	var response struct {
		Data struct {
			Commit *autoCommitPayload `json:"commit"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.Commit == nil || !response.Data.Commit.Committed {
		t.Fatalf("response = %s", output)
	}
	if !strings.HasPrefix(response.Data.Commit.Message, "auto: close 00001 (") {
		t.Fatalf("message = %q", response.Data.Commit.Message)
	}
	if count := internalCommitCount(t, root); count != 1 {
		t.Fatalf("close commit count = %d", count)
	}
	if _, err := os.Stat(filepath.Join(root, ".historic", "00001-topic")); !os.IsNotExist(err) {
		t.Fatalf("open workdir still present: %v", err)
	}
}

func TestRebuildReportsAutoCommit(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	configureInternalIdentity(t, root)
	if _, err := executeCommand(t, "create", "Topic", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	if output, err := executeCommand(t, "save", "00001", "--json"); err != nil {
		t.Fatalf("save: %v (%s)", err, output)
	}
	// Introduce a mirror change that only rebuild reconciles into _meta.yaml.
	extra := filepath.Join(root, ".historic", ".database", "00001-topic", "extra.md")
	if err := os.WriteFile(extra, []byte("---\ntitle: Extra\nstatus: draft\ncreated: \"2026-11-02\"\n---\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	output, err := executeCommand(t, "rebuild", "--json")
	if err != nil {
		t.Fatalf("rebuild: %v (%s)", err, output)
	}
	var response struct {
		Data struct {
			Commit *autoCommitPayload `json:"commit"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.Commit == nil || !response.Data.Commit.Committed {
		t.Fatalf("response = %s", output)
	}
	if !strings.HasPrefix(response.Data.Commit.Message, "auto: rebuild ? (") {
		t.Fatalf("message = %q", response.Data.Commit.Message)
	}
}

func TestAutoCommitIdentityFailureIsNonFatalWarning(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	root := t.TempDir()
	t.Chdir(root)
	if _, err := executeCommand(t, "init"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCommand(t, "create", "Topic", "--id", "00001"); err != nil {
		t.Fatal(err)
	}
	output, err := executeCommand(t, "save", "00001", "--json")
	if err != nil {
		t.Fatalf("save must succeed despite commit failure: %v (%s)", err, output)
	}
	var response struct {
		OK   bool `json:"ok"`
		Data struct {
			Commit *autoCommitPayload `json:"commit"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatal(err)
	}
	if !response.OK || response.Data.Commit == nil || response.Data.Commit.Committed {
		t.Fatalf("response = %s", output)
	}
	if !strings.Contains(strings.ToLower(response.Data.Commit.Reason), "identity") {
		t.Fatalf("reason = %q", response.Data.Commit.Reason)
	}

	human, err := executeCommand(t, "save", "00001")
	if err != nil {
		t.Fatalf("human save: %v", err)
	}
	if !strings.Contains(human, "auto-commit gagal (tidak fatal)") || !strings.Contains(human, "historic git add --all") {
		t.Fatalf("human output = %q", human)
	}
}
