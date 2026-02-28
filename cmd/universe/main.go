package main

import (
	"fmt"
	"os"

	"github.com/jterrazz/universe/cmd/universe/commands"
)

func main() {
	if err := commands.Root().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
