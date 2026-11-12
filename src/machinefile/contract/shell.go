package contract

import "context"

// value: how the owner's lines of a phase ended: the shell's exit code and
// what they said, stdout then stderr.

type ShellRun struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

type Shell interface {
	// invariant: the lines run as this machine's user, from its home, in
	// one shell that stops at the first line that fails; an error is a
	// shell that could not run at all.
	Run(ctx context.Context, lines string) (ShellRun, error)
}
