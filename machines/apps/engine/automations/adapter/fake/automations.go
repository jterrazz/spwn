package fake

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/automations"
	"github.com/jterrazz/studio-station/apps/engine/automations/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

type Automations struct {
	// invariant: the Automations one machine of the world keeps; what the
	// runs say of an automation is read from the runs, never stored.
	w       *world.World
	machine string
}

var _ contract.Automations = (*Automations)(nil)

func Of(w *world.World, machine string) *Automations {
	return &Automations{w: w, machine: machine}
}

func (a *Automations) store() *world.Stores { return a.w.Mac(a.machine).Store() }

func (a *Automations) read(declared automations.Automation, project string) automations.Automation {
	declared.LastFired, declared.Fires, declared.Failed = time.Time{}, 0, 0
	for _, run := range a.store().AutomationRuns[project] {
		if run.Name != declared.Name {
			continue
		}
		declared.Fires++
		if !run.OK {
			declared.Failed++
		}
		if run.At.After(declared.LastFired) {
			declared.LastFired = run.At
		}
	}
	return declared
}

func (a *Automations) List(_ context.Context, project string) ([]automations.Automation, error) {
	a.w.Lock()
	defer a.w.Unlock()
	declared := a.store().Automations[project]
	out := make([]automations.Automation, 0, len(declared))
	for _, d := range declared {
		out = append(out, a.read(d, project))
	}
	slices.SortFunc(out, func(x, y automations.Automation) int { return strings.Compare(x.Name, y.Name) })
	return out, nil
}

func (a *Automations) Automation(_ context.Context, project, name string) (automations.Automation, error) {
	a.w.Lock()
	defer a.w.Unlock()
	for _, d := range a.store().Automations[project] {
		if d.Name == name {
			return a.read(d, project), nil
		}
	}
	return automations.Automation{}, automations.ErrNotFound
}

func (a *Automations) Log(_ context.Context, project, name string, last int) ([]automations.Run, error) {
	a.w.Lock()
	defer a.w.Unlock()
	if name != "" && !slices.ContainsFunc(a.store().Automations[project], func(d automations.Automation) bool { return d.Name == name }) {
		return nil, automations.ErrNotFound
	}
	var out []automations.Run
	for _, run := range a.store().AutomationRuns[project] {
		if name == "" || run.Name == name {
			out = append(out, run)
		}
	}
	if last > 0 && len(out) > last {
		out = out[len(out)-last:]
	}
	return out, nil
}
