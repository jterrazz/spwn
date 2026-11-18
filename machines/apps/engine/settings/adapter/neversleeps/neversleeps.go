package neversleeps

import (
	"context"

	awakecontract "github.com/jterrazz/studio-station/apps/engine/awake/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract"
)

type NeverSleeps struct {
	// invariant: Never sleeps writes one preference and nothing of the
	// system: the engine's awake job holds it, so it needs no admin, from
	// any Mac.
	prefs awakecontract.NeverSleepsPrefs
}

var _ contract.Applier = (*NeverSleeps)(nil)

const held = "held by the engine"

func New(prefs awakecontract.NeverSleepsPrefs) *NeverSleeps { return &NeverSleeps{prefs: prefs} }

func (*NeverSleeps) Kind() catalog.EffectKind { return catalog.ProcNeverSleeps.Kind() }

func (n *NeverSleeps) values() []settings.KeyValue {
	on, _ := n.prefs.Get()
	if on {
		return []settings.KeyValue{{Key: "never sleeps", Value: held}}
	}
	return []settings.KeyValue{{Key: "never sleeps", Value: "off"}}
}

func (n *NeverSleeps) Observe(_ context.Context, it catalog.Setting) (settings.Observation, error) {
	o := settings.Observation{State: catalog.StateAbsent, Before: n.values(), After: settings.Sets(it)}
	if on, err := n.prefs.Get(); err != nil {
		return o, err
	} else if on {
		o.State, o.Detail = catalog.StateOK, held
	}
	return o, nil
}

func (n *NeverSleeps) Apply(_ context.Context, it catalog.Setting, _ map[string]string) (settings.Change, error) {
	change := settings.Change{Before: n.values()}
	if err := n.prefs.Set(true); err != nil {
		return change, err
	}
	change.Steps = []settings.ChangeStep{{Label: it.Name, State: catalog.StateOK, Detail: held}}
	change.After = n.values()
	return change, nil
}

func (n *NeverSleeps) Undo(_ context.Context, it catalog.Setting) (settings.Change, error) {
	change := settings.Change{Before: n.values()}
	if err := n.prefs.Set(false); err != nil {
		return change, err
	}
	change.Steps = []settings.ChangeStep{{Label: it.Name, State: catalog.StateOK, Detail: "the Mac sleeps normally"}}
	change.After = n.values()
	return change, nil
}
