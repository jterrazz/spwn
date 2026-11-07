package wire

import (
	"context"
	"time"

	"github.com/jterrazz/studio-station/src/activity"
	"github.com/jterrazz/studio-station/src/engine"
	"github.com/jterrazz/studio-station/src/settings/adapter/docker"
	"github.com/jterrazz/studio-station/src/settings/adapter/gitaliases"

	accountscalls "github.com/jterrazz/studio-station/src/accounts/calls"
	activitycalls "github.com/jterrazz/studio-station/src/activity/calls"
	activitycontract "github.com/jterrazz/studio-station/src/activity/contract"
	"github.com/jterrazz/studio-station/src/agents/adapter/claudecode"
	"github.com/jterrazz/studio-station/src/agents/adapter/codex"
	agentscalls "github.com/jterrazz/studio-station/src/agents/calls"
	agentscontract "github.com/jterrazz/studio-station/src/agents/contract"
	appscalls "github.com/jterrazz/studio-station/src/apps/calls"
	awakecalls "github.com/jterrazz/studio-station/src/awake/calls"
	backgroundcalls "github.com/jterrazz/studio-station/src/background/calls"
	"github.com/jterrazz/studio-station/src/core/call"
	coremachine "github.com/jterrazz/studio-station/src/core/machine"
	diskcalls "github.com/jterrazz/studio-station/src/disk/calls"
	enginecalls "github.com/jterrazz/studio-station/src/engine/calls"
	imagescalls "github.com/jterrazz/studio-station/src/images/calls"
	machinescalls "github.com/jterrazz/studio-station/src/machines/calls"
	overviewcalls "github.com/jterrazz/studio-station/src/overview/calls"
	portscalls "github.com/jterrazz/studio-station/src/ports/calls"
	settingscalls "github.com/jterrazz/studio-station/src/settings/calls"
	settingscontract "github.com/jterrazz/studio-station/src/settings/contract"
	setupcalls "github.com/jterrazz/studio-station/src/setup/calls"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

// invariant: the one registry of this process, every family's All with its
// Deps; the Router runs its calls on this machine.
func (m machine) registry(env kit.Env) *call.Registry {
	kind := m.profile.Kind
	settings := settingsOf(m)
	var ask settingscalls.Ask
	if env.Interactive {
		ask = settingsAsk{settings: settings, ask: env.Prompt}
	}
	setupOf := setupOf(env, m)
	var named setupcalls.Ask
	if env.Interactive {
		named = setupAsk{setup: setupOf, ask: env.Prompt}
	}
	sections := statusDeps(env, m)
	thisMac := &localCalls{profile: m.profile, late: m.far}
	sections.Calls, sections.Kind = thisMac, kind
	registry, err := call.NewRegistry(
		activitycalls.All(m.activityDeps(sections.Meter)),
		overviewcalls.All(sections),
		agentscalls.All(agentscalls.Deps{Hooks: []agentscontract.Hooks{m.agentHooks(claudecode.ID), m.agentHooks(codex.ID)}, Journal: m.journal(), Session: m.agentSession(), Forge: m.forge(), Now: m.sys.Clock.Now}),
		accountscalls.All(accountscalls.Deps{Accounts: m.accounts(), Shares: m.secrets(), Kind: kind}),
		appscalls.All(appscalls.Deps{Apps: appsOf(env, m), Kind: kind}),
		awakecalls.All(m.awakeDeps()),
		backgroundcalls.All(m.backgroundDeps()),
		diskcalls.All(diskcalls.Deps{Disk: m.diskOf(env), Kind: kind}),
		enginecalls.All(enginecalls.Deps{Control: m.control(), Kind: kind}),
		imagescalls.All(imagescalls.Deps{Images: imageBuilder{m.images()}, Kind: kind}),
		machinescalls.All(machinescalls.Deps{Machines: machineCalls{Machinist: m.machines(env.Interactive), m: m}, Images: createImages{imageBuilder{m.images()}}, Kind: kind, Laptop: m.profile.Caps.Battery, Agents: farAgents{m}}),
		portscalls.All(portscalls.Deps{Ports: m.portsOf(env), Tailnet: m.tailnetStatus(), Kind: kind,
			Declarations: m.stores.declarations, Routes: m.routes(), Far: farPorts{m}, Prefs: m.stores.ports()}),
		setupcalls.All(setupcalls.Deps{Setup: setupRun{setupOf}, Ask: named}),
		settingscalls.All(settingscalls.Deps{
			Settings: settings, Appliers: settings.Appliers(), Applied: m.stores.applied, OffThisMac: settingscontract.OffHere(m.stores.prefs),
			Now: m.sys.Clock.Now, Ask: ask, Shortcuts: docker.New(m.sys.Run), GitAliases: gitaliases.NewShortcuts(m.sys.Run), Shell: m.shell(), Kind: kind,
			Facts: m.facts(), Caps: m.profile.Caps, Admin: m.sys.Env.Euid() == 0, Mac: m.profile.Ref.Name, All: settingsAllMachines{m}, Public: settingsPublic{thisMac},
		}),
	)
	if err != nil {
		panic(err)
	}
	thisMac.registry = registry
	return registry
}

// value: the Router a call reads this machine's other calls through (the
// Overview its sections): this process's own once it is built, so a call
// of the engine reaches the engine; else the registry, past the same gate
// as a frame.
type localCalls struct {
	registry *call.Registry
	profile  coremachine.Profile
	late     *late
}

func (l *localCalls) routed() call.Router {
	if l.late == nil || l.late.router == nil {
		return nil
	}
	return l.late.router
}

func (l *localCalls) Invoke(ctx context.Context, _ coremachine.Ref, name string, in, out any, p call.Progress) error {
	if r := l.routed(); r != nil {
		return r.Invoke(ctx, coremachine.Ref{}, name, in, out, p)
	}
	v, err := l.registry.Run(ctx, l.profile, name, in, p)
	call.Set(out, v)
	return err
}

func (l *localCalls) Watch(ctx context.Context, _ coremachine.Ref, name string, in any, p call.Progress) (<-chan call.Update[any], error) {
	if r := l.routed(); r != nil {
		return r.Watch(ctx, coremachine.Ref{}, name, in, p)
	}
	return l.registry.Stream(ctx, l.profile, name, in, p)
}

type awakeKeeper struct{ *awakecalls.Waker }

// invariant: the history a Mac's engine keeps, asked over its socket.
func engineKept(api engineAPI) func(ctx context.Context, name string) (activity.Kept, time.Time, bool, error) {
	return func(ctx context.Context, name string) (activity.Kept, time.Time, bool, error) {
		return keptOf(ctx, api, name)
	}
}

// invariant: the process that holds a VM's guest, as this Mac's engine names it.
func engineRuns(api engineAPI) func(ctx context.Context) ([]activity.VMRun, error) {
	return func(ctx context.Context) ([]activity.VMRun, error) {
		views, err := api.VMs(ctx)
		if err != nil {
			return nil, err
		}
		runs := make([]activity.VMRun, 0, len(views))
		for _, v := range views {
			if v.Holder != 0 {
				runs = append(runs, activity.VMRun{VM: v.Name, Holder: v.Holder})
			}
		}
		return runs, nil
	}
}

func keptOf(ctx context.Context, api engineAPI, name string) (activity.Kept, time.Time, bool, error) {
	var out engine.ActivityOut
	if err := api.do(ctx, engine.CallActivity, nil, &out); err != nil {
		return activity.Kept{}, time.Time{}, false, err
	}
	for _, kept := range out.Machines {
		if kept.Machine == name {
			return activity.Kept{Last: kept.Last, Series: activity.SeriesOf(kept.Fine, kept.Coarse)}, out.Since, true, nil
		}
	}
	return activity.Kept{}, out.Since, false, nil
}

// value: a sample composes its free space from the Disk family, and on a
// Mac its thermal and battery from Power: a VM's are its Mac's.
func (m machine) activityDeps(meter activitycontract.Meter) activitycalls.Deps {
	d := activitycalls.Deps{Meter: meter, Volume: m.disk(), Name: m.profile.Ref.Name, Kind: m.profile.Kind, Sleep: m.sys.Clock.Sleep, Now: m.sys.Clock.Now}
	if m.profile.Kind == coremachine.Mac {
		d.Kept = engineKept(m.engineAPI())
		d.Runs = engineRuns(m.engineAPI())
	}
	if m.powerPort != nil && (m.profile.Caps.Thermal || m.profile.Caps.Battery) {
		d.Power = m.powerPort
	}
	return d
}
