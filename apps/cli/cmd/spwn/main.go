package main

import (
	"fmt"
	"os"

	"github.com/jterrazz/spwn/apps/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
