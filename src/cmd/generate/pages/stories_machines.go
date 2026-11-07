package pages

import (
	"time"

	"github.com/jterrazz/studio-station/src/accounts"
	"github.com/jterrazz/studio-station/src/activity"
	"github.com/jterrazz/studio-station/src/awake"
	"github.com/jterrazz/studio-station/src/catalog"
	coremachine "github.com/jterrazz/studio-station/src/core/machine"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/overview"
	"github.com/jterrazz/studio-station/src/page"
	pageactivity "github.com/jterrazz/studio-station/src/pages/activity"
	pageawake "github.com/jterrazz/studio-station/src/pages/awake"
	pagemenubar "github.com/jterrazz/studio-station/src/pages/menubar"
	"github.com/jterrazz/studio-station/src/ports"
)

const (
	proName    = "macbook-pro"
	miniMac    = "mac-mini"
	airName    = "macbook-air"
	tailnet    = ".tail77a797.ts.net"
	stationVer = "v4.0.0"
)

type Scene struct {
	All      model.AllMachines
	Ports    ports.AllMachines
	Home     overview.Summary
	Away     []overview.Summary
	Insides  map[string]overview.Summary
	Samples  map[string][]activity.Reading
	Accounts accounts.Report
	Awake    awake.Report
	Macs     []awake.Report
	Activity pageactivity.View
}

func everyMachine() Scene {
	return Scene{
		All:      sceneMachines(),
		Ports:    scenePorts(),
		Home:     proSummary(),
		Away:     []overview.Summary{miniOverview()},
		Insides:  map[string]overview.Summary{"linux-1": linux1Inside(), "infra": infraInside()},
		Samples:  sceneSamples(),
		Accounts: sceneAccounts(),
		Awake:    proAwake(),
		Macs:     []awake.Report{miniAwakeReport(), airAwake()},
		Activity: proActivity(),
	}
}

func sceneVM(name, mac string, life model.Lifetime, state model.VMState, cpus int, memoryGB int64, ip string) model.VM {
	vm := sized(model.VM{Name: name, Image: "linux", Version: "3", Mac: mac, Kind: catalog.Linux, Lifetime: life, State: state, SizeGB: 40, IP: ip, CreatedAt: storyNow.Add(-5 * 24 * time.Hour)}, cpus, memoryGB)
	if state == model.Running {
		vm.Guest, vm.SSH = "connected", true
		vm.BootAt = storyNow.Add(-2 * time.Hour)
	}
	return vm
}

func sceneMachines() model.AllMachines {
	engine := model.EngineView{Known: true, Running: true, Version: stationVer}
	infra := sceneVM("infra", miniMac, model.AlwaysOn, model.Running, 8, 24, "192.168.64.7")
	infra.SizeGB, infra.Resources.DiskBytes = 80, 80*gb
	infra.Want, infra.Check, infra.Device = "running", "https://infra:6443/readyz", "infra"
	infra.BootAt = storyNow.Add(-12 * 24 * time.Hour)
	infra.CheckStatus = model.CheckStatus{State: model.CheckPassing, At: storyNow.Add(-40 * time.Second), Answer: "200"}
	return model.AllMachines{Read: storyNow, Macs: []model.MacView{
		{Name: proName, User: "jterrazz", StationVersion: stationVer, ThisMac: true, Reachable: true, Answered: storyNow, Engine: engine, Node: model.Node{Online: true, IP: "100.98.133.14"}, VMs: []model.VM{
			sceneVM("linux-1", proName, model.Throwaway, model.Running, 4, 8, "192.168.64.5"),
			sceneVM("agent-7f", proName, model.Throwaway, model.Running, 4, 8, "192.168.64.6"),
			sceneVM("api", proName, model.Permanent, model.Stopped, 4, 8, ""),
		}},
		{Name: miniMac, User: "jterrazz", StationVersion: stationVer, Reachable: true, Answered: storyNow, Engine: engine, Node: model.Node{Online: true, IP: "100.98.133.20"}, LAN: "192.168.1.20", VMs: []model.VM{
			infra,
			sceneVM("linux-3", miniMac, model.Throwaway, model.Running, 4, 8, "192.168.64.8"),
			sceneVM("review-bot", miniMac, model.Permanent, model.Running, 2, 4, "192.168.64.9"),
		}},
		{Name: airName, User: "jterrazz", StationVersion: stationVer, Answered: storyNow.Add(-2 * time.Hour), Node: model.Node{IP: "100.98.133.31"}},
	}}
}

