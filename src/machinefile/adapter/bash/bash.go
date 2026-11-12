package bash

import (
	"context"
	"time"

	"github.com/jterrazz/studio-station/src/machinefile/contract"
	"github.com/jterrazz/studio-station/src/platform/local"
)

// value: the longest the owner's lines of one phase run.
const bound = 20 * time.Minute

type Bash struct {
	run  local.Runner
	home string
}

var _ contract.Shell = (*Bash)(nil)

func New(run local.Runner, home string) *Bash { return &Bash{run: run, home: home} }

func (b *Bash) Run(ctx context.Context, lines string) (contract.ShellRun, error) {
	c := local.Command("bash", "-e", "-c", lines)
	c.Dir, c.Timeout = b.home, bound
	out, err := b.run.Output(ctx, c)
	if err != nil {
		return contract.ShellRun{}, err
	}
	return contract.ShellRun{ExitCode: out.ExitCode, Stdout: out.Stdout, Stderr: out.Stderr}, nil
}
