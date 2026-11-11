package local

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

func Command(name string, args ...string) Cmd { return Cmd{Name: name, Args: args} }

func Attach(ctx context.Context, r Runner, c Cmd) error {
	// invariant: a command attached to the owner's terminal fails by its name and
	// first word: "bun install exited 1", or "bun: <why>" when it could not run.
	ran, err := r.Attach(ctx, c, ToStderr)
	switch {
	case err != nil:
		return fmt.Errorf("%s: %w", c.Name, err)
	case ran.ExitCode != 0:
		return &ExitError{Step: c.Name + " " + first(c.Args), Code: ran.ExitCode, LastStderr: ran.LastStderr}
	}
	return nil
}

type ExitError struct {
	Step       string
	Code       int
	LastStderr string
}

func (e *ExitError) Error() string { return e.Step + " exited " + strconv.Itoa(e.Code) }

func first(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func Played(play func(c Cmd) (Output, error), names ...string) Runner {
	// value: a Runner answered by a function, as an adapter's test plays its
	// tool: a name it answers for is on PATH, any other is not found.
	return played{play: play, names: names}
}

type played struct {
	play  func(Cmd) (Output, error)
	names []string
}

func (p played) on(name string) bool { return slices.Contains(p.names, name) }

func (p played) Output(_ context.Context, c Cmd) (Output, error) {
	if !p.on(c.Name) {
		return Output{}, ErrNotFound
	}
	return p.play(c)
}

func (p played) Attach(ctx context.Context, c Cmd, _ Stream) (Attached, error) {
	out, err := p.Output(ctx, c)
	last := ""
	for _, line := range strings.Split(out.Stderr, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			last = line
		}
	}
	return Attached{ExitCode: out.ExitCode, LastStderr: last}, err
}

func (p played) Start(Cmd, string) (int, error) { return 0, errors.ErrUnsupported }

func (p played) LookPath(name string) (string, bool) {
	if !p.on(name) {
		return "", false
	}
	return "/played/" + name, true
}

func (p played) Alive(int) bool        { return false }
func (p played) Signal(int, Sig) error { return errors.ErrUnsupported }
func (p played) Reap(string) int       { return 0 }

func Anchored(r Runner, dir string, env ...string) Runner {
	// value: a Runner whose commands run from dir, unless one names its own
	// folder, with env before their own, so a command's own Env still wins.
	return anchored{Runner: r, dir: dir, env: env}
}

type anchored struct {
	Runner
	dir string
	env []string
}

func (a anchored) at(c Cmd) Cmd {
	if c.Dir == "" {
		c.Dir = a.dir
	}
	c.Env = append(append([]string{}, a.env...), c.Env...)
	return c
}

func (a anchored) Output(ctx context.Context, c Cmd) (Output, error) {
	return a.Runner.Output(ctx, a.at(c))
}

func (a anchored) Attach(ctx context.Context, c Cmd, out Stream) (Attached, error) {
	return a.Runner.Attach(ctx, a.at(c), out)
}

func (a anchored) Start(c Cmd, logPath string) (int, error) { return a.Runner.Start(a.at(c), logPath) }

func Read(ctx context.Context, r Runner, c Cmd) (string, bool) {
	// value: a read: the trimmed stdout, ok only when the command ran and
	// exited 0.
	out, err := r.Output(ctx, c)
	if err != nil || out.ExitCode != 0 {
		return strings.TrimSpace(out.Stdout), false
	}
	return strings.TrimSpace(out.Stdout), true
}

func Must(ctx context.Context, r Runner, c Cmd) (string, error) {
	// invariant: a command that exits non-zero fails with its argv and the
	// first line of why: "systemsetup -setremotelogin on: <why>".
	out, err := r.Output(ctx, c)
	if err != nil {
		return "", err
	}
	if out.ExitCode == 0 {
		return strings.TrimSpace(out.Stdout), nil
	}
	reason := firstLine(out.Stderr)
	if reason == "" {
		reason = firstLine(out.Stdout)
	}
	if reason == "" {
		reason = "exited " + strconv.Itoa(out.ExitCode)
	}
	return strings.TrimSpace(out.Stdout), errors.New(strings.Join(append([]string{c.Name}, c.Args...), " ") + ": " + reason)
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}
