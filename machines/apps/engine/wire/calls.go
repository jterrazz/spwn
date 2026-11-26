package wire

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/engine/adapter/github"
	"github.com/jterrazz/studio-station/apps/engine/engine/feeds"
	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/ui/token"

	accountscalls "github.com/jterrazz/studio-station/apps/engine/accounts/calls"
	activitycalls "github.com/jterrazz/studio-station/apps/engine/activity/calls"
	activitycontract "github.com/jterrazz/studio-station/apps/engine/activity/contract"
	agentscalls "github.com/jterrazz/studio-station/apps/engine/agents/calls"
	appscalls "github.com/jterrazz/studio-station/apps/engine/apps/calls"
	automationscalls "github.com/jterrazz/studio-station/apps/engine/automations/calls"
	awakecalls "github.com/jterrazz/studio-station/apps/engine/awake/calls"
	backgroundcalls "github.com/jterrazz/studio-station/apps/engine/background/calls"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/guide"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	devicescalls "github.com/jterrazz/studio-station/apps/engine/devices/calls"
	diskcalls "github.com/jterrazz/studio-station/apps/engine/disk/calls"
	enginecalls "github.com/jterrazz/studio-station/apps/engine/engine/calls"
	grantscalls "github.com/jterrazz/studio-station/apps/engine/grants/calls"
	imagescalls "github.com/jterrazz/studio-station/apps/engine/images/calls"
	journalcalls "github.com/jterrazz/studio-station/apps/engine/journal/calls"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	machinefilecalls "github.com/jterrazz/studio-station/apps/engine/machinefile/calls"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	overviewcalls "github.com/jterrazz/studio-station/apps/engine/overview/calls"
	portscalls "github.com/jterrazz/studio-station/apps/engine/ports/calls"
	projectcalls "github.com/jterrazz/studio-station/apps/engine/project/calls"
	settingscalls "github.com/jterrazz/studio-station/apps/engine/settings/calls"
	settingscontract "github.com/jterrazz/studio-station/apps/engine/settings/contract"
	setupcalls "github.com/jterrazz/studio-station/apps/engine/setup/calls"
	skillscalls "github.com/jterrazz/studio-station/apps/engine/skills/calls"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
	teamscalls "github.com/jterrazz/studio-station/apps/engine/teams/calls"
	vaultcalls "github.com/jterrazz/studio-station/apps/engine/vault/calls"
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
	sections := statusDeps(m)
	thisMac := &localCalls{profile: m.profile, late: m.far}
	sections.Calls, sections.Kind = thisMac, kind
	registry, err := call.NewRegistry(
		activitycalls.All(m.activityDeps(sections.Meter)),
		overviewcalls.All(sections),
		agentscalls.All(agentscalls.Deps{Minds: m.minds(), Kind: kind, Home: m.sys.Env.Home()}),
		skillscalls.All(skillscalls.Deps{Files: m.skillFiles(), Kind: kind, Home: m.sys.Env.Home()}),
		projectcalls.All(projectcalls.Deps{Projects: m.projects()}),
		teamscalls.All(teamscalls.Deps{Teams: m.teams(), Home: m.sys.Env.Home()}),
		automationscalls.All(automationscalls.Deps{Automations: m.automations(), Home: m.sys.Env.Home()}),
		accountscalls.All(accountscalls.Deps{SignIns: m.signIns(), Kind: kind}),
		vaultcalls.All(vaultcalls.Deps{Passwords: m.passwords(), SignIns: m.signIns(), Mac: m.where.Machine, Separator: token.Sep, Kind: kind}),
		grantscalls.All(grantscalls.Deps{Grants: m.grants(), Kind: kind}),
		devicescalls.All(m.devicesDeps()),
		appscalls.All(appscalls.Deps{Apps: appsOf(env, m), Known: m.cat.Apps, Origin: machinefile.OriginsOf(m.cat), Kind: kind, All: appsAllMachines{m}, Now: m.sys.Clock.Now}),
		awakecalls.All(m.awakeDeps()),
		backgroundcalls.All(m.backgroundDeps()),
		diskcalls.All(diskcalls.Deps{Disk: m.diskOf(env), Kind: kind, Name: m.profile.Ref.Name}),
		enginecalls.All(m.engineDeps()),
		imagescalls.All(imagescalls.Deps{Images: imageBuilder{m.images()}, Kind: kind}),
		journalcalls.All(m.journalDeps()),
		machinefilecalls.All(m.machinefileDeps(thisMac)),
		machinescalls.All(machinescalls.Deps{Machines: machineCalls{Machinist: m.machines(env.Interactive), m: m}, Images: createImages{imageBuilder{m.images()}}, Kind: kind, OS: m.profile.System.OS, Laptop: m.profile.Caps.Battery, Holdings: holdings{m}, Now: m.sys.Clock.Now}),
		portscalls.All(portscalls.Deps{Ports: m.portsOf(env), Tailnet: m.tailnetStatus(), Kind: kind,
			Declarations: m.stores.declarations, Routes: m.routes(), Far: farPorts{m}, Prefs: m.stores.ports()}),
		setupcalls.All(setupcalls.Deps{Setup: setupInit{setupOf}, Ask: named}),
		settingscalls.All(settingscalls.Deps{
			Settings: settings, Appliers: settings.Appliers(), Applied: m.stores.applied, OffThisMac: settingscontract.OffHere(m.stores.prefs),
			Now: m.sys.Clock.Now, Ask: ask, Shell: m.shell(), Kind: kind,
			Origin: machinefile.OriginsOf(m.cat), Facts: m.facts(), Caps: m.profile.Caps, Admin: m.sys.Env.Euid() == 0, Mac: m.profile.Ref.Name, All: settingsAllMachines{m}, Public: settingsPublic{thisMac},
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
	if m.profile.Kind == coremachine.Computer {
		d.Kept = engineKept(m.engineAPI())
		d.Runs = engineRuns(m.engineAPI())
	}
	if m.powerPort != nil && (m.profile.Caps.Thermal || m.profile.Caps.Battery) {
		d.Power = m.powerPort
	}
	return d
}

// invariant: the engine's calls over this Mac's engine, its job and the
// releases of station: in a world, its fakes; else GitHub, asked through
// the same variables as the apps' releases.
func (m machine) engineDeps() enginecalls.Deps {
	d := enginecalls.Deps{
		Control: m.control(), Installer: m.installer(), Kind: m.profile.Kind,
		Cache: m.stores.release, Version: m.engineVersion(), Arch: m.sys.Env.GOARCH(), Now: m.sys.Clock.Now,
		Asking: &m.stores.asking, Page: page.Revision, Views: enginecalls.Projected(feeds.Pages{ThisMachine: m.where}),
		Forward: func(ctx context.Context, in engine.StateIn) (<-chan json.RawMessage, error) {
			frames, err := m.engineAPI().c.Watch(ctx, enginecalls.StateWatch.Name, in)
			return frames, downAs(err, &enginecalls.DownError{})
		},
		Guided: func(ctx context.Context) (guide.Guides, error) {
			var out guide.Guides
			err := m.engineAPI().do(ctx, enginecalls.Guides.Name, struct{}{}, &out)
			return out, err
		},
	}
	if m.world != nil {
		d.Releases = m.world.releases()
		return d
	}
	d.Releases = github.New(github.Options{
		Net: m.sys.Net, Repo: catalog.StationRepo,
		APIURL: m.or("STATION_GITHUB_API_URL", "https://api.github.com"),
		Token:  m.or("GITHUB_TOKEN", m.vars["GH_TOKEN"]),
	})
	return d
}
