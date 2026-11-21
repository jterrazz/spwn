package readers

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	grantscalls "github.com/jterrazz/studio-station/apps/engine/grants/calls"
	vaultcalls "github.com/jterrazz/studio-station/apps/engine/vault/calls"
	"sync"
	"time"

	accountscalls "github.com/jterrazz/studio-station/apps/engine/accounts/calls"
	activitycalls "github.com/jterrazz/studio-station/apps/engine/activity/calls"
	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/engine/store"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	"github.com/jterrazz/studio-station/apps/engine/overview"
	overviewcalls "github.com/jterrazz/studio-station/apps/engine/overview/calls"
	portscalls "github.com/jterrazz/studio-station/apps/engine/ports/calls"
)

const (
	NameMac          = "mac"
	NamePorts        = "ports"
	NameOverview     = "overview"
	NameOverviewVMs  = "overview-vms"
	NameActivity     = "activity"
	NameSignins      = "signins"
	NameAwake        = "awake"
	NameRecords      = "records"
	NameVault        = "vault"
	activityGraph    = 6 * time.Minute
	macIdle          = time.Minute
	macAttached      = 30 * time.Second
	overviewEvery    = 15 * time.Minute
	vmOverviewEvery  = 5 * time.Minute
	activityIdle     = 5 * time.Minute
	activityAttached = 30 * time.Second
	signinsIdle      = 15 * time.Minute
	signinsAttached  = 5 * time.Minute
	vaultIdle        = 15 * time.Minute
	vaultAttached    = 5 * time.Minute
)

type Local struct {
	// value: the Router the readings run through, this engine's own: a VM's
	// through its guest.
	Calls   call.Router
	ThisMac string
	// value: This Mac's Awake as the engine holds it; nil reads none.
	Awake func() awake.Report
	// value: every other Mac as This Mac's records keep it and the
	// tailnet's node of each Mac; nil reads neither.
	Known func(ctx context.Context) (map[string]machines.MacRecord, map[string]machines.Node)
}

func (l Local) Readers() []*Reader {
	// invariant: ADR-070's cadences: This Mac and its ports every 60 s idle
	// and 30 s attached; the overview every 15 min, a VM's every 5 min
	// attached; activity every 5 min idle and 30 s attached; sign-ins every
	// 15 min idle and 5 min attached; Awake at every tick, from memory; the
	// records and the tailnet with This Mac, the vault with the sign-ins.
	out := []*Reader{
		{Name: NameMac, Idle: macIdle, Attached: macAttached, Budget: 5 * time.Second, Read: l.mac},
		{Name: NamePorts, Idle: macIdle, Attached: macAttached, Budget: 5 * time.Second, Read: l.ports},
		{Name: NameOverview, Idle: overviewEvery, Attached: overviewEvery, Budget: time.Minute, Read: l.overview(false)},
		{Name: NameOverviewVMs, Idle: overviewEvery, Attached: vmOverviewEvery, Budget: time.Minute, Read: l.overview(true)},
		{Name: NameActivity, Idle: activityIdle, Attached: activityAttached, Budget: 2 * time.Second, Read: l.activity},
		{Name: NameSignins, Idle: signinsIdle, Attached: signinsAttached, Budget: 10 * time.Second, Read: l.signins},
		{Name: NameVault, Idle: vaultIdle, Attached: vaultAttached, Budget: 5 * time.Second, Read: l.vault},
	}
	if l.Known != nil {
		out = append(out, &Reader{Name: NameRecords, Idle: macIdle, Attached: macAttached, Budget: 5 * time.Second, Read: l.known})
	}
	if l.Awake != nil {
		out = append(out, &Reader{Name: NameAwake, Read: l.awake})
	}
	return out
}

func (l Local) mac(ctx context.Context, _ *store.Snapshot) Reading {
	k := store.Of(store.FamilyMac, l.ThisMac)
	out, err := call.Read(ctx, l.Calls, machine.Ref{}, machinescalls.List, machinescalls.ListIn{ThisMac: true, Unwritten: true}, nil)
	if err != nil || len(out.All.Macs) == 0 {
		return Reading{Owns: []store.Key{k}, Failed: map[store.Key]string{k: Why(err)}}
	}
	return Reading{Owns: []store.Key{k}, Puts: []store.Put{{Key: k, Value: out.All.Macs[0], ReadAt: out.All.Read}}}
}

func (l Local) ports(ctx context.Context, _ *store.Snapshot) Reading {
	k := store.Of(store.FamilyPorts, l.ThisMac)
	out, err := call.Invoke(ctx, l.Calls, machine.Ref{}, portscalls.List, portscalls.ListIn{}, nil)
	if err != nil || len(out.AllMachines.Macs) == 0 {
		return Reading{Owns: []store.Key{k}, Failed: map[store.Key]string{k: Why(err)}}
	}
	return Reading{Owns: []store.Key{k}, Puts: []store.Put{{Key: k, Value: out.AllMachines.Macs[0]}}}
}

