package pages

import (
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/accounts"
	model "github.com/jterrazz/studio-station/src/machines"

	"github.com/jterrazz/studio-station/src/activity"
	"github.com/jterrazz/studio-station/src/awake"
	"github.com/jterrazz/studio-station/src/catalog"
	coremachine "github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/engine"
	"github.com/jterrazz/studio-station/src/overview"
	"github.com/jterrazz/studio-station/src/page"
	pagemenubar "github.com/jterrazz/studio-station/src/pages/menubar"
	"github.com/jterrazz/studio-station/src/ports"
	"github.com/jterrazz/studio-station/src/ui/token"
)

func samples() []activity.Reading {
	return readings(16*gb, 0, 0)
}

// value: 16 readings 2 s apart of a machine with this memory and disk,
// its CPU climbing and falling as vision 07's sparkline draws it.
func readings(memory, disk, free int64) []activity.Reading {
	return readingsAt(keepAwakeNow, memory, disk, free)
}

func readingsAt(now time.Time, memory, disk, free int64) []activity.Reading {
	cpu := []float64{12, 18, 25, 34, 30, 22, 41, 55, 48, 36, 28, 31, 44, 38, 29, 34}
	top := activity.Busiest{
		CPU:    []activity.Process{{PID: 812, Name: "station", Value: cpu[len(cpu)-1]}},
		Memory: []activity.Process{{PID: 812, Name: catalog.HypervisorApp, Value: 12.4e9}},
	}
	used := memory * 31 / 40
	at := func(i int) time.Time { return now.Add(time.Duration(i-len(cpu)) * 2 * time.Second) }
	out := make([]activity.Reading, 0, len(cpu)+1)
	out = append(out, activity.Reading{Sample: activity.Sample{At: at(0), CPUTotal: 1000, MemUsed: used, MemTotal: memory, DiskTotal: disk, DiskFree: free}, Top: top})
	for i, c := range cpu {
		prev := out[len(out)-1].Sample
		out = append(out, activity.Reading{Sample: activity.Sample{
			At: at(i + 1), CPUBusy: prev.CPUBusy + uint64(c*10), CPUTotal: prev.CPUTotal + 1000, MemUsed: used + int64(i%4)*gb/4, MemTotal: memory,
			NetIn: prev.NetIn + 2_000_000 + uint64(c*40_000), NetOut: prev.NetOut + uint64(200_000), DiskTotal: disk, DiskFree: free,
		}, Top: top})
	}
	return out
}

var Levels pagemenubar.Levels // value: the confirmation rule the machine pages draw, set by the generator that owns the calls (confirm.For).

func menubarHere() page.Context { return page.Context{Machine: thisMacName, Kind: "mac"} }

const (
	thisMacName = "jterrazz-macbook-pro"
	miniName    = "jterrazz-mac-mini"
)

// value: the two-macs world of USE-CASES §0.2 as the menu bar reads it:
// This Mac with linux-1 (running, two ports shared) and api (stopped), the
// mini with infra, a healthy server.
func twoMacs() model.AllMachines {
	linux1 := sized(model.VM{Name: "linux-1", Image: "linux", Version: "3", Mac: thisMacName, Kind: catalog.Linux, Lifetime: model.Throwaway, State: model.Running, SizeGB: 20, IP: "192.168.64.5", Guest: "connected", CreatedAt: built}, 4, 4)
	api := sized(model.VM{Name: "api", Image: "linux", Version: "3", Mac: thisMacName, Kind: catalog.Linux, Lifetime: model.Permanent, State: model.Stopped, SizeGB: 40, CreatedAt: built}, 4, 8)
	infra := sized(model.VM{Name: "infra", Image: "linux", Version: "3", Mac: miniName, Kind: catalog.Linux, Lifetime: model.AlwaysOn, State: model.Running, SizeGB: 80, IP: "192.168.64.7", Guest: "connected", Want: "running",
		Check: "https://infra:6443/readyz", Device: "infra", CreatedAt: built, CheckStatus: model.CheckStatus{State: model.CheckPassing, At: keepAwakeNow.Add(-40 * time.Second), Answer: "200"}}, 8, 24)
	infra.Resources.DiskBytes = 80 * gb
	running := model.EngineView{Known: true, Running: true, Version: "v3.0.0"}
	infra.BootAt = keepAwakeNow.Add(-(12*24*time.Hour + 4*time.Hour))
	return model.AllMachines{Read: keepAwakeNow, Macs: []model.MacView{
		{Name: thisMacName, User: "jterrazz", StationVersion: "v3.0.0", ThisMac: true, Reachable: true, Engine: running, Node: model.Node{Online: true, IP: "100.98.133.14"}, VMs: []model.VM{linux1, api}},
		{Name: miniName, User: "jterrazz", StationVersion: "v3.0.0", Reachable: true, Engine: running, Node: model.Node{Online: true, IP: "100.98.133.20"}, LAN: "192.168.1.20", VMs: []model.VM{infra}},
	}}
}

