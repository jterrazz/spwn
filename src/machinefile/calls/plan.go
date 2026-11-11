package calls

import (
	"context"

	"github.com/jterrazz/studio-station/src/apps"
	appscalls "github.com/jterrazz/studio-station/src/apps/calls"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/machinefile"
	"github.com/jterrazz/studio-station/src/settings"
	settingscalls "github.com/jterrazz/studio-station/src/settings/calls"
)

type PlanIn struct {
	// value: the Machinefile, by its name or its path; empty, the one this
	// machine is bound to, else the one named after it.
	Machinefile string
}

var Plan = call.Call[Deps, PlanIn, machinefile.Plan]{
	// value: what apply would do here: what it asks of the owner, then
	// each item to install, set, adopt or leave, what is extra, and what
	// has to happen once it is applied. It changes nothing.
	Name: "machinefile.plan", Where: call.AtTarget, Applies: machine.Anywhere,
	Errors: []fail.Code{NotFound},
	Run: func(ctx context.Context, d Deps, in PlanIn, _ call.Progress) (machinefile.Plan, error) {
		return judged(ctx, d, in.Machinefile)
	},
}

var Drift = call.Call[Deps, PlanIn, machinefile.Plan]{
	// value: the plan read as conditions: each item present, absent,
	// outdated, changed, not managed, extra or unknown. It never acts.
	Name: "machinefile.drift", Where: call.AtTarget, Applies: machine.Anywhere,
	Errors: []fail.Code{NotFound},
	Run: func(ctx context.Context, d Deps, in PlanIn, _ call.Progress) (machinefile.Plan, error) {
		return judged(ctx, d, in.Machinefile)
	},
}

func judged(ctx context.Context, d Deps, asked string) (machinefile.Plan, error) {
	r, cat, err := resolveIn(ctx, d, asked)
	if err != nil {
		return machinefile.Plan{}, err
	}
	seen, err := observe(ctx, d, appscalls.ListIn{})
	if err != nil {
		return machinefile.Plan{}, err
	}
	ledger, err := d.Ledger.List(ctx)
	if err != nil {
		return machinefile.Plan{}, err
	}
	return machinefile.Judge(r, cat, seen, ledger), nil
}

// invariant: what each item reads is its family's own read, the one
// `station apps` and `station settings` print: apps' inventory and
// settings' judge, through this machine's calls.
func observe(ctx context.Context, d Deps, in appscalls.ListIn) (map[string]machinefile.Seen, error) {
	self := machine.Ref{}
	reports, err := call.Read(ctx, d.Calls, self, appscalls.List, in, nil)
	if err != nil {
		return nil, err
	}
	rows, err := call.Read(ctx, d.Calls, self, settingscalls.List, settingscalls.ListIn{}, nil)
	if err != nil {
		return nil, err
	}
	seen := map[string]machinefile.Seen{}
	for _, r := range reports {
		seen[r.App.Name] = appSeen(r)
	}
	for _, r := range rows {
		if r.Group != settings.Checked {
			seen[r.Name] = settingSeen(r)
		}
	}
	return seen, nil
}

func appSeen(r apps.Report) machinefile.Seen {
	s := machinefile.Seen{State: r.State, Method: string(r.Method), Version: r.Version, Detail: r.Detail}
	if r.Why == apps.WhyMissing {
		s.Lacks = r.Needs
	}
	switch apps.StateOf(r) {
	case apps.OK:
		s.Condition = machinefile.Present
	case apps.UpdateAvailable:
		s.Condition, s.Version = machinefile.Outdated, r.Version
	case apps.Failed:
		s.Condition = machinefile.Changed
	case apps.NotInstalled:
		s.Condition, s.State = machinefile.Absent, catalog.StateAbsent
	case apps.NotManaged:
		s.Condition = machinefile.NotManaged
	case apps.NotKnown, apps.Pending:
		s.Condition = machinefile.Unknown
	}
	return s
}

func settingSeen(r settings.Row) machinefile.Seen {
	s := machinefile.Seen{Detail: r.Why}
	switch r.State {
	case settings.StateApplied:
		s.Condition, s.State, s.Detail = machinefile.Present, catalog.StateOK, r.Where
	case settings.StateChangedSince:
		s.Condition, s.State, s.Detail = machinefile.Changed, catalog.StateDegraded, r.Where
	case settings.StateNotApplied:
		s.Condition, s.State = machinefile.Absent, catalog.StateAbsent
	case settings.StateOffThisMac:
		s.Condition, s.State, s.Detail = machinefile.Absent, catalog.StateDegraded, "off here, your choice"
	case settings.StateNotForThisMachine, settings.StateReading:
		s.Condition, s.State = machinefile.Unknown, catalog.StateUnknown
	}
	return s
}
