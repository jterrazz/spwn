package calls

import (
	"context"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	appscalls "github.com/jterrazz/studio-station/apps/engine/apps/calls"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	settingscalls "github.com/jterrazz/studio-station/apps/engine/settings/calls"
)

type PlanIn struct {
	// value: the Machinefile, by its name or its path; empty, the one this
	// machine is bound to, else the one named after it.
	Machinefile string
	// value: the phase read: empty, apply's; image, what the VM image
	// bakes; boot, a VM's first start.
	Phase catalog.Phase
}

var Plan = call.Call[Deps, PlanIn, machinefile.Plan]{
	// value: what apply would do here: what it asks of the owner, then
	// each package to install, set, adopt or leave, what is extra, and what
	// has to happen once it is applied. It changes nothing.
	Name: "machinefile.plan", Document: 2, Where: call.AtTarget, Applies: machine.Anywhere,
	Errors: []fail.Code{NotFound, BadPhase},
	Run: func(ctx context.Context, d Deps, in PlanIn, _ call.Progress) (machinefile.Plan, error) {
		if bad := badPhase(in.Phase, false); bad != nil {
			return machinefile.Plan{}, bad
		}
		p, err := judged(ctx, d, in.Machinefile, in.Phase)
		return p.Answered(), err
	},
}

var Drift = call.Call[Deps, PlanIn, machinefile.Plan]{
	// value: the plan read as conditions: each package present, absent,
	// outdated, changed, not managed, extra or unknown. It never acts.
	Name: "machinefile.drift", Document: 2, Where: call.AtTarget, Applies: machine.Anywhere,
	Errors: []fail.Code{NotFound},
	Run: func(ctx context.Context, d Deps, in PlanIn, _ call.Progress) (machinefile.Plan, error) {
		p, err := judged(ctx, d, in.Machinefile, in.Phase)
		return p.Answered(), err
	},
}

// invariant: whether a line run as root asks a password is the machine's
// answer, read once and only for a plan a sudo need is in; a read that
// cannot tell asks it, so a run off a terminal is refused as before.
func judged(ctx context.Context, d Deps, asked string, phase catalog.Phase) (machinefile.Plan, error) {
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
	p := machinefile.Judge(r, cat, seen, ledger, false, phase)
	if p, err = withBlocks(ctx, d, p, cat); err != nil {
		return machinefile.Plan{}, Worded(err)
	}
	// invariant: whether the machine asks a password is read when a run of
	// this plan, an apply or an upgrade, would ask one.
	sudo := func(a machinefile.Asked) bool { return a.Need == catalog.NeedSudo }
	if !slices.ContainsFunc(p.Interactive(true), sudo) && !slices.ContainsFunc(p.UpgradeNeeds(), sudo) {
		return p, nil
	}
	if asked, err := d.Password.Asked(ctx); err == nil && !asked {
		p = machinefile.Judge(r, cat, seen, ledger, true, phase)
		if p, err = withBlocks(ctx, d, p, cat); err != nil {
			return machinefile.Plan{}, Worded(err)
		}
	}
	return p, nil
}

// invariant: what each package reads is its family's own read, the one
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
	s := machinefile.Seen{State: r.State, Method: string(r.Method), Version: r.Version, Detail: r.Detail, Irreversible: r.Irreversible, Holds: r.Holds, HoldsUnread: r.HoldsUnread}
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
