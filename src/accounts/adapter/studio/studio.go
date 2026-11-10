package studio

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/accounts/contract"
	"github.com/jterrazz/studio-station/src/platform/local"
)

const (
	command = "studio"
	// value: the program a share exports an account with, as the owner
	// names it.
	Command = command
)

type PathLooker interface {
	LookPath(name string) (string, bool)
}

func Present(run PathLooker) bool {
	// value: the command is on this machine's search path.
	_, ok := run.LookPath(command)
	return ok
}

type Runner interface {
	Output(ctx context.Context, c local.Cmd) (local.Output, error)
}

type Options struct {
	// value: the OS CLI keeps its credentials in a file, not the keychain: a
	// VM has none.
	FileBackend bool
	// value: how long one export may take; zero is no limit.
	Budget time.Duration
}

type Credentials struct {
	run Runner
	o   Options
}

var _ contract.SecretSource = Credentials{}

func New(run Runner, o Options) Credentials { return Credentials{run: run, o: o} }

func (c Credentials) Export(ctx context.Context, names []string) ([]byte, error) {
	cmd := local.Cmd{Name: command, Args: append([]string{"credentials", "export"}, names...), Timeout: c.o.Budget}
	if c.o.FileBackend {
		cmd.Env = []string{"STUDIO_CREDENTIALS_BACKEND=file"}
	}
	out, err := c.run.Output(ctx, cmd)
	switch {
	case err != nil:
		return nil, err
	case out.ExitCode != 0:
		if text := strings.TrimSpace(out.Stderr); text != "" {
			return nil, errors.New(text)
		}
		return nil, errors.New("exit " + strconv.Itoa(out.ExitCode))
	}
	return []byte(out.Stdout), nil
}
