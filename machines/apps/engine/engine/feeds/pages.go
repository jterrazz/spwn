package feeds

import (
	"context"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	accountscalls "github.com/jterrazz/studio-station/apps/engine/accounts/calls"
	activitycalls "github.com/jterrazz/studio-station/apps/engine/activity/calls"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	grantscalls "github.com/jterrazz/studio-station/apps/engine/grants/calls"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	overviewcalls "github.com/jterrazz/studio-station/apps/engine/overview/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageactivity "github.com/jterrazz/studio-station/apps/engine/pages/activity"
	pageawake "github.com/jterrazz/studio-station/apps/engine/pages/awake"
	pagemenubar "github.com/jterrazz/studio-station/apps/engine/pages/menubar"
	portscalls "github.com/jterrazz/studio-station/apps/engine/ports/calls"
	vaultcalls "github.com/jterrazz/studio-station/apps/engine/vault/calls"
)

func (o Menubar) Ports() engine.Feed {
	// invariant: SPEC §10.2 Ports: the ports of the machine the page is of,
	// a Mac or a VM, read through the Mac it is on, so a server down still
	// lists its ports unreachable; of every Mac when it names none; read when
	// it opens, every 30 s and on a change of this engine's state.
	return func(ctx context.Context, in engine.FeedInput) <-chan any {
		read := func(ctx context.Context) page.Page {
			of, list := pagemenubar.PortsOf{Machine: in.Subject, Now: o.Clock.Now()}, portscalls.ListIn{AllMachines: true, Of: in.Subject}
			if listed, err := call.Read(ctx, o.Calls, machine.Ref{}, machinescalls.List, machinescalls.ListIn{Unwritten: true}, nil); err == nil {
				of.Machines = &listed.All
			}
			l, err := call.Invoke(ctx, o.Calls, machine.Ref{}, portscalls.List, list, nil)
			if err != nil {
				return pagemenubar.Unread(pagemenubar.PortsID, o.ThisMachine, err)
			}
			return pagemenubar.Ports(l, of, o.ThisMachine)
		}
		return o.reread(ctx, in, pagemenubar.PortsID, remoteEvery, read)
	}
}

func (o Menubar) Accounts() engine.Feed {
	// invariant: SPEC B5.8: the sign-ins and grants of every answering Mac
	// and its VMs, each read through its own engine; of one Mac, or
	// of the Mac a VM is on, when the page names one; read when it opens,
	// every 30 s, on a refresh and on a change of this engine's state.
	return func(ctx context.Context, in engine.FeedInput) <-chan any {
		read := func(ctx context.Context) page.Page {
			listed, err := call.Read(ctx, o.Calls, machine.Ref{}, machinescalls.List, machinescalls.ListIn{Unwritten: true}, nil)
			if err != nil {
				return pagemenubar.Unread(pagemenubar.AccountsID, o.ThisMachine, err)
			}
			of := accountsOf(listed.All, in.Subject)
			of.Asks = levels
			return pagemenubar.Accounts(o.signIns(ctx, listed.All, of.Mac), of, o.ThisMachine)
		}
		return o.reread(ctx, in, pagemenubar.AccountsID, remoteEvery, read)
	}
}

// value: the sign-ins and grants of every Mac and its running VMs, or of
// one Mac's, each read through the Mac they are on; a Mac that does not
// answer is named, never dropped. This Mac's vault and what grants.new
// says of each password are read alongside.
func (o Menubar) signIns(ctx context.Context, listed machines.AllMachines, only string) pagemenubar.SignIns {
	var out pagemenubar.SignIns
	for _, h := range listed.Macs {
		if h.ThisMac {
			out.Vault, out.Offers = o.vault(ctx)
		}
		if only != "" && h.Name != only {
			continue
		}
		m := pagemenubar.MachineSignIns{Name: h.Name, Mac: h.Name, ThisMac: h.ThisMac}
		if !h.ThisMac && !h.Reachable {
			m.Silent = page.MacName(h.Name, o.ThisMachine.Machine) + " does not answer"
			out.Machines = append(out.Machines, m)
			continue
		}
		out.Machines = append(out.Machines, o.programsOf(ctx, h, m, ""))
		for _, vm := range h.VMs {
			if vm.State != machines.Running {
				continue
			}
			out.Machines = append(out.Machines, o.programsOf(ctx, h, pagemenubar.MachineSignIns{Name: vm.Name, Mac: h.Name, ThisMac: h.ThisMac, VM: true}, vm.Name))
		}
	}
	return out
}