func scenePorts() ports.AllMachines {
	pro, mini := proName+tailnet, miniMac+tailnet
	linux1, agent := coremachine.Ref{Name: "linux-1", On: proName}, coremachine.Ref{Name: "agent-7f", On: proName}
	infra, linux3, bot := coremachine.Ref{Name: "infra", On: miniMac}, coremachine.Ref{Name: "linux-3", On: miniMac}, coremachine.Ref{Name: "review-bot", On: miniMac}
	listen := func(m coremachine.Ref, port int, process string) ports.Port {
		return ports.Port{Machine: m, Port: port, Process: process, Binds: []string{"0.0.0.0"}, Reach: ports.Tailnet}
	}
	return ports.AllMachines{Schema: ports.AllMachinesSchema, Macs: []ports.MacPorts{
		{Name: proName, ThisMac: true, Reachable: true, Engine: ports.EngineInfo{Running: true}, PublicName: pro, Serve: "on", HTTPS: "port",
			Shared: []ports.SharedPort{
				{VM: "linux-1", GuestPort: 5173, Reserved: 40001, State: ports.PhaseShared, Process: "node", HTTP: true, Profile: "vite", URLs: ports.URLSet{Public: "https://" + pro + ":40001"}},
				{VM: "linux-1", GuestPort: 3000, Reserved: 40002, State: ports.PhaseShared, Process: "node", HTTP: true, URLs: ports.URLSet{Public: "https://" + pro + ":40002"}},
			},
			Ports: []ports.Port{listen(linux1, 4321, "astro"), listen(linux1, 9229, "node"), listen(agent, 6006, "storybook")}},
		{Name: miniMac, Reachable: true, Engine: ports.EngineInfo{Running: true}, PublicName: mini, Serve: "on", HTTPS: "port",
			Shared: []ports.SharedPort{
				{VM: "linux-3", GuestPort: 8080, Reserved: 40001, State: ports.PhaseShared, Process: "python3", HTTP: true, URLs: ports.URLSet{Public: "https://" + mini + ":40001"}},
			},
			Ports: []ports.Port{
				{Machine: infra, Port: 443, Process: "nginx", Binds: []string{"0.0.0.0"}, Reach: ports.Outside,
					Declared: &ports.Declaration{Machine: infra.Name, Port: 443, Via: catalog.RouteCloudflareTunnel}, Verified: &ports.Verified{Seen: true}},
				{Machine: infra, Port: 6443, Process: "kube-apiserver", Binds: []string{"0.0.0.0"}, Reach: ports.Tailnet, Device: true},
				listen(linux3, 5432, "postgres"), listen(bot, 8000, "uvicorn"), listen(coremachine.Ref{Name: miniMac}, 5900, "screensharingd"),
			}},
		{Name: airName},
	}}
}

func proSummary() overview.Summary {
	return overview.Summary{
		Head:   overview.Head{Name: proName, Kind: coremachine.Mac, ThisMac: true, Up: 3 * 24 * time.Hour},
		Engine: overview.EngineRunning,
		Needs: []overview.Need{
			{Section: "accounts", Subject: "npm", Broken: true, Text: "npm signed out", Command: "npm login"},
			{Section: "background", Subject: "com.jterrazz.brain.x-snapshot", Broken: true, Text: "1 login item failing", Command: "station background"},
			{Section: "apps", Subject: "updates", Text: "2 app updates", Call: "apps.update"},
			{Section: "settings", Subject: "firewall", Text: "Firewall off", Path: "System Settings > Network > Firewall"},
		},
		Lines: []overview.Line{
			{Section: "activity", Parts: []overview.Part{{Word: "CPU 34%"}, {Word: "memory 31 of 48 GB"}, {Bytes: 388_000_000_000, Word: "free"}}},
			{Section: "apps", Parts: []overview.Part{overview.Count(catalog.StateOK, 39, "ok", "ok"), overview.Count(catalog.StateDegraded, 2, "updates", "update-available")}},
			{Section: "settings", Parts: []overview.Part{{Word: "set by station"}, {State: catalog.StateDegraded, N: 11, Of: 12, Key: "applied", Counted: true}}},
			{Section: "disk", Parts: []overview.Part{{Bytes: 388_000_000_000, Word: "free"}}},
			{Section: "ports", Parts: []overview.Part{{Word: "Private", N: 2, Key: "private", Counted: true}}},
			{Section: "accounts", Parts: []overview.Part{overview.Count(catalog.StateOK, 6, "", "signed-in"), {State: catalog.StateBroken, Word: "npm"}}},
			{Section: "background", Parts: []overview.Part{overview.Count(catalog.StateOK, 8, "", "ok"), overview.Count(catalog.StateBroken, 1, "failing", "failing")}},
			{Section: "awake", Parts: []overview.Part{{State: catalog.StateOK, Word: "until 18:00"}}},
		},
		VMs: []overview.VM{{Name: "linux-1", Running: true}, {Name: "agent-7f", Running: true}, {Name: "api"}},
	}
}

