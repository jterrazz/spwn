package feeds

import (
	"context"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/accounts"
	accountscalls "github.com/jterrazz/studio-station/src/accounts/calls"
	activitycalls "github.com/jterrazz/studio-station/src/activity/calls"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/engine"
	"github.com/jterrazz/studio-station/src/machines"
	machinescalls "github.com/jterrazz/studio-station/src/machines/calls"
	"github.com/jterrazz/studio-station/src/page"
	pageactivity "github.com/jterrazz/studio-station/src/pages/activity"
	pageawake "github.com/jterrazz/studio-station/src/pages/awake"
	pagemenubar "github.com/jterrazz/studio-station/src/pages/menubar"
	portscalls "github.com/jterrazz/studio-station/src/ports/calls"
)

func (o Menubar) Ports() engine.Feed {
	// invariant: SPEC §10.2 Ports: the ports of the machine the page is of,
	// a Mac or a VM, read through the Mac it is on, so a server down still
	// lists its ports unreachable; of every Mac when it names none; read when
	// it opens, every 30 s and on a change of this engine's state.
	return func(ctx context.Context, in engine.FeedInput) <-chan any {
		read := func(ctx context.Context) page.Page {
			of, list := pagemenubar.PortsOf{Machine: in.Subject}, portscalls.ListIn{AllMachines: true, Of: in.Subject}
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
	// invariant: SPEC B5.8: the accounts of every answering Mac and the
	// copies each shared, each read through its own engine; of one Mac, or
	// of the Mac a VM is on, when the page names one; read when it opens,
	// every 30 s, on a refresh and on a change of this engine's state.
	return func(ctx context.Context, in engine.FeedInput) <-chan any {
		read := func(ctx context.Context) page.Page {
			listed, err := call.Read(ctx, o.Calls, machine.Ref{}, machinescalls.List, machinescalls.ListIn{Unwritten: true}, nil)
			if err != nil {
				return pagemenubar.Unread(pagemenubar.AccountsID, o.ThisMachine, err)
			}
			of := accountsOf(listed.All, in.Subject)
			all := accounts.AllMachines{Now: o.Clock.Now()}
			for _, h := range listed.All.Macs {
				if of.Mac != "" && h.Name != of.Mac {
					continue
				}
				m := accounts.MacAccounts{Name: h.Name, ThisMac: h.ThisMac}
				if !h.ThisMac && !h.Reachable {
					m.Silent = page.MacName(h.Name, o.ThisMachine.Machine) + " does not answer"
					all.Macs = append(all.Macs, m)
					continue
				}
				ref := machine.Ref{Name: h.Name}
				if h.ThisMac {
					ref = machine.Ref{}
				}
				r, err := call.Read(ctx, o.Calls, ref, accountscalls.List, accountscalls.ListIn{}, nil)
				if err != nil {
					m.Silent = err.Error()
				} else if len(r.Machines) > 0 {
					m.Accounts, m.Shares = r.Machines[0].Accounts, r.Shares
				}
				all.Macs = append(all.Macs, m)
			}
			return pagemenubar.Accounts(all, of, o.ThisMachine)
		}
		return o.reread(ctx, in, pagemenubar.AccountsID, remoteEvery, read)
	}
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
	// invariant: SPEC §10.2 Activity, §6.3: This Mac's measures over the
	// window the page names after its colon (1 h unless named), read from the
	// activity watch every 2 s while the page is open.
	return func(ctx context.Context, in engine.FeedInput) <-chan any {
		window, err := time.ParseDuration(in.Subject)
		if err != nil || window <= 0 {
			window = time.Hour
		}
		read := func(ctx context.Context) page.Page {
			v := pageactivity.View{Name: o.ThisMachine.Machine, Label: "This Mac", Kind: string(machine.Mac), ThisMac: true, Graph: window}
			snaps, err := call.Follow(ctx, o.Calls, machine.Ref{}, activitycalls.Read, activitycalls.ReadIn{Graph: window}, nil)
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
