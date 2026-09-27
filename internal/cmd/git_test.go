package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitCommandPassesThroughArgumentsStreamsAndExitCode(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	database := filepath.Join(root, ".historic", ".database")
	if err := os.MkdirAll(database, 0o755); err != nil {
		t.Fatal(err)
	}

	bin := t.TempDir()
	captureArgs := filepath.Join(root, "args.txt")
	captureCwd := filepath.Join(root, "cwd.txt")
	captureStdin := filepath.Join(root, "stdin.txt")
	t.Setenv("CAPTURE_ARGS", captureArgs)
	t.Setenv("CAPTURE_CWD", captureCwd)
	t.Setenv("CAPTURE_STDIN", captureStdin)
	t.Setenv("CHILD_EXIT", "7")
	writeGitStub(t, bin, `#!/bin/sh
printf '%s\n' "$PWD" > "$CAPTURE_CWD"
printf '%s\n' "$@" > "$CAPTURE_ARGS"
printf 'child-stdout\n'
printf 'child-stderr\n' >&2
IFS= read -r input
printf '%s' "$input" > "$CAPTURE_STDIN"
exit "$CHILD_EXIT"
`)
	t.Setenv("PATH", bin)

	rootCommand := NewRootCommand()
	var stdout, stderr bytes.Buffer
	ConfigureOutput(rootCommand, &stdout, &stderr)
	rootCommand.SetIn(strings.NewReader("stdin-payload\n"))
	rootCommand.SetArgs([]string{"git", "status", "--short", "path with spaces", "$(touch injected-file);"})
	err := rootCommand.Execute()
	if err == nil {
		t.Fatal("expected child exit status to be returned")
	}
	var childExit interface{ ExitCode() int }
	if !errors.As(err, &childExit) || childExit.ExitCode() != 7 {
		t.Fatalf("error = %T %v, want child exit code 7", err, err)
	}
	if got := stdout.String(); got != "child-stdout\n" {
		t.Fatalf("stdout = %q", got)
	}
	if got := stderr.String(); got != "child-stderr\n" {
		t.Fatalf("stderr = %q", got)
	}
	if got, want := readTestFile(t, captureCwd), database+"\n"; got != want {
		t.Fatalf("Git cwd = %q, want %q", got, want)
	}
	if got, want := readTestFile(t, captureArgs), "status\n--short\npath with spaces\n$(touch injected-file);\n"; got != want {
		t.Fatalf("Git args = %q, want %q", got, want)
	}
	if got := readTestFile(t, captureStdin); got != "stdin-payload" {
		t.Fatalf("stdin = %q", got)
	}
	if _, err := os.Stat(filepath.Join(root, "injected-file")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("shell expression was evaluated (stat err = %v)", err)
	}
}

func TestGitCommandRequiresArguments(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	rootCommand := NewRootCommand()
	var stdout bytes.Buffer
	ConfigureOutput(rootCommand, &stdout, &stdout)
	rootCommand.SetArgs([]string{"git"})
	err := rootCommand.Execute()
	if err == nil || !strings.Contains(err.Error(), "git arguments are required") {
		t.Fatalf("error = %v, want missing arguments error", err)
	}
	if !strings.Contains(stdout.String(), "Usage: historic git <git-args...>") || !strings.Contains(stdout.String(), "historic git status") {
		t.Fatalf("usage output = %q", stdout.String())
	}
}

func TestGitCommandRejectsMissingDatabaseWithoutCreatingIt(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if err := os.Mkdir(filepath.Join(root, ".historic"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := executeCommand(t, "git", "status")
	if err == nil || !strings.Contains(err.Error(), "internal Git directory") {
		t.Fatalf("error = %v, want missing database error", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, ".historic", ".database")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("database was unexpectedly created: %v", statErr)
	}
}

func TestGitCommandReportsUnavailableExecutable(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if err := os.MkdirAll(filepath.Join(root, ".historic", ".database"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", t.TempDir())
	_, err := executeCommand(t, "git", "status")
	if err == nil || !strings.Contains(err.Error(), "Git executable is unavailable") {
		t.Fatalf("error = %v, want unavailable executable error", err)
	}
}

func TestGitCommandForwardsNotRepositoryFailure(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if err := os.MkdirAll(filepath.Join(root, ".historic", ".database"), 0o755); err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	writeGitStub(t, bin, `#!/bin/sh
printf 'fatal: not a git repository (or any parent up to mount point): .git\\n' >&2
exit 128
`)
	t.Setenv("PATH", bin)

	rootCommand := NewRootCommand()
	var stdout, stderr bytes.Buffer
	ConfigureOutput(rootCommand, &stdout, &stderr)
	rootCommand.SetArgs([]string{"git", "status"})
	err := rootCommand.Execute()
	var childExit interface{ ExitCode() int }
	if !errors.As(err, &childExit) || childExit.ExitCode() != 128 {
		t.Fatalf("error = %v, want child exit code 128", err)
	}
	if !strings.Contains(stderr.String(), "fatal: not a git repository") {
		t.Fatalf("stderr = %q, want not-a-repository diagnostic", stderr.String())
	}
}

func TestGitCommandIsRegisteredWithoutReplacingInternalCommands(t *testing.T) {
	root := NewRootCommand()
	for _, name := range []string{"git", "log", "diff", "save", "restore"} {
		if _, _, err := root.Find([]string{name}); err != nil {
			t.Errorf("find %q: %v", name, err)
		}
	}
	var help bytes.Buffer
	ConfigureOutput(root, &help, &help)
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(help.String(), "Run Git in the internal Historic repository") {
		t.Fatalf("root help does not list git command: %q", help.String())
	}
}

func writeGitStub(t *testing.T, bin, script string) {
	t.Helper()
	path := filepath.Join(bin, "git")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(fmt.Errorf("read %s: %w", path, err))
	}
	return string(contents)
}
