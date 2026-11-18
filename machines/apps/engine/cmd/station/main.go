package main

import (
	"os"

	"github.com/jterrazz/studio-station/apps/engine/wire"
)

func main() {
	os.Exit(wire.Run(os.Args[1:], os.Environ(), os.Stdin, os.Stdout, os.Stderr))
}
