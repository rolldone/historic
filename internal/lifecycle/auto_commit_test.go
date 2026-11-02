package lifecycle

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"historic/internal/config"
	"historic/internal/domain"
	"historic/internal/markdown"
)

func initAutoCommitWorkspace(t *testing.T) config.Workspace {
	t.Helper()
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	configureGitIdentity(t, workspace.Database)
	return workspace
}

func configureGitIdentity(t *testing.T, root string) {
	t.Helper()
	for key, value := range map[string]string{"user.name": "Historic Test", "user.email": "historic@test.local"} {
		command := exec.Command("git", "config", key, value)
		command.Dir = root
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git config %s: %v: %s", key, err, output)
		}
	}
}

func writeOpenTopic(t *testing.T, workspace config.Workspace, id, slug string) {
	t.Helper()
	openPath := filepath.Join(workspace.Histories, id+"-"+slug)
	if err := os.MkdirAll(openPath, 0o755); err != nil {
		t.Fatal(err)
	}
	meta, err := markdown.NewDocument(domain.Frontmatter{
		ID: domain.ID(id), Title: slug, Status: domain.StatusProgress, Created: "2026-11-02", Updated: "2026-11-02",
	}, "body")
	if err != nil {
		t.Fatal(err)
	}
	if err := markdown.WriteFile(filepath.Join(openPath, markdown.MetaFilename), meta); err != nil {
		t.Fatal(err)
	}
}

func internalCommitCount(t *testing.T, root string) int {
	t.Helper()
	command := exec.Command("git", "rev-list", "--count", "HEAD")
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("rev-list: %v: %s", err, output)
	}
	count := 0
	for _, character := range strings.TrimSpace(string(output)) {
		count = count*10 + int(character-'0')
	}
	return count
}

func TestAutoCommitTopicCommitsThenSkipsWhenClean(t *testing.T) {
	workspace := initAutoCommitWorkspace(t)
	writeOpenTopic(t, workspace, "00001", "topic")
	if _, err := NewService(workspace).MirrorTopic("00001"); err != nil {
		t.Fatal(err)
	}

	result := AutoCommitTopic(workspace, "save", domain.ID("00001"))
	if !result.Committed {
		t.Fatalf("commit = %+v", result)
	}
	if result.ChangedPaths == 0 {
		t.Fatalf("changed paths = %d", result.ChangedPaths)
	}
	wantMessage := "auto: save 00001 ("
	if !strings.HasPrefix(result.Message, wantMessage) || !strings.HasSuffix(result.Message, " path berubah)") {
		t.Fatalf("message = %q", result.Message)
	}
	if len(result.Hash) != 7 {
		t.Fatalf("short hash = %q", result.Hash)
	}
	if count := internalCommitCount(t, workspace.Database); count != 1 {
		t.Fatalf("commit count = %d", count)
	}

	second := AutoCommitTopic(workspace, "save", domain.ID("00001"))
	if second.Committed || second.Reason != "no-changes" {
		t.Fatalf("second commit = %+v", second)
	}
	if count := internalCommitCount(t, workspace.Database); count != 1 {
		t.Fatalf("commit count after skip = %d", count)
	}
}

func TestAutoCommitAllProducesOneAggregateCommit(t *testing.T) {
	workspace := initAutoCommitWorkspace(t)
	writeOpenTopic(t, workspace, "00001", "first")
	writeOpenTopic(t, workspace, "00002", "second")
	if _, err := NewService(workspace).MirrorAllOpen(); err != nil {
		t.Fatal(err)
	}
	result := AutoCommitAll(workspace, "save")
	if !result.Committed {
		t.Fatalf("commit = %+v", result)
	}
	if !strings.HasPrefix(result.Message, "auto: save --all (") {
		t.Fatalf("message = %q", result.Message)
	}
	if result.ChangedPaths < 2 {
		t.Fatalf("changed paths = %d", result.ChangedPaths)
	}
	if count := internalCommitCount(t, workspace.Database); count != 1 {
		t.Fatalf("bulk operation created %d commits", count)
	}
}

func TestAutoCommitSkipsWithoutInternalRepository(t *testing.T) {
	workspace := config.NewWorkspace(t.TempDir())
	result := AutoCommitTopic(workspace, "save", domain.ID("00001"))
	if result.Committed || result.Reason != "no-repository" {
		t.Fatalf("result = %+v", result)
	}
}

func TestAutoCommitReportsMissingIdentityAsWarning(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	workspace, err := config.Initialize(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeOpenTopic(t, workspace, "00001", "topic")
	if _, err := NewService(workspace).MirrorTopic("00001"); err != nil {
		t.Fatal(err)
	}
	result := AutoCommitTopic(workspace, "save", domain.ID("00001"))
	if result.Committed {
		t.Fatalf("commit unexpectedly succeeded: %+v", result)
	}
	if !strings.Contains(strings.ToLower(result.Reason), "identity") {
		t.Fatalf("reason = %q", result.Reason)
	}
	if count := internalCommitCountAllowFailure(t, workspace.Database); count != 0 {
		t.Fatalf("commit count = %d", count)
	}
}

func internalCommitCountAllowFailure(t *testing.T, root string) int {
	t.Helper()
	command := exec.Command("git", "rev-list", "--count", "HEAD")
	command.Dir = root
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
