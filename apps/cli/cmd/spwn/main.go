package main

import (
	"errors"
	"os"

	"spwn.sh/apps/cli"
)

func main() {
	cli.EnsureDockerFriendlyPATH()

	if err := cli.Execute(); err != nil {
		var coder cli.ExitCoder
		if errors.As(err, &coder) {
			os.Exit(coder.ExitCode())
		}
		os.Exit(1)
	}
}
