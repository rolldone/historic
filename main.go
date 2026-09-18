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
		if silent, ok := err.(interface{ Silent() bool }); ok && silent.Silent() {
			os.Exit(1)
		}
		_, _ = os.Stderr.WriteString("historic: " + err.Error() + "\n")
		os.Exit(1)
	}
}
