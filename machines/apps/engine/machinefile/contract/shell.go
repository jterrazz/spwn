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
	// one shell, so a cd or an export carries to the next line; the first
	// command whose status is not zero (a line, or the lines of one
	// construct), the left of an `&&` and a pipe included, ends the run
	// with that status, and the last line on stderr names it
	// (`station: exit <n> at: <its first line>`); an error is a shell that
	// could not run at all.
	Run(ctx context.Context, lines string) (ShellRun, error)
}
