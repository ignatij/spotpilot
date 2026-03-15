package main

import (
	"os"

	"github.com/ignatij/spotpilot/cmd/spotpilot/commands"
)

func main() {
	bi := buildInfo()
	root := commands.NewRoot(commands.BuildInfo{
		Version:   bi.Version,
		Commit:    bi.Commit,
		BuildDate: bi.BuildDate,
	})
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
