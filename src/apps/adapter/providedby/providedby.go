package providedby

import (
	"context"
	"errors"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local"
)

type ProvidedBy struct {
	// invariant: a command that comes with another item's install runs
	// nothing of its own: the item that provides it is its requirement,
	// installed before it, and its install is that its commands are on the
	// search path now.
	run local.Runner
}

func New(run local.Runner) *ProvidedBy { return &ProvidedBy{run: run} }

func (*ProvidedBy) Method() catalog.Method    { return catalog.MethodProvidedBy }
func (*ProvidedBy) Manager() contract.Manager { return contract.Manager{} }
func (*ProvidedBy) Can() contract.Abilities   { return contract.Abilities{Install: true} }

func (*ProvidedBy) Ready(context.Context) error { return nil }

func (*ProvidedBy) Inventory(context.Context) (map[string]string, error) {
	return nil, contract.ErrUnlisted
}

func (*ProvidedBy) Outdated(context.Context, []string) (map[string]apps.Currency, error) {
	return nil, nil
}

func (p *ProvidedBy) Install(_ context.Context, installs []catalog.Install, _ contract.Logs) error {
	for _, i := range installs {
		v, ok := i.(catalog.ProvidedByInstall)
		if !ok {
			return errors.New(i.Key() + " is not provided by another item")
		}
		for _, name := range v.Commands {
			if _, found := p.run.LookPath(name); !found {
				return errors.New(name + " did not come with " + v.Item)
			}
		}
	}
	return nil
}

func (*ProvidedBy) Update(context.Context, []catalog.Install, contract.Logs) error {
	return contract.ErrCannot
}

func (*ProvidedBy) Remove(context.Context, []catalog.Install, contract.Logs) error {
	return contract.ErrCannot
}
