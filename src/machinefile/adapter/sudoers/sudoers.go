package sudoers

import (
	"context"
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/machinefile/contract"
	"github.com/jterrazz/studio-station/src/platform/local"
)

// value: how long the read waits for sudo's answer: it reads a rule, so a
// longer wait is a sudo stuck on its own lookups, read as asking.
const budget = 5 * time.Second

type Password struct{ run local.Runner }

var _ contract.Password = (*Password)(nil)

func New(run local.Runner) *Password { return &Password{run: run} }

func (p *Password) Asked(ctx context.Context) (bool, error) {
	// outside: `-n` makes sudo fail rather than prompt (Touch ID included)
	// and `-k` ignores a password typed minutes ago, so the exit status says
	// whether the sudoers rule itself asks one; `true` is all it runs.
	out, err := p.run.Output(ctx, local.Cmd{Name: catalog.Sudo, Args: []string{"-k", "-n", "true"}, Timeout: budget})
	if err != nil {
		return true, err
	}
	return out.ExitCode != 0, nil
}
