package shell

import (
	"context"
	"errors"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	"github.com/jterrazz/studio-station/apps/engine/apps/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

type Shell struct{ run local.Runner }

func New(run local.Runner) *Shell { return &Shell{run: run} }

func (s *Shell) Method() catalog.Method    { return catalog.MethodShell }
func (s *Shell) Manager() contract.Manager { return contract.Manager{} }
func (s *Shell) Can() contract.Abilities {
	return contract.Abilities{Install: true, Update: true, Remove: true}
}

func (s *Shell) Ready(context.Context) error { return nil }

func (s *Shell) Inventory(context.Context) (map[string]string, error) {
	return nil, contract.ErrUnlisted
}

func (s *Shell) Outdated(context.Context, []string) (map[string]apps.Currency, error) {
	return nil, nil
}

func (s *Shell) line(ctx context.Context, line string, logs contract.Logs) error {
	c := local.Command("bash", "-c", line)
	c.Logs = logs
	return local.Attach(ctx, s.run, c)
}

func (s *Shell) Install(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	for _, i := range installs {
		v, ok := i.(catalog.ShellInstall)
		if !ok {
			return errors.New(i.Key() + " is not a shell line")
		}
		if err := s.line(ctx, v.Line, logs); err != nil {
			return err
		}
	}
	return nil
}

func (s *Shell) Update(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	// invariant: an update of a line is the line again: it reads no currency,
	// so only an update by name reaches it.
	return s.Install(ctx, installs, logs)
}

func (s *Shell) Remove(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	// invariant: a line is taken back by its `undo` alone; one with none is
	// refused before anything runs, for every install of the call.
	undos := make([]string, 0, len(installs))
	for _, i := range installs {
		v, ok := i.(catalog.ShellInstall)
		if !ok {
			return errors.New(i.Key() + " is not a shell line")
		}
		if v.Undo == "" {
			return contract.ErrCannot
		}
		undos = append(undos, v.Undo)
	}
	for _, undo := range undos {
		if err := s.line(ctx, undo, logs); err != nil {
			return err
		}
	}
	return nil
}
