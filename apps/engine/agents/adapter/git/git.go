package git

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/agents/contract"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

const budget = 5 * time.Second

type Runner interface {
	Output(ctx context.Context, c local.Cmd) (local.Output, error)
}

type Repo struct{ run Runner }

var _ contract.Repo = (*Repo)(nil)

func New(run Runner) *Repo { return &Repo{run: run} }

var shortstat = regexp.MustCompile(`(\d+) files? changed(?:, (\d+) insertions?\(\+\))?(?:, (\d+) deletions?\(-\))?`)

func (r *Repo) Change(ctx context.Context, folder, base string) (agents.Change, error) {
	out, err := r.run.Output(ctx, local.Cmd{Name: "git", Args: []string{"-C", folder, "diff", "--shortstat", base + "...HEAD"}, Timeout: budget})
	if err != nil {
		return agents.Change{}, err
	}
	if out.ExitCode != 0 {
		return agents.Change{}, errors.New("the change of " + folder + " could not be counted: " + strings.TrimSpace(out.Stderr))
	}
	m := shortstat.FindStringSubmatch(out.Stdout)
	if m == nil {
		return agents.Change{}, nil
	}
	n := func(s string) int { v, _ := strconv.Atoi(s); return v }
	return agents.Change{Files: n(m[1]), Plus: n(m[2]), Minus: n(m[3])}, nil
}