func miniOverview() overview.Summary {
	ready := func(name, value string) overview.Check {
		return overview.Check{Name: name, State: catalog.StateOK, Value: value}
	}
	return overview.Summary{Head: overview.Head{Name: miniMac, Kind: coremachine.Mac, Up: 21 * 24 * time.Hour}, Engine: overview.EngineRunning,
		Readiness: []overview.Check{
			ready(model.ItemRestartsAfterPowerCut, "applied"), ready(model.ItemNeverSleeps, "applied"), ready(model.ItemLogsIn, "applied"),
			ready(model.ItemDiskUnlock, "no password at boot"), ready(model.ItemPlannedRestarts, "supported"), ready(model.ItemEngineAtLogin, "installed"),
		},
		VMs: []overview.VM{{Name: "infra", Running: true}, {Name: "linux-3", Running: true}, {Name: "review-bot", Running: true}},
	}
}

func linux1Inside() overview.Summary {
	return overview.Summary{Head: overview.Head{Name: "linux-1", Kind: coremachine.Workspace, Up: 2 * time.Hour},
		Needs: []overview.Need{{Machine: "linux-1", Section: "apps", Subject: "updates", Text: "2 app updates", Call: "apps.update"}},
		Lines: []overview.Line{
			{Section: "apps", Parts: []overview.Part{overview.Count(catalog.StateOK, 38, "installed", "ok"), overview.Count(catalog.StateDegraded, 2, "updates", "update-available")}},
			{Section: "disk", Parts: []overview.Part{{Word: "6.2 of 40 GB"}}},
		}}
}

func infraInside() overview.Summary {
	return overview.Summary{Head: overview.Head{Name: "infra", Kind: coremachine.Server, Up: 12 * 24 * time.Hour}, Lines: []overview.Line{
		{Section: "ports", Parts: []overview.Part{{Word: "Public", N: 1, Key: "public", Counted: true}, {Word: "Private", N: 1, Key: "private", Counted: true}}},
		{Section: "ssh", Parts: []overview.Part{{Word: "admin@infra"}}},
	}}
}

func sceneSamples() map[string][]activity.Reading {
	return map[string][]activity.Reading{
		proName:      readingsAt(storyNow, 48*gb, 994e9, 388e9),
		miniMac:      readingsAt(storyNow, 64*gb, 994e9, 610e9),
		"linux-1":    readingsAt(storyNow, 8*gb, 40e9, 33.8e9),
		"agent-7f":   readingsAt(storyNow, 8*gb, 40e9, 35e9),
		"infra":      readingsAt(storyNow, 24*gb, 80e9, 44e9),
		"linux-3":    readingsAt(storyNow, 8*gb, 40e9, 31e9),
		"review-bot": readingsAt(storyNow, 4*gb, 40e9, 36e9),
	}
}

func loan(provider, name string, left time.Duration) accounts.Account {
	return accounts.Account{Provider: provider, State: catalog.StateOK, Origin: accounts.OriginShared, Identity: []string{"admin@jterrazz.com"}, Scopes: []string{name},
		Shared: &accounts.Loan{Name: name, At: storyNow.Add(left - 12*time.Hour), Until: storyNow.Add(left)}}
}

