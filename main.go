package main

import (
	"os"

	"github.com/robotx-dev/cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(cmd.HandleError(err))
	}
}
