package main

import (
	"os"

	"spwn.sh/apps/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
