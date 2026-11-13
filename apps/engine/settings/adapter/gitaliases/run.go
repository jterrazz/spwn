package gitaliases

import (
	"context"
	"errors"
	"strconv"

	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

type Attacher interface {
	Attach(ctx context.Context, c local.Cmd, out local.Stream) (local.Attached, error)
}

type Shortcuts struct{ run Attacher }

func NewShortcuts(run Attacher) *Shortcuts { return &Shortcuts{run: run} }

func (s *Shortcuts) Git(ctx context.Context, alias string, words []string) error {
	// value: one alias of the Git shortcuts run by git itself, on the owner's
	// terminal, its words passed as they were typed.
	ran, err := s.run.Attach(ctx, local.Cmd{Name: "git", Args: append([]string{alias}, words...)}, local.ToStdout)
	switch {
	case errors.Is(err, local.ErrNotFound):
		return errors.New("git is not installed")
	case err != nil:
		return err
	case ran.ExitCode != 0:
		return errors.New("git " + alias + " exited " + strconv.Itoa(ran.ExitCode))
	}
	return nil
}