func (o Menubar) programsOf(ctx context.Context, h machines.MacView, m pagemenubar.MachineSignIns, vm string) pagemenubar.MachineSignIns {
	read, err := call.Read(ctx, o.Calls, macRef(h), accountscalls.Programs, accountscalls.ProgramsIn{Machine: vm}, nil)
	if err != nil {
		m.Silent = err.Error()
		return m
	}
	m.Rows = read.Programs
	return m
}

func (o Menubar) vault(ctx context.Context) (vault *accounts.VaultRead, offers map[string]accounts.GrantOffer) {
	read, err := call.Read(ctx, o.Calls, machine.Ref{}, vaultcalls.VaultList, vaultcalls.VaultListIn{}, nil)
	if err != nil {
		return nil, nil
	}
	offers = map[string]accounts.GrantOffer{}
	for _, e := range read.Entries {
		if offer, err := call.Read(ctx, o.Calls, machine.Ref{}, grantscalls.GrantNew, grantscalls.GrantNewIn{Entry: e.ID}, nil); err == nil {
			offers[e.ID] = offer
		}
	}
	return &read, offers
}

func macRef(h machines.MacView) machine.Ref {
	if h.ThisMac {
		return machine.Ref{}
	}
	return machine.Ref{Name: h.Name}
}

func (o Menubar) Needs() engine.Feed {
	// invariant: SPEC B5.7: menubar.needs[:<machine>] reads what the needs
	// are made of: the machines, every Mac's ports, accounts and Overview, each
	// Mac's through its engine; read when it opens, every 30 s, on a refresh
	// and on a change of this engine's state.
	return func(ctx context.Context, in engine.FeedInput) <-chan any {
		read := func(ctx context.Context) page.Page {
			listed, err := call.Read(ctx, o.Calls, machine.Ref{}, machinescalls.List, machinescalls.ListIn{Unwritten: true}, nil)
			if err != nil {
				return pagemenubar.Unread(pagemenubar.NeedsID, o.ThisMachine, err)
			}
			r := pagemenubar.NeedsReading{Now: o.Clock.Now(), Zone: o.zone(), All: &listed.All, Asks: levels}
			if l, err := call.Invoke(ctx, o.Calls, machine.Ref{}, portscalls.List, portscalls.ListIn{AllMachines: true}, nil); err == nil {
				r.Ports = &l.AllMachines
			}
			for _, h := range listed.All.Macs {
				if !h.ThisMac && !h.Reachable {
					continue
				}
				if s, err := call.Read(ctx, o.Calls, macRef(h), overviewcalls.Read, overviewcalls.ReadIn{}, nil); err == nil {
					r.Overviews = append(r.Overviews, s.Summary)
				}
			}
			return pagemenubar.NeedsPage(r, needsOf(listed.All, in.Subject), o.ThisMachine)
		}
		return o.reread(ctx, in, pagemenubar.NeedsID, remoteEvery, read)
	}
}

// value: the machine a needs page is of, by its full name: a Mac named as
// the interface names it, or a VM; empty is every machine.
func needsOf(all machines.AllMachines, subject string) string {
	if subject == "" {
		return ""
	}
	for _, h := range all.Macs {
		if h.Name == subject || strings.HasSuffix(h.Name, "-"+subject) {
			return h.Name
		}
	}
	return subject
}

// value: what an accounts page is of: a Mac named in full or as the
// interface names it, or a VM by the Mac it is on; empty is every Mac.
func accountsOf(all machines.AllMachines, subject string) pagemenubar.AccountsOf {
	if subject == "" {
		return pagemenubar.AccountsOf{}
	}
	for _, h := range all.Macs {
		if h.Name == subject || strings.HasSuffix(h.Name, "-"+subject) {
			return pagemenubar.AccountsOf{Mac: h.Name}
		}
	}
	for _, h := range all.Macs {
		for _, vm := range h.VMs {
			if vm.Name == subject {
				return pagemenubar.AccountsOf{Mac: h.Name, VM: vm.Name}
			}
		}
	}
	return pagemenubar.AccountsOf{Mac: subject}
}

