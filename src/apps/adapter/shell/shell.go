package shell

import (
	"context"
	"errors"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local"
)

type Shell struct{ run local.Runner }

func New(run local.Runner) *Shell { return &Shell{run: run} }

func (s *Shell) Method() catalog.Method    { return catalog.MethodShell }
func (s *Shell) Manager() contract.Manager { return contract.Manager{} }
func (s *Shell) Can() contract.Abilities   { return contract.Abilities{Install: true} }

func (s *Shell) Ready(context.Context) error { return nil }

func (s *Shell) Inventory(context.Context) (map[string]string, error) {
	return nil, contract.ErrUnlisted
}

func (s *Shell) Outdated(context.Context, []string) (map[string]apps.Currency, error) {
	return nil, nil
}

func (s *Shell) Install(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	for _, i := range installs {
		v, ok := i.(catalog.ShellInstall)
		if !ok {
			return errors.New(i.Key() + " is not a shell line")
		}
		c := local.Command("bash", "-c", v.Line)
		c.Logs = logs
		if err := local.Attach(ctx, s.run, c); err != nil {
			return err
		}
	}
	return nil
}

func (s *Shell) Update(context.Context, []catalog.Install, contract.Logs) error {
	return contract.ErrCannot
}

func (s *Shell) Remove(context.Context, []catalog.Install, contract.Logs) error {
	return contract.ErrCannot
}
