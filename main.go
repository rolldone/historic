package main

import (
	"os"

	"historic/internal/cmd"
)

func main() {
	root := cmd.NewRootCommand()
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)

	if err := root.Execute(); err != nil {
		_, _ = os.Stderr.WriteString("historic: " + err.Error() + "\n")
		os.Exit(1)
	}
}
