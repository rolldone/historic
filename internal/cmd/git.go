package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func newGitCommand() *cobra.Command {
	return &cobra.Command{
		Use:                "git <git-args...>",
		Short:              "Run Git in the internal Historic repository",
		DisableFlagParsing: true,
		Args:               cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Usage: historic git <git-args...>\nExample: historic git status")
				return SilentError{Err: errors.New("git arguments are required")}
			}

			workspace, err := commandWorkspaceForRecovery()
			if err != nil {
				return err
			}
			info, err := os.Stat(workspace.Database)
			if err != nil {
				return fmt.Errorf("inspect internal Git directory %s: %w (run historic init if the workspace is not initialized)", workspace.Database, err)
			}
			if !info.IsDir() {
				return fmt.Errorf("internal Git path %s is not a directory", workspace.Database)
			}

			gitPath, err := exec.LookPath("git")
			if err != nil {
				return fmt.Errorf("Git executable is unavailable on PATH; install Git and retry: %w", err)
			}
			process := exec.Command(gitPath, args...)
			process.Dir = workspace.Database
			process.Stdin = cmd.InOrStdin()
			process.Stdout = cmd.OutOrStdout()
			process.Stderr = cmd.ErrOrStderr()
			if err := process.Run(); err != nil {
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					return gitChildExitError{err: err, code: exitErr.ExitCode()}
				}
				return fmt.Errorf("run Git: %w", err)
			}
			return nil
		},
	}
}

type gitChildExitError struct {
	err  error
	code int
}

func (err gitChildExitError) Error() string { return err.err.Error() }
func (err gitChildExitError) Unwrap() error { return err.err }
func (err gitChildExitError) ExitCode() int { return err.code }
func (err gitChildExitError) Silent() bool  { return true }