func sceneAccounts() accounts.Report {
	vm := func(name, life string, loans ...accounts.Account) accounts.Machine {
		return accounts.Machine{Name: name, Kind: accounts.KindVM, Lifetime: life, Running: true, Guest: accounts.GuestConnected, Accounts: loans}
	}
	return accounts.Report{Now: storyNow, Engine: accounts.EngineRunning, Machines: []accounts.Machine{
		{Name: proName, Kind: accounts.KindMac, ThisMac: true, Accounts: []accounts.Account{
			own(catalog.ProviderGitHub, []string{"jterrazz"}, []string{"repo", "workflow"}, "", "github.com"),
			{Provider: catalog.ProviderNPM, State: catalog.StateBroken, Reason: accounts.ReasonRefused, Origin: accounts.OriginOwn, Identity: []string{"?"}, Detail: "signed out of registry.npmjs.org", Hint: "npm login"},
			own(catalog.ProviderClaude, []string{"admin@jterrazz.com"}, []string{"max"}, "", "claude.ai"),
		}},
		vm("linux-1", "throwaway", loan(catalog.ProviderClaude, catalog.ClaudeTokenName, 24*time.Minute), loan(catalog.ProviderGitHub, "github", 4*time.Hour)),
		vm("linux-3", "throwaway", loan(catalog.ProviderGitHub, "github", 6*time.Hour)),
		vm("review-bot", "permanent", loan(catalog.ProviderClaude, catalog.ClaudeTokenName, 9*time.Hour)),
	}}
}

func proAwake() awake.Report {
	until := time.Date(2026, 11, 7, 18, 0, 0, 0, time.UTC)
	reading := awake.Reading{Lid: "open", Source: "ac", HasBattery: true, Percent: 80}
	return awake.Report{Now: storyNow, Zone: time.UTC, Engine: awake.EngineRunning, Mac: proName, ThisMac: true, Reading: reading,
		Caps:    coremachine.Capabilities{Battery: true, Lid: true, Thermal: true, Engine: true},
		Session: &awake.Session{Lid: true, StartedAt: storyNow.Add(-time.Hour), Until: until, By: "station awake --until 18:00 --lid"}}
}

func miniAwakeReport() awake.Report {
	return awake.Report{Now: storyNow, Zone: time.UTC, Engine: awake.EngineRunning, Mac: miniMac, Reading: awake.Reading{Source: "ac"},
		Caps:        coremachine.Capabilities{Thermal: true, Engine: true},
		NeverSleeps: true, NeverHeld: true, NeverSince: storyNow.Add(-21 * 24 * time.Hour), Servers: []string{"infra"}}
}

func airAwake() awake.Report {
	return awake.Report{Now: storyNow, Zone: time.UTC, Mac: airName, Caps: coremachine.Capabilities{Battery: true, Lid: true}}
}

func proActivity() pageactivity.View {
	v := thisMacActivity()
	v.Name = proName
	shift := storyNow.Sub(v.Report.Last.Sample.At)
	v.Report.Since = v.Report.Since.Add(shift)
	v.Report.Last.Sample.At = storyNow
	v.Report.Last.Sample.Boot = storyNow.Add(-3 * 24 * time.Hour)
	v.Report.Last.Sample.MemTotal, v.Report.Last.Sample.MemUsed = 48*gb, 31*gb
	return v
}

func (f Scene) reading() pagemenubar.Reading {
	return pagemenubar.Reading{Home: &f.Home, Samples: f.Samples[proName], All: &f.All, Ports: &f.Ports, Awake: &f.Awake, Macs: f.Macs, Away: f.Away, Accounts: f.shares()}
}

func (f Scene) machine(name string) pagemenubar.Machine {
	m := pagemenubar.Machine{Asks: Levels, Name: name, All: &f.All, Samples: f.Samples[name], Ports: &f.Ports, Accounts: &f.Accounts, Now: storyNow}
	if name == proName {
		m.Summary = &f.Home
	}
	for i, s := range f.Away {
		if s.Head.Name == name {
			m.Summary = &f.Away[i]
		}
	}
	if s, ok := f.Insides[name]; ok {
		m.Summary = &s
	}
	return m
}

func (f Scene) portsOf(mac string) ports.Listing {
	l := ports.Listing{AllMachines: ports.AllMachines{Schema: ports.AllMachinesSchema}}
	for _, h := range f.Ports.Macs {
		if mac == "" || h.Name == mac {
			l.AllMachines.Macs = append(l.AllMachines.Macs, h)
		}
	}
	return l
}