func twoMacsPorts() ports.AllMachines {
	public := "jterrazz-macbook-pro.tail77a797.ts.net"
	return ports.AllMachines{Schema: ports.AllMachinesSchema, Macs: []ports.MacPorts{
		{Name: thisMacName, ThisMac: true, Reachable: true, Engine: ports.EngineInfo{Running: true}, PublicName: public, Serve: "on", HTTPS: "port",
			Shared: []ports.SharedPort{
				{VM: "linux-1", GuestPort: 5173, Reserved: 40001, State: ports.PhaseShared, Process: "node", HTTP: true, Profile: "vite", URLs: ports.URLSet{Public: "https://" + public + ":40001"}},
				{VM: "linux-1", GuestPort: 7000, Reserved: 40002, State: ports.PhaseShared, Process: "nc"},
			}},
		{Name: miniName, Reachable: true, Engine: ports.EngineInfo{Running: true}},
	}}
}

// value: This Mac's Overview in two-macs: five things need you, the app
// updates first, whose fix is the apps' update.
func twoMacsSummary(updates bool) overview.Summary {
	s := overviewOf(true)
	apps := slices.IndexFunc(s.Needs, func(n overview.Need) bool { return n.Section == "apps" })
	s.Needs = append([]overview.Need{{Section: "apps", Subject: "updates", Text: "2 app updates", Call: "apps.update"}}, slices.Delete(slices.Clone(s.Needs), apps, apps+1)...)
	s.Lines[1].Parts[1] = overview.Count(catalog.StateDegraded, 2, "updates", "update-available")
	s.Lines[5].Parts = []overview.Part{overview.Count(catalog.StateOK, 8, "", "signed-in"), {State: catalog.StateBroken, Word: "npm"}}
	if !updates {
		s.Needs = s.Needs[1:]
		s.Lines[1].Parts = slices.Delete(slices.Clone(s.Lines[1].Parts), 1, 2)
	}
	return s
}

func thisMacAwake() *awake.Report {
	until := time.Date(2026, 11, 3, 18, 0, 0, 0, time.UTC)
	return &awake.Report{Now: keepAwakeNow, Engine: awake.EngineRunning, Mac: thisMacName, ThisMac: true, Reading: thisMacReading(),
		Session: &awake.Session{Lid: true, StartedAt: keepAwakeNow.Add(-time.Hour), Until: until, By: "station awake --until 18:00 --lid"}}
}

func miniAwake() awake.Report {
	return awake.Report{Now: keepAwakeNow, Engine: awake.EngineRunning, Mac: miniName, NeverSleeps: true, NeverHeld: true, NeverSince: keepAwakeNow.Add(-12 * 24 * time.Hour), Servers: []string{"infra"}}
}

func twoMacsReading(updates bool) pagemenubar.Reading {
	f, svc, home := twoMacs(), twoMacsPorts(), twoMacsSummary(updates)
	return pagemenubar.Reading{Home: &home, Samples: samples(), All: &f, Ports: &svc, Awake: thisMacAwake(), Macs: []awake.Report{miniAwake()}, Away: []overview.Summary{miniSummary(false)}}
}

