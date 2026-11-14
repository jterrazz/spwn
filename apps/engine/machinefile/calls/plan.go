package calls

import (
	"cmp"
	"context"
	"errors"
	"maps"
	"slices"
	"strings"

	accountscalls "github.com/jterrazz/studio-station/apps/engine/accounts/calls"
	"github.com/jterrazz/studio-station/apps/engine/apps"
	appscalls "github.com/jterrazz/studio-station/apps/engine/apps/calls"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/ports"
	portscalls "github.com/jterrazz/studio-station/apps/engine/ports/calls"
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

type DriftIn struct {
	// value: the machine read, a Mac or a VM, by its name; empty, This Mac.
	Machine string
	// value: the Machinefile, by its name or its path; empty, the one the
	// machine is bound to, else the one named after it.
	Machinefile string
}

var Drift = call.Call[Deps, DriftIn, machinefile.Drift]{
	// value: the plan read as conditions (each package present, absent,
	// outdated, changed, not managed, extra or unknown), then counted by
	// chapter, a VM's ports and accounts included, with what apply would do
	// and the verdict. It never acts.
	Name: "machinefile.drift", Document: 3, Where: call.InProcess, Applies: machine.Anywhere,
	Errors: []fail.Code{NotFound},
	Run: func(ctx context.Context, d Deps, in DriftIn, _ call.Progress) (answer machinefile.Drift, err error) {
		defer func() { answer = answer.Answered() }()
		// invariant: the machine is the input's, never a target, since a
		// call with a target runs inside it, where its Mac's shares do not
		// read: its packages are judged on it, and a VM's ports and
		// accounts are read from its Mac. A drift asked inside a VM reads
		// its packages alone, and its verdict says so; so does a VM whose
		// Mac answered its plan but not its shares. A VM is reached through
		// its Mac: one whose Mac does not answer is refused as that silence.
		name := cmp.Or(in.Machine, d.Name)
		p, err := plannedOn(ctx, d, name, in.Machinefile)
		if err != nil {
			return machinefile.Drift{}, err
		}
		mac, vm := "", false
		if name != d.Name && d.MacOf != nil {
			mac, vm = d.MacOf(ctx, name)
		}
		if !vm {
			drift := machinefile.DriftOf(p, nil)
			drift.Machine = name
			if d.Kind != machine.Computer {
				drift = drift.AccessUnread()
			}
			return drift, nil
		}
		access, err := shares(ctx, d, name, mac, p.Name)
		drift := machinefile.DriftOf(p, access)
		if err != nil {
			drift = drift.AccessUnread()
		}
		drift.Machine = name
		return drift, nil
	},
}

func plannedOn(ctx context.Context, d Deps, name, file string) (machinefile.Plan, error) {
	if name == d.Name {
		return judged(ctx, d, file, "")
	}
	return call.Invoke(ctx, d.Far, machine.Ref{Name: name}, Plan, PlanIn{Machinefile: file}, nil)
}

var errUnanswered = errors.New("its Mac did not answer its ports")

// invariant: a VM's shares are its Mac's: its ports read through every
// Mac with no target and kept for it, a listing that came with an error
// kept when it holds Macs; they are unread unless the listing holds its
// Mac answering, its engine running (This Mac's entry for a VM of This
// Mac, else the Mac its name names, in any case). Its accounts are asked
// of its Mac, the first reading carrying them; what its file gives it is
// read from This Mac's sources, as the Mac's rights read it. A port it
// listens on inside its rights and does not share is absent. A server's
// ports and accounts are what it runs, never its file's: they are not
// read, so not counted.
func answering(h ports.MacPorts, mac string, thisMac bool) bool {
	named := (h.ThisMac && thisMac) || (!h.ThisMac && !thisMac && strings.EqualFold(h.Name, mac))
	return named && (h.ThisMac || h.Reachable) && h.Engine.Running
}

func shares(ctx context.Context, d Deps, vm, mac, file string) (*machinefile.Access, error) {
	lib, err := library(ctx, d)
	if err != nil {
		return nil, err
	}
	asked, err := machinefile.GivenOf(lib, file)
	if err != nil {
		return nil, err
	}
	if asked.Server {
		return nil, nil
	}
	listing, err := call.Invoke(ctx, d.Far, machine.Ref{}, portscalls.List, portscalls.ListIn{Of: vm}, nil)
	if err != nil && len(listing.AllMachines.Macs) == 0 {
		return nil, err
	}
	if !slices.ContainsFunc(listing.AllMachines.Macs, func(h ports.MacPorts) bool { return answering(h, mac, mac == d.Name) }) {
		return nil, errUnanswered
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	updates, err := call.Follow(ctx, d.Far, machine.Ref{Name: mac}, accountscalls.List, accountscalls.ListIn{}, nil)
	if err != nil {
		return nil, err
	}
	first, ok := <-updates
	if !ok {
		return nil, ctx.Err()
	}
	if first.Err != nil {
		return nil, first.Err
	}
	rights := ports.RightsOf(asked.Ports, asked.Server)
	access := &machinefile.Access{}
	held := map[int]bool{}
	for _, h := range listing.AllMachines.Macs {
		for _, s := range h.Shared {
			if s.VM != vm {
				continue
			}
			port := machinefile.PortHeld{Port: s.GuestPort, Seen: s.To}
			if rights.Allows(s.GuestPort) {
				port.Asked = rights.Audience()
			}
			held[s.GuestPort] = true
			access.Ports = append(access.Ports, port)
		}
	}
	for _, h := range listing.AllMachines.Macs {
		for _, l := range h.Ports {
			if l.Machine.Name != vm || held[l.Port] || !rights.Allows(l.Port) {
				continue
			}
			held[l.Port] = true
			access.Ports = append(access.Ports, machinefile.PortHeld{Port: l.Port, Asked: rights.Audience()})
		}
	}
	copied := map[string]bool{}
	for _, s := range first.Value.Shares {
		if s.VM == vm {
			copied[s.Account] = true
		}
	}
	for _, s := range asked.Shares {
		access.Accounts = append(access.Accounts, machinefile.AccountHeld{Account: s.Account, Seen: copied[s.Account], Asked: true})
		delete(copied, s.Account)
	}
	for _, account := range slices.Sorted(maps.Keys(copied)) {
		access.Accounts = append(access.Accounts, machinefile.AccountHeld{Account: account, Seen: true})
	}
	return access, nil
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