func (f Scene) awakeAll() pageawake.AllMachines {
	reports := append([]awake.Report{f.Awake}, f.Macs...)
	return pageawake.AllMachines{Now: storyNow, Zone: time.UTC, Reports: reports, Label: func(mac string) string { return page.MacName(mac, proName) }}
}

func (f Scene) draw() []Fixture {
	thisMac := page.Context{Machine: proName, Kind: "mac"}
	r := f.reading()
	out := []Fixture{
		{Name: "menubar", Page: pagemenubar.Page(r, thisMac)},
		{Name: "switcher", Page: pagemenubar.Switcher(r)},
		{Name: "ports", Page: pagemenubar.Ports(f.portsOf(""), pagemenubar.PortsOf{}, thisMac)},
		{Name: "awake", Page: pagemenubar.Awake(f.awakeAll(), thisMac)},
		{Name: "activity", Page: pagemenubar.Activity(f.Activity, thisMac)},
	}
	for _, h := range f.All.Macs {
		out = append(out, Fixture{Name: "machine." + h.Name, Page: pagemenubar.MachinePage(f.machine(h.Name))})
		if h.Reachable {
			of := pagemenubar.PortsOf{Machine: h.Name}
			if h.ThisMac {
				of.Machine = ""
			}
			out = append(out, Fixture{Name: "ports." + h.Name, Page: pagemenubar.Ports(f.portsOf(h.Name), of, thisMac)})
		}
		for _, vm := range h.VMs {
			out = append(out, Fixture{Name: "machine." + vm.Name, Page: pagemenubar.MachinePage(f.machine(vm.Name))})
		}
	}
	return out
}

func infraDown(f *Scene) {
	for i, h := range f.Ports.Macs {
		for j, p := range h.Ports {
			if p.Machine.Name == "infra" {
				f.Ports.Macs[i].Ports[j].Answer = ports.Unreachable
			}
		}
	}
	vms := f.All.Macs[1].VMs
	vms[0].State, vms[0].Detail, vms[0].BootAt = model.Stopped, "out of memory", time.Time{}
	vms[0].Cause, vms[0].LastLines = "out of memory", []string{
		"[ 4126.903117] k3s[812]: etcd: slow fdatasync, took 1.9 s",
		"[ 4127.551204] Out of memory: Killed process 812 (k3s-server) total-vm:2318764kB",
		"[ 4127.611008] oom_reaper: reaped process 812 (k3s-server), now anon-rss:0kB",
	}
	vms[0].CheckStatus = model.CheckStatus{State: model.CheckFailing, Since: storyNow.Add(-4 * time.Minute), At: storyNow.Add(-20 * time.Second)}
	vms[0].Restarts = model.Restarts{Count: 5, Streak: 5, Looping: true, Wait: time.Minute, Last: storyNow.Add(-time.Minute)}
}

func nothingNeedsYou(f *Scene) {
	f.Home.Needs = nil
	f.Accounts.Machines[1].Accounts[0].Shared.Until = storyNow.Add(4 * time.Hour)
	f.Home.Lines[1].Parts = f.Home.Lines[1].Parts[:1]
	f.Home.Lines[2].Parts[1] = overview.Part{State: catalog.StateOK, N: 12, Of: 12, Key: "applied", Counted: true}
	f.Home.Lines[5].Parts = f.Home.Lines[5].Parts[:1]
	f.Home.Lines[6].Parts = f.Home.Lines[6].Parts[:1]
}

func thisMacAlone(f *Scene) {
	pro := f.All.Macs[0]
	pro.VMs = nil
	f.All.Macs = []model.MacView{pro}
	f.Ports.Macs = []ports.MacPorts{{Name: proName, ThisMac: true, Reachable: true, Engine: ports.EngineInfo{Running: true}, PublicName: proName + tailnet, Serve: "on", HTTPS: "port"}}
	f.Home.VMs, f.Away, f.Macs = nil, nil, nil
	nothingNeedsYou(f)
}

func miniStillAnswering(f *Scene) {
	f.Away = nil
	f.Macs = f.Macs[1:]
	f.All.Macs[1].Answered = storyNow.Add(-40 * time.Second)
}
