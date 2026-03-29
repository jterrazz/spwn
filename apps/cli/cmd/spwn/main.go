package main

import (
	"fmt"
	"os"

	"spwn.sh/apps/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
