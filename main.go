package main

import (
	"errors"
	"os"

	"historic/internal/cmd"
)

func main() {
	root := cmd.NewRootCommand()
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)

	if err := root.Execute(); err != nil {
		var childExit interface{ ExitCode() int }
		if errors.As(err, &childExit) {
			os.Exit(childExit.ExitCode())
		}
		if silent, ok := err.(interface{ Silent() bool }); ok && silent.Silent() {
			os.Exit(1)
		}
		_, _ = os.Stderr.WriteString("historic: " + err.Error() + "\n")
		os.Exit(1)
	}
}