func menubarReadings() (frame, quiet, full pagemenubar.Reading) {
	off := keepAwakeReport(awake.EngineRunning, nil, thisMacReading())
	off.Engine, off.ThisMac, off.Mac = awake.EngineRunning, true, thisMacName
	calm := overviewOf(false)
	calm.Lines = calm.Lines[:1]
	calm.VMs = nil
	f := twoMacs()
	alone := model.AllMachines{Macs: []model.MacView{f.Macs[0]}}
	alone.Macs[0].VMs = nil
	frame = pagemenubar.Reading{Awake: &off}
	quiet = pagemenubar.Reading{Home: &calm, Samples: samples(), All: &alone, Awake: &off}
	full = twoMacsReading(true)
	return frame, quiet, full
}

func menubarPages() []Fixture {
	frame, quiet, full := menubarReadings()
	return append(states("menubar",
		pagemenubar.Page(frame, menubarHere()),
		pagemenubar.Page(quiet, menubarHere()),
		pagemenubar.Stopped(thisMacContext(), true),
		pagemenubar.Page(full, menubarHere())), menubarWorlds()...)
}

// value: the menu bar's pages as each world of USE-CASES §0.2 the menu
// bar's proofs cite draws them, named menubar.<page>.<world>.
func menubarWorlds() []Fixture {
	_, quiet, _ := menubarReadings()
	two := twoMacsReading(true)
	down := *two.All
	down.Macs = slices.Clone(down.Macs)
	infra := down.Macs[1].VMs[0]
	infra.State, infra.Detail = model.Stopped, "restarting"
	infra.Restarts = model.Restarts{Count: 5, Streak: 5, Looping: true, Wait: time.Minute}
	down.Macs[1].VMs = []model.VM{infra}
	infraDown := two
	infraDown.All = &down
	return []Fixture{
		{Name: "menubar.home.one-mac", Page: pagemenubar.Page(quiet, menubarHere())},
		{Name: "menubar.home.two-macs", Page: pagemenubar.Page(two, menubarHere())},
		{Name: "menubar.home.infra-down", Page: pagemenubar.Page(infraDown, menubarHere())},
		{Name: "menubar.home.engine-stopped", Page: pagemenubar.Stopped(menubarHere(), true)},
		{Name: "menubar.home.engine-outdated", Page: pagemenubar.Outdated(menubarHere())},
		{Name: "menubar.switcher.two-macs", Page: pagemenubar.Switcher(two)},
		{Name: "menubar.vm.linux-1", Page: pagemenubar.MachinePage(linux1Page())},
		{Name: "menubar.vm.infra", Page: pagemenubar.MachinePage(infraPage())},
		{Name: "menubar.mac.mini", Page: pagemenubar.MachinePage(miniPage(false))},
		{Name: "menubar.mac.mini-offline", Page: pagemenubar.MachinePage(miniPage(true))},
	}
}

// value: linux-1's page: its Overview read inside it, GitHub and Claude
// shared from This Mac, its two ports shared.
func linux1Page() pagemenubar.Machine {
	f, svc := twoMacs(), twoMacsPorts()
	inside := overview.Summary{Head: overview.Head{Name: "linux-1", Kind: coremachine.Workspace, Up: 2 * time.Hour}, Lines: []overview.Line{
		{Section: "apps", Parts: []overview.Part{overview.Count(catalog.StateOK, 38, "installed", "ok")}},
		{Section: "settings", Parts: []overview.Part{overview.Count(catalog.StateDegraded, 2, "need a look", "attention")}},
		{Section: "disk", Parts: []overview.Part{{Word: "2.4 of 20 GB"}}},
	}}
	shares := accounts.Report{Now: keepAwakeNow, Engine: accounts.EngineRunning, Machines: []accounts.Machine{{Name: "linux-1", Kind: accounts.KindVM, Lifetime: "throwaway", Running: true, Accounts: []accounts.Account{
		{Provider: catalog.ProviderGitHub, State: catalog.StateOK, Origin: accounts.OriginShared, Scopes: []string{"read only"}, Shared: &accounts.Loan{Name: "github", At: keepAwakeNow.Add(-8 * time.Hour), Until: keepAwakeNow.Add(4 * time.Hour)}},
		{Provider: catalog.ProviderClaude, State: catalog.StateOK, Origin: accounts.OriginShared, Shared: &accounts.Loan{Name: catalog.ClaudeTokenName, At: keepAwakeNow, Until: keepAwakeNow.Add(12 * time.Hour)}},
	}}}}
	return pagemenubar.Machine{Asks: Levels, Name: "linux-1", All: &f, Summary: &inside, Samples: readings(4*gb, 20_000_000_000, 17_600_000_000), Ports: &svc, Accounts: &shares, Now: keepAwakeNow}
}

