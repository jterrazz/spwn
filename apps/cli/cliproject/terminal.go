package cliproject

import (
	"os"

	"golang.org/x/term"
)

// AskTerminal reports whether the answer can be asked of a person: the
// prompt is written to stderr and the answer read from stdin, so both
// must be terminals; `2>file` alone leaves nobody seeing the prompt.
func AskTerminal() bool {
	return bothTerminals(term.IsTerminal, int(os.Stdin.Fd()), int(os.Stderr.Fd()))
}

func bothTerminals(isTerminal func(fd int) bool, stdin, stderr int) bool {
	return isTerminal(stdin) && isTerminal(stderr)
}
