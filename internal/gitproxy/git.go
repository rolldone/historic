package gitproxy

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	ErrGitUnavailable = errors.New("git executable is unavailable")
	ErrGitConflict    = errors.New("git operation conflict")
)

// Repository is an internal Git repository rooted at .database.
type Repository struct {
	Root string
}

// Open initializes or opens an internal repository without configuring remotes.
func Open(root string) (Repository, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return Repository{}, fmt.Errorf("create git root: %w", err)
	}
	repository := Repository{Root: root}
	if _, err := os.Stat(filepath.Join(root, ".git")); errors.Is(err, os.ErrNotExist) {
		if _, err := repository.run("init"); err != nil {
			return Repository{}, err
		}
	} else if err != nil {
		return Repository{}, fmt.Errorf("inspect git repository: %w", err)
	}
	return repository, nil
}

func (repository Repository) run(args ...string) (string, error) {
	command := exec.Command("git", args...)
	command.Dir = repository.Root
	output, err := command.CombinedOutput()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", ErrGitUnavailable
		}
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

// Add stages allowed workspace changes.
func (repository Repository) AddAll() error {
	_, err := repository.run("add", "--all")
	return err
}

// Commit creates a commit and returns its ID. A clean tree returns ErrNothingToCommit.
func (repository Repository) Commit(message string) (string, error) {
	if strings.TrimSpace(message) == "" {
		return "", fmt.Errorf("%w: commit message is empty", ErrGitConflict)
	}
	status, err := repository.run("status", "--porcelain")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(status) == "" {
		return "", ErrNothingToCommit
	}
	if _, err := repository.run("-c", "user.name=Historic", "-c", "user.email=historic@localhost", "commit", "-m", message); err != nil {
		return "", err
	}
	return repository.Head()
}

var ErrNothingToCommit = errors.New("nothing to commit")

// Head returns the current commit ID.
func (repository Repository) Head() (string, error) {
	output, err := repository.run("rev-parse", "HEAD")
	return strings.TrimSpace(output), err
}

// Log returns one-line commit history.
func (repository Repository) Log(path string) (string, error) {
	if _, err := repository.run("rev-parse", "--verify", "HEAD"); err != nil {
		return "", nil
	}
	args := []string{"log", "--format=%H%x09%aI%x09%s"}
	if path != "" {
		args = append(args, "--", path)
	}
	return repository.run(args...)
}

// Diff returns uncommitted changes, optionally scoped to a path.
func (repository Repository) Diff(path string) (string, error) {
	args := []string{"diff", "--no-ext-diff", "--binary"}
	if path != "" {
		args = append(args, "--", path)
	}
	return repository.run(args...)
}

// Restore replaces a path from a committed snapshot after caller validation.
func (repository Repository) Restore(snapshot, path string) error {
	_, err := repository.run("restore", "--source", snapshot, "--", path)
	return err
}