// value: infra's page from This Mac: up 12 days, its four health steps
// passing, one public port and two private.
func infraPage() pagemenubar.Machine {
	f, svc := twoMacs(), twoMacsPorts()
	inside := overview.Summary{Head: overview.Head{Name: "infra", Kind: coremachine.Server, Up: 12*24*time.Hour + 4*time.Hour}, Lines: []overview.Line{
		{Section: "ports", Parts: []overview.Part{{Word: "Public", N: 1, Key: "public", Counted: true}, {Word: "Private", N: 2, Key: "private", Counted: true}}},
		{Section: "ssh", Parts: []overview.Part{{Word: "admin@infra"}}},
	}}
	return pagemenubar.Machine{Asks: Levels, Name: "infra", All: &f, Summary: &inside, Samples: readings(24*gb, 80_000_000_000, 44_000_000_000), Ports: &svc, Now: keepAwakeNow}
}

// value: mac-mini's page as This Mac reads it from the mini's engine: its
// readiness all applied, infra, and the port that needs you; offline, the
// mini waits at pre-boot for its disk's password.
func miniPage(offline bool) pagemenubar.Machine {
	f, svc := twoMacs(), twoMacsPorts()
	s := miniSummary(offline)
	if offline {
		f.Macs[1].Reachable, f.Macs[1].Node.Online, f.Macs[1].Engine = false, false, model.EngineView{}
	}
	return pagemenubar.Machine{Asks: Levels, Name: miniName, All: &f, Summary: &s, Ports: &svc, Now: keepAwakeNow}
}

// value: mac-mini's Overview, as its engine reads it: its readiness all
// applied and the port that needs you; offline, it waits at pre-boot for
// its disk's password.
func miniSummary(offline bool) overview.Summary {
	ready := func(name, value string) overview.Check {
		return overview.Check{Name: name, State: catalog.StateOK, Value: value}
	}
	s := overview.Summary{Head: overview.Head{Name: miniName, Kind: coremachine.Mac}, Engine: overview.EngineRunning,
		Needs: []overview.Need{{Machine: miniName, Section: "ports", Broken: true, Subject: "5000", Text: "port 5000 is reachable outside Tailscale", Command: "station ports --on mac-mini"}},
		Readiness: []overview.Check{
			ready(model.ItemRestartsAfterPowerCut, "applied"), ready(model.ItemNeverSleeps, "applied"), ready(model.ItemLogsIn, "applied"),
			ready(model.ItemDiskUnlock, "no password at boot"), ready(model.ItemPlannedRestarts, "supported"), ready(model.ItemEngineAtLogin, "installed"),
		}}
	if offline {
		s.Engine, s.Needs = overview.EngineNone, nil
		s.Readiness[3] = overview.Check{Name: model.ItemDiskUnlock, State: catalog.StateDegraded, Value: "waits at pre-boot", Detail: "station machines unlock mac-mini (from the same network)"}
	}
	return s
}