func (o Menubar) Awake() engine.Feed {
	// invariant: SPEC §10.2 Awake: This Mac's Awake then each other Mac's,
	// each through its engine, read when it opens, every 30 s and on a change
	// of this engine's state (a keep-awake taken or let go).
	return func(ctx context.Context, in engine.FeedInput) <-chan any {
		read := func(ctx context.Context) page.Page {
			f, err := o.EveryMac(ctx)
			if err != nil {
				return pagemenubar.Unread(pagemenubar.AwakeID, o.ThisMachine, err)
			}
			return pagemenubar.Awake(f, o.ThisMachine)
		}
		return o.reread(ctx, in, pagemenubar.AwakeID, remoteEvery, read)
	}
}

func (o Menubar) Activity() engine.Feed {
	// invariant: SPEC §10.2 Activity, §6.3, B5.16: a machine's measures over
	// the window the page names after its colon (1 h unless named), This
	// Mac's unless a machine is named before the window
	// (`menubar.activity:infra:6m`), read every 2 s while the page is open.
	return func(ctx context.Context, in engine.FeedInput) <-chan any {
		name, window := activitySubject(in.Subject)
		read := func(ctx context.Context) page.Page {
			at, v, ask := o.activityOf(ctx, name)
			v.Graph, ask.Graph = window, window
			snaps, err := call.Follow(ctx, o.Calls, at, activitycalls.Read, ask, nil)
			if err != nil {
				return pagemenubar.Unread(pagemenubar.ActivityID, o.ThisMachine, err)
			}
			for u := range snaps {
				if u.Err != nil {
					return pagemenubar.Unread(pagemenubar.ActivityID, o.ThisMachine, u.Err)
				}
				v.Report = u.Value
			}
			return pagemenubar.Activity(v, o.ThisMachine)
		}
		return o.reread(ctx, in, pagemenubar.ActivityID, sampleEvery, read)
	}
}

// value: the machine and the window an Activity page names: a window alone
// is This Mac's, a machine alone is its last hour.
func activitySubject(subject string) (name string, window time.Duration) {
	name, words, named := strings.Cut(subject, ":")
	if !named {
		words = subject
	}
	if d, err := time.ParseDuration(words); err == nil {
		window = d
		if !named {
			name = ""
		}
	}
	if window <= 0 {
		window = time.Hour
	}
	return name, window
}

// value: SPEC B5.16: where a machine's activity is read, through its
// engine: a Mac on itself, a VM on the Mac that runs it, whose engine keeps
// its guest's history.
func (o Menubar) activityOf(ctx context.Context, name string) (machine.Ref, pageactivity.View, activitycalls.ReadIn) {
	if name == "" || name == o.ThisMachine.Machine {
		return machine.Ref{}, pageactivity.View{Name: o.ThisMachine.Machine, Label: "This Mac", Kind: string(machine.Computer), ThisMac: true}, activitycalls.ReadIn{}
	}
	mac, isVM := "", false
	if o.Live != nil {
		mac, isVM = o.Live.macOf(ctx, name)
	}
	if !isVM {
		return machine.Ref{Name: name}, pageactivity.View{Name: name, Label: name, Kind: string(machine.Computer)}, activitycalls.ReadIn{}
	}
	at := machine.Ref{Name: mac}
	if mac == o.ThisMachine.Machine {
		at = machine.Ref{}
	}
	v := pageactivity.View{Name: name, Label: name, Mac: mac, On: mac}
	return at, v, activitycalls.ReadIn{Of: name, Machine: machine.Ref{Name: name, On: mac}}
}

// invariant: a page of the menu bar's own goes out at once as its frame,
// loading, then once per read: on its period, on a refresh and on a change
// of the engine's state; nothing of it runs once the watch ends.
func (o Menubar) reread(ctx context.Context, in engine.FeedInput, id page.ID, every time.Duration, read func(context.Context) page.Page) <-chan any {
	out := make(chan any, 1)
	go func() {
		defer close(out)
		out <- page.Document{Schema: page.Schema, Page: pagemenubar.Opening(id, o.ThisMachine)}
		for {
			p := read(ctx)
			if ctx.Err() != nil {
				return
			}
			select {
			case <-out:
			default:
			}
			select {
			case out <- page.Document{Schema: page.Schema, Page: p}:
			case <-ctx.Done():
				return
			}
			wait, cancel := context.WithCancel(ctx)
			go func() {
				defer cancel()
				select {
				case <-in.Changed:
				case <-in.Refresh:
				case <-wait.Done():
				}
			}()
			_ = o.Clock.Sleep(wait, every)
			cancel()
			if ctx.Err() != nil {
				return
			}
		}
	}()
	return out
}

type EveryMac func(ctx context.Context) (pageawake.AllMachines, error)
