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
	declared.LastFired, declared.LastOK, declared.Fires, declared.Failed = time.Time{}, false, 0, 0
	for _, run := range a.store().AutomationRuns[project] {
		if run.Group != declared.Group || run.Name != declared.Name {
			continue
		}
		declared.Fires++
		if !run.OK {
			declared.Failed++
		}
		if run.At.After(declared.LastFired) {
			declared.LastFired, declared.LastOK = run.At, run.OK
		}
	}
	return declared
}

func (a *Automations) all(project string) []automations.Automation {
	declared := a.store().Automations[project]
	out := make([]automations.Automation, 0, len(declared))
	for _, d := range declared {
		out = append(out, a.read(d, project))
	}
	slices.SortFunc(out, func(x, y automations.Automation) int {
		if c := strings.Compare(x.Name, y.Name); c != 0 {
			return c
		}
		return strings.Compare(x.Group, y.Group)
	})
	return automations.Mark(out)
}

func (a *Automations) List(_ context.Context, project string) ([]automations.Automation, error) {
	a.w.Lock()
	defer a.w.Unlock()
	return a.all(project), nil
}

func (a *Automations) Automation(_ context.Context, project, ref string) (automations.Automation, error) {
	a.w.Lock()
	defer a.w.Unlock()
	return automations.Select(a.all(project), ref)
}

func (a *Automations) Log(_ context.Context, project, ref string, last int) ([]automations.Run, error) {
	a.w.Lock()
	defer a.w.Unlock()
	var chosen automations.Automation
	if ref != "" {
		found, err := automations.Select(a.all(project), ref)
		if err != nil {
			return nil, err
		}
		chosen = found
	}
	var out []automations.Run
	for _, run := range a.store().AutomationRuns[project] {
		if ref == "" || (run.Group == chosen.Group && run.Name == chosen.Name) {
			out = append(out, run)
		}
	}
	if last > 0 && len(out) > last {
		out = out[len(out)-last:]
	}
	return out, nil
}