func MenubarStream() []byte {
	// invariant: the recorded stream is written by the daemon's own types from
	// the menubar fixtures: the Swift client reads the bytes a daemon sends.
	byName := map[string]page.Page{}
	for _, f := range All() {
		byName[f.Name] = f.Page
	}
	frame, _ := json.Marshal(page.Document{Schema: page.Schema, Page: byName["menubar.loading"]})
	ready, _ := json.Marshal(page.Document{Schema: page.Schema, Page: byName["menubar.full"]})
	outs := []any{
		engine.Opened{Stream: page.Schema, Pages: []string{engine.PageMenubar}},
		engine.Event{Event: "page", Page: frame},
		engine.Event{Event: "page", Page: ready},
		sampled(byName["menubar.full"].Live, "2026-11-03T10:30:02Z", 3, false),
		sampled(byName["menubar.full"].Live, "2026-11-03T10:30:04Z", 4, true),
		engine.Event{Event: "tick", At: "2026-11-03T10:30:15Z"},
		engine.Event{Event: "notice", Notice: &engine.Notice{Kind: engine.NoticeAwakeEnded, Icon: engine.NoticeIcon(engine.NoticeAwakeEnded), Text: engine.AwakeStopped("battery 20 %")}},
	}
	type answer struct {
		V   int    `json:"v"`
		ID  string `json:"id"`
		Out any    `json:"out"`
	}
	lines := make([]answer, 0, len(outs))
	for _, o := range outs {
		lines = append(lines, answer{V: 2, ID: "1", Out: o})
	}
	var out []byte
	for _, l := range lines {
		data, err := json.Marshal(l)
		if err != nil {
			panic(err)
		}
		out = append(append(out, data...), '\n')
	}
	return out
}

// value: one tick of the live keys a page lists, read over n readings of
// This Mac 2 s apart; stale, every key is told late.
func sampled(keys []string, at string, n int, stale bool) engine.Event {
	readings := make([]activity.Reading, 0, n)
	for i := range n {
		busy := uint64(i*i) * 150
		readings = append(readings, activity.Reading{Sample: activity.Sample{At: time.Date(2026, 11, 3, 10, 30, 2*i, 0, time.UTC), CPUBusy: busy, CPUTotal: uint64(i) * 1000,
			MemUsed: 9 << 30, MemTotal: 16 << 30, NetIn: uint64(i) * 2_400_000, NetOut: uint64(i) * 600_000}})
	}
	e := engine.Event{Event: engine.EventSample, At: at, Values: map[string]any{}}
	for _, k := range keys {
		v := pagemenubar.LiveValue(k, readings)
		if v.Kind == "" {
			continue
		}
		if stale {
			v.Stale = true
			e.Stale = append(e.Stale, k)
		}
		e.Values[k] = v
	}
	return e
}

func EventsStream() []byte {
	// invariant: the recorded events watch is written by the engine's own
	// event words, one frame per event of SPEC 10.4 the engine tells, so the
	// menu bar's notifications read what an engine sends.
	var loop model.Restarts
	at := time.Date(2026, 11, 3, 14, 0, 0, 0, time.UTC)
	for i := range 5 {
		loop = loop.After(time.Second, at.Add(time.Duration(i)*time.Minute))
	}
	exposed := ports.Exposed(ports.Survey([]ports.Machine{ports.MacOf("mac-mini", []ports.Listener{{Port: 5000, Bind: "*", Process: "ControlCenter"}})}, nil))
	low := activity.Sample{DiskFree: 3_000_000_000, DiskTotal: 100_000_000_000}
	events := []engine.Told{
		engine.Restarting("infra", 3), engine.Looping("infra", loop), engine.PoweredOff("infra"), engine.Back("infra"),
		{Kind: engine.NoticeAwakeEnded, Parts: []string{engine.AwakeStopped("battery 20 %")}},
		{Kind: engine.EventExposed, Parts: []string{ports.Told(exposed[0])}},
		{Kind: engine.EventDiskLow, Parts: []string{activity.DiskFreeTold("This Mac", low)}},
	}
	outs := make([]any, 0, 1+len(events))
	outs = append(outs, engine.Opened{Stream: engine.EventsStream})
	for _, e := range events {
		outs = append(outs, engine.Event{Event: "notice", Notice: &engine.Notice{Kind: e.Kind, Icon: engine.NoticeIcon(e.Kind), Text: strings.Join(e.Parts, token.Sep)}})
	}
	var out []byte
	for _, o := range outs {
		data, err := json.Marshal(struct {
			V   int    `json:"v"`
			ID  string `json:"id"`
			Out any    `json:"out"`
		}{V: 2, ID: "1", Out: o})
		if err != nil {
			panic(err)
		}
		out = append(append(out, data...), '\n')
	}
	return out
}
