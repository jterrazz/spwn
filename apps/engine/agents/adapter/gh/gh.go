package gh

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/agents/contract"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

const budget = 60 * time.Second

type Runner interface {
	Output(ctx context.Context, c local.Cmd) (local.Output, error)
}

type Forge struct{ run Runner }

var _ contract.Forge = (*Forge)(nil)

func New(run Runner) *Forge { return &Forge{run: run} }

func (f *Forge) Merge(ctx context.Context, folder string, number int) error {
	// value: a merge commit, so the PR's commits, each a unit of meaning,
	// land as they were written.
	out, err := f.run.Output(ctx, local.Cmd{Name: "gh", Args: []string{"pr", "merge", strconv.Itoa(number), "--merge"}, Dir: folder, Timeout: budget})
	if err != nil {
		return err
	}
	if out.ExitCode != 0 {
		return errors.New("PR #" + strconv.Itoa(number) + " was not merged: " + strings.TrimSpace(out.Stderr))
	}
	return nil
}