// value: the VMs of This Mac that run, as its slice last said them.
func running(now *store.Snapshot, thisMac string) []string {
	sl, ok := now.Slices[store.Of(store.FamilyMac, thisMac)]
	if !ok {
		return nil
	}
	view, ok := sl.Value.(machines.MacView)
	if !ok {
		return nil
	}
	var out []string
	for _, vm := range view.VMs {
		if vm.State == machines.Running {
			out = append(out, vm.Name)
		}
	}
	return out
}

// value: This Mac then each VM of it that runs, by the name its keys carry.
func (l Local) machines(now *store.Snapshot) []string {
	return append([]string{l.ThisMac}, running(now, l.ThisMac)...)
}

func ref(thisMac, m string) machine.Ref {
	if m == thisMac {
		return machine.Ref{}
	}
	return machine.Ref{Name: m}
}

// value: one reading per machine, all at once, each its own put or failure.
func each(ctx context.Context, family store.Family, names []string, read func(ctx context.Context, m string) (any, time.Time, error)) Reading {
	var mu sync.Mutex
	var wg sync.WaitGroup
	got := Reading{Failed: map[store.Key]string{}}
	for _, m := range names {
		k := store.Of(family, m)
		got.Owns = append(got.Owns, k)
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, at, err := read(ctx, m)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				got.Failed[k] = Why(err)
				return
			}
			got.Puts = append(got.Puts, store.Put{Key: k, Value: v, ReadAt: at})
		}()
	}
	wg.Wait()
	return got
}

func (l Local) overview(vms bool) func(ctx context.Context, now *store.Snapshot) Reading {
	return func(ctx context.Context, now *store.Snapshot) Reading {
		names := []string{l.ThisMac}
		if vms {
			names = running(now, l.ThisMac)
		}
		return each(ctx, store.FamilyOverview, names, func(ctx context.Context, m string) (any, time.Time, error) {
			out, err := call.Read(ctx, l.Calls, ref(l.ThisMac, m), overviewcalls.Read, overviewcalls.ReadIn{Fresh: true}, nil)
			// invariant: the store keeps facts, never a value of the clock:
			// the reading's instant is the slice's, the uptime a boot
			// instant, so two readings of an unchanged machine hash equal.
			return overview.KeptOf(out.Summary), out.Summary.Read, err
		})
	}
}

func (l Local) activity(ctx context.Context, now *store.Snapshot) Reading {
	return each(ctx, store.FamilyActivity, l.machines(now), func(ctx context.Context, m string) (any, time.Time, error) {
		out, err := call.Read(ctx, l.Calls, ref(l.ThisMac, m), activitycalls.Read, activitycalls.ReadIn{History: true, Graph: activityGraph, Quick: true}, nil)
		return out, time.Time{}, err
	})
}

func (l Local) signins(ctx context.Context, now *store.Snapshot) Reading {
	return each(ctx, store.FamilySignins, l.machines(now), func(ctx context.Context, m string) (any, time.Time, error) {
		in := accountscalls.ProgramsIn{}
		if m != l.ThisMac {
			in.Machine = m
		}
		out, err := call.Read(ctx, l.Calls, machine.Ref{}, accountscalls.Programs, in, nil)
		return out, time.Time{}, err
	})
}

func (l Local) awake(context.Context, *store.Snapshot) Reading {
	k := store.Of(store.FamilyAwake, l.ThisMac)
	r := l.Awake()
	r.Now = time.Time{}
	return Reading{Owns: []store.Key{k}, Puts: []store.Put{{Key: k, Value: r}}}
}

func (l Local) known(ctx context.Context, _ *store.Snapshot) Reading {
	records, nodes := l.Known(ctx)
	if records == nil {
		records = map[string]machines.MacRecord{}
	}
	if nodes == nil {
		nodes = map[string]machines.Node{}
	}
	r, t := store.Of(store.FamilyRecords, ""), store.Of(store.FamilyTailnet, "")
	return Reading{Owns: []store.Key{r, t}, Puts: []store.Put{{Key: r, Value: records}, {Key: t, Value: nodes}}}
}

func (l Local) vault(ctx context.Context, _ *store.Snapshot) Reading {
	k := store.Of(store.FamilyVault, "")
	read, err := call.Read(ctx, l.Calls, machine.Ref{}, vaultcalls.VaultList, vaultcalls.VaultListIn{}, nil)
	if err != nil {
		return Reading{Owns: []store.Key{k}, Failed: map[store.Key]string{k: Why(err)}}
	}
	out := accounts.VaultOffers{Vault: read, Offers: map[string]accounts.GrantOffer{}}
	for _, e := range read.Entries {
		if offer, err := call.Read(ctx, l.Calls, machine.Ref{}, grantscalls.GrantNew, grantscalls.GrantNewIn{Entry: e.ID}, nil); err == nil {
			out.Offers[e.ID] = offer
		}
	}
	return Reading{Owns: []store.Key{k}, Puts: []store.Put{{Key: k, Value: out}}}
}
