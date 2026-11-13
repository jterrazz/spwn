package muxmachines

import (
	"context"
	"slices"
	"strings"

	agentscontract "github.com/jterrazz/studio-station/apps/engine/agents/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract"
)

type Applier struct {
	// invariant: the workspaces are this Mac's, by name; a machine the multiplexer
	// reaches that is no workspace of this Mac is the owner's, never touched.
	multiplexer agentscontract.Multiplexer
	workspaces  func(ctx context.Context) ([]string, error)
}

var _ contract.Applier = (*Applier)(nil)

func New(multiplexer agentscontract.Multiplexer, workspaces func(ctx context.Context) ([]string, error)) *Applier {
	return &Applier{multiplexer: multiplexer, workspaces: workspaces}
}

func (*Applier) Kind() catalog.EffectKind { return catalog.EffectMuxMachines }

func (a *Applier) read(ctx context.Context) (workspaces, reached, missing []string, err error) {
	if workspaces, err = a.workspaces(ctx); err != nil {
		return nil, nil, nil, err
	}
	if reached, err = a.multiplexer.Machines(ctx); err != nil {
		return nil, nil, nil, err
	}
	for _, w := range workspaces {
		if !slices.Contains(reached, w) {
			missing = append(missing, w)
		}
	}
	return workspaces, reached, missing, nil
}

func (a *Applier) Observe(ctx context.Context, _ catalog.Setting) (settings.Observation, error) {
	workspaces, reached, missing, err := a.read(ctx)
	if err != nil {
		return settings.Observation{}, err
	}
	// invariant: the fingerprint is the workspaces the multiplexer reaches
	// now, never the ones station would add: one the owner took out moves
	// it, and a new workspace station has yet to add does not.
	var there []string
	for _, w := range workspaces {
		if slices.Contains(reached, w) {
			there = append(there, w)
		}
	}
	o := settings.Observation{Digest: "reached:" + strings.Join(there, ",")}
	for _, w := range missing {
		o.Before = append(o.Before, settings.KeyValue{Key: w, Value: "not reached"})
		o.After = append(o.After, settings.KeyValue{Key: w, Value: "reached"})
	}
	switch {
	case len(missing) == 0:
		o.State, o.Detail = catalog.StateOK, "reaches "+strings.Join(workspaces, ", ")
		if len(workspaces) == 0 {
			o.Detail = "no workspace to reach"
		}
	case len(missing) == len(workspaces):
		o.State, o.Detail = catalog.StateAbsent, "reaches none of "+strings.Join(workspaces, ", ")
	default:
		o.State, o.Detail = catalog.StateDegraded, "does not reach "+strings.Join(missing, ", ")
	}
	return o, nil
}

func (a *Applier) Apply(ctx context.Context, _ catalog.Setting, _ map[string]string) (settings.Change, error) {
	_, _, missing, err := a.read(ctx)
	if err != nil {
		return settings.Change{}, err
	}
	var c settings.Change
	for _, w := range missing {
		if err := a.multiplexer.Add(ctx, w); err != nil {
			return c, err
		}
		c.Steps = append(c.Steps, settings.ChangeStep{Label: w, State: catalog.StateOK, Detail: "reached"})
	}
	return c, nil
}

func (a *Applier) Undo(ctx context.Context, _ catalog.Setting) (settings.Change, error) {
	// invariant: off forgets this Mac's workspaces alone; their terminals
	// keep running in them.
	workspaces, reached, _, err := a.read(ctx)
	if err != nil {
		return settings.Change{}, err
	}
	var c settings.Change
	for _, w := range workspaces {
		if !slices.Contains(reached, w) {
			continue
		}
		if err := a.multiplexer.Remove(ctx, w); err != nil {
			return c, err
		}
		c.Steps = append(c.Steps, settings.ChangeStep{Label: w, State: catalog.StateOK, Detail: "forgotten"})
	}
	return c, nil
}
