package pages

import (
	"bytes"
	"encoding/json"
	"slices"
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/background"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/images"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/overview"
	"github.com/jterrazz/studio-station/apps/engine/ports"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

const (
	PlayWorldPath = "specs/usecases/worlds/stories.json"
	PlayWorldBase = "specs/usecases/worlds/two-macs.json"
)

type playWorld struct {
	Now      time.Time           `json:"now"`
	ThisMac  string              `json:"this_mac"`
	User     string              `json:"user"`
	Macs     map[string]*playMac `json:"macs"`
	Registry map[string]string   `json:"registry,omitempty"`
}

type playMac struct {
	Engine      *playEngine                    `json:"engine,omitempty"`
	Offline     bool                           `json:"offline,omitempty"`
	Preferences map[string]any                 `json:"preferences,omitempty"`
	Images      []string                       `json:"images,omitempty"`
	VMs         map[string]*playVM             `json:"vms,omitempty"`
	Jobs        map[string]background.Job      `json:"jobs,omitempty"`
	Secrets     map[string]string              `json:"secrets,omitempty"`
	Identities  map[string]accounts.Identity   `json:"identities,omitempty"`
	Facts       map[catalog.Fact]settings.Fact `json:"facts,omitempty"`
	Self        *playSelf                      `json:"self,omitempty"`
	Listening   []ports.Listener               `json:"listening,omitempty"`
	Tailnet     *playNode                      `json:"tailnet,omitempty"`
	Apps        map[catalog.Method]*playApps   `json:"apps,omitempty"`
	Meter       *playMeter                     `json:"meter,omitempty"`
	Disk        *playDisk                      `json:"disk,omitempty"`
	Stores      *playStores                    `json:"stores,omitempty"`
	Files       map[string]string              `json:"files,omitempty"`
}

type playEngine struct {
	Version   string        `json:"version,omitempty"`
	StartedAt time.Time     `json:"started_at,omitzero"`
	Memory    engine.Memory `json:"memory,omitempty"`
	Awake     *awake.Kept   `json:"awake,omitempty"`
	// value: the activity the engine kept before the story, by machine.
	Past map[string]activity.Profile `json:"past,omitempty"`
}

type playSelf struct {
	Kind     coremachine.Kind     `json:"kind,omitempty"`
	System   coremachine.System   `json:"system"`
	Hardware coremachine.Hardware `json:"hardware"`
}

type playNode struct {
	Self   ports.Self    `json:"self"`
	Shares []ports.Share `json:"shares,omitempty"`
}

type playApps struct {
	Installed map[string]string `json:"installed,omitempty"`
	Behind    map[string]string `json:"behind,omitempty"`
}

type playMeter struct {
	Sample    activity.Sample        `json:"sample"`
	PerSecond *activity.Sample       `json:"per_second,omitempty"`
	Processes map[int]activity.Usage `json:"processes,omitempty"`
}

type playDisk struct {
	Total int64 `json:"total,omitempty"`
	Free  int64 `json:"free,omitempty"`
}

type playStores struct {
	Images       []images.Record             `json:"images,omitempty"`
	Shares       map[string][]accounts.Share `json:"shares,omitempty"`
	Reservations []ports.Reservation         `json:"reservations,omitempty"`
	Declarations []ports.Declaration         `json:"declarations,omitempty"`
	Uptime       map[string]model.Uptime     `json:"uptime,omitempty"`
}

type playVM struct {
	Kept      *playKept        `json:"kept,omitempty"`
	Disk      bool             `json:"disk,omitempty"`
	Size      model.Size       `json:"size"`
	PID       int              `json:"pid,omitempty"`
	UpAt      time.Time        `json:"up_at,omitzero"`
	Guest     *struct{}        `json:"guest,omitempty"`
	Listening []ports.Listener `json:"listening,omitempty"`
	Agents    *playAgents      `json:"agents,omitempty"`
	Console   []string         `json:"console,omitempty"`
}

type playKept struct {
	Image     string         `json:"image,omitempty"`
	Version   string         `json:"version,omitempty"`
	Lifetime  model.Lifetime `json:"lifetime"`
	Want      string         `json:"want,omitempty"`
	Check     string         `json:"check,omitempty"`
	CreatedAt time.Time      `json:"created_at,omitzero"`
	Template  string         `json:"template,omitempty"`
}

type playAgents struct {
	Journal   map[string]agents.Agent `json:"journal,omitempty"`
	Hooks     map[agents.App]bool     `json:"hooks,omitempty"`
	Terminals []agents.Terminal       `json:"terminals,omitempty"`
	PRs       map[int]string          `json:"prs,omitempty"`
}

func PlayWorld(base []byte) []byte {
	f := everyMachine()
	infraServes(&f)
	cpu := map[string]float64{proName: .30, miniMac: .55, "linux-1": .37, "agent-7f": .04, "infra": .18, "linux-3": .60, "review-bot": .06}
	hardware := map[string]coremachine.Hardware{
		proName: {Model: "Mac16,5", Chip: "Apple M4 Max", Memory: 48 * gb, CPUs: 16},
		miniMac: {Model: "Mac16,11", Chip: "Apple M4 Pro", Memory: 64 * gb, CPUs: 14},
		airName: {Model: "Mac15,12", Chip: "Apple M3", Memory: 24 * gb, CPUs: 8},
	}
	w := playWorld{Now: storyNow, ThisMac: proName, User: "jterrazz", Macs: map[string]*playMac{}, Registry: map[string]string{"station-linux": "3"}}
	known := map[string]any{}
	past := profiles()
	pid := 1000
	for _, h := range f.All.Macs {
		hw := hardware[h.Name]
		if !h.Reachable && !h.ThisMac {
			known[h.Name] = map[string]any{"user": h.User, "answered": h.Answered, "hardware": hw, "laptop": true, "asleep": true}
			w.Macs[h.Name] = &playMac{Offline: true, Engine: &playEngine{Version: stationVer, StartedAt: storyNow.Add(-5 * 24 * time.Hour)},
				Preferences: map[string]any{"vm_user": "admin"}, Self: &playSelf{Hardware: hw}}
			continue
		}
		if !h.ThisMac {
			known[h.Name] = map[string]any{"user": h.User, "lan": h.LAN}
		}
		m := &playMac{
			Engine:      &playEngine{Version: stationVer, StartedAt: storyNow.Add(-macUp(f, h.Name))},
			Preferences: map[string]any{"vm_user": "admin"},
			Images:      []string{"ghcr.io/jterrazz/station-linux:3"},
			VMs:         map[string]*playVM{},
			Self:        &playSelf{Hardware: hw},
			Tailnet:     &playNode{Self: ports.Self{Node: ports.Node{ID: h.Name, Name: h.Name, DNSName: h.Name + tailnet, IP: h.Node.IP, Online: true}, Sharing: ports.SharingState{Backend: "Running", On: true, DNSName: h.Name + tailnet}}},
			Meter:       playMeterOf(past[h.Name], meterSample(cpu[h.Name], hw.CPUs, hw.Memory, storyNow.Add(-macUp(f, h.Name)))),
			Facts:       macFacts(h.ThisMac),
			Stores:      &playStores{Images: []images.Record{{Name: "station-linux", Version: "3", Base: "ghcr.io/cirruslabs/ubuntu:24.04", Station: stationVer, BuiltAt: storyNow.Add(-6 * 24 * time.Hour), SizeGB: 4}}},
		}
		m.VMs["station-image-station-linux-3"] = &playVM{Disk: true, Size: model.Size{Disk: 4e9}}
		if h.ThisMac {
			m.Engine.Awake = &awake.Kept{Session: f.Awake.Session}
			m.Disk = &playDisk{Total: 994e9, Free: 388e9}
			m.Files = ownerFiles()
			m.Jobs = map[string]background.Job{"com.jterrazz.brain.x-snapshot": {Label: "com.jterrazz.brain.x-snapshot", Program: "~/.local/bin/x-snapshot", File: "~/Library/LaunchAgents/com.jterrazz.brain.x-snapshot.plist", Scope: "user", Loaded: true, LastExit: 2, ProgramExists: true, OwnerExists: true}}
			m.Apps = map[catalog.Method]*playApps{
				catalog.MethodBrew: {Installed: map[string]string{"jq": "1.8.1", "gh": "2.97.0", "lazygit": "0.55.1"}, Behind: map[string]string{"gh": "2.98.0"}},
				catalog.MethodCask: {Installed: map[string]string{"ghostty": "1.2.0"}, Behind: map[string]string{"ghostty": "1.2.1"}},
			}
		} else {
			m.Preferences["never_sleeps"] = true
			m.Disk = &playDisk{Total: 994e9, Free: 610e9}
		}
		m.Identities, m.Secrets = playAccounts(f, h.Name)
		m.Engine.Past = map[string]activity.Profile{h.Name: past[h.Name]}
		m.Meter.Processes = map[int]activity.Usage{}
		for _, vm := range h.VMs {
			pid++
			m.VMs[vm.Name] = playMachine(f, h.Name, vm, pid)
			if vm.State == model.Running {
				if p, ok := past[vm.Name]; ok {
					m.Engine.Past[vm.Name] = p
				}
				m.Meter.Processes[pid] = activity.Usage{CPU: 100 * cpu[vm.Name] * float64(vm.Resources.CPUs) / float64(hw.CPUs), Memory: vm.Resources.MemoryBytes * 2 / 3, Started: vm.BootAt}
			}
			if vm.State == model.Running && !vm.BootAt.IsZero() {
				if m.Engine.Memory == nil {
					m.Engine.Memory = engine.Memory{}
				}
				m.Engine.Memory[vm.Name] = engine.Remembered{State: engine.StateRunning, BootAt: vm.BootAt, Check: vm.CheckStatus, Latency: vm.Latency}
			}
			if len(vm.Uptime.Days) > 0 {
				if m.Stores.Uptime == nil {
					m.Stores.Uptime = map[string]model.Uptime{}
				}
				m.Stores.Uptime[vm.Name] = vm.Uptime
			}
			w.Macs[vm.Name] = inside(f, vm, cpu[vm.Name], past[vm.Name])
		}
		var notShared []string
		for _, p := range f.portsOf(h.Name).AllMachines.Macs {
			for _, port := range p.Ports {
				if port.Machine.On != h.Name {
					continue
				}
				switch {
				case port.Device:
					m.Tailnet.Self.Peers = append(m.Tailnet.Self.Peers, ports.Node{ID: port.Machine.Name, Name: port.Machine.Name, DNSName: port.Machine.Name + tailnet, Online: true})
				case port.Reach == ports.Tailnet && port.Declared == nil:
					notShared = append(notShared, port.Machine.Name+":"+strconv.Itoa(port.Port))
				}
			}
		}
		if notShared != nil {
			m.Preferences["ports"] = ports.PortsGroup{NotShared: notShared}
		}
		for _, p := range f.portsOf(h.Name).AllMachines.Macs {
			for _, s := range p.Shared {
				m.Stores.Reservations = append(m.Stores.Reservations, ports.Reservation{VM: s.VM, GuestPort: s.GuestPort, MacPort: s.Reserved, By: ports.ByOwner, HTTP: &s.HTTP, ReservedAt: storyNow.Add(-2 * time.Hour)})
				m.Tailnet.Shares = append(m.Tailnet.Shares, ports.Share{Port: s.Reserved, Reserved: s.Reserved, Scheme: "https", Target: "http://127.0.0.1:" + strconv.Itoa(s.Reserved)})
			}
			for _, port := range p.Ports {
				if port.Machine.Name == h.Name {
					m.Listening = append(m.Listening, ports.Listener{Port: port.Port, Bind: h.Node.IP, Process: port.Process, PID: 500 + port.Port%100})
				}
				if port.Declared != nil {
					m.Stores.Declarations = append(m.Stores.Declarations, *port.Declared)
				}
			}
		}
		m.Stores.Shares = loansOf(f, h)
		w.Macs[h.Name] = m
	}
	w.Macs[proName].Preferences["macs"] = known
	out, err := json.Marshal(w)
	if err != nil {
		panic(err)
	}
	out, err = inherit(out, base, map[string]string{proName: "", miniMac: "jterrazz-mac-mini"})
	if err != nil {
		panic(err)
	}
	return append(out, '\n')
}

func inherit(world, base []byte, from map[string]string) ([]byte, error) {
	var w, b struct {
		ThisMac string                                `json:"this_mac"`
		Macs    map[string]map[string]json.RawMessage `json:"macs"`
	}
	var whole map[string]json.RawMessage
	if err := json.Unmarshal(world, &w); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(world, &whole); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(base, &b); err != nil {
		return nil, err
	}
	for name, source := range from {
		if source == "" {
			source = b.ThisMac
		}
		for key, value := range b.Macs[source] {
			if _, ours := w.Macs[name][key]; !ours && key != "ssh" {
				w.Macs[name][key] = value
			}
		}
	}
	macs, err := json.Marshal(w.Macs)
	if err != nil {
		return nil, err
	}
	whole["macs"] = macs
	merged, err := json.Marshal(whole)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	err = json.Indent(&out, merged, "", "    ")
	return out.Bytes(), err
}

func macUp(f Scene, name string) time.Duration {
	for _, s := range append([]overview.Summary{f.Home}, f.Away...) {
		if s.Head.Name == name {
			return s.Head.Up
		}
	}
	return 24 * time.Hour
}

func playMeterOf(p activity.Profile, s activity.Sample) *playMeter {
	// value: a meter whose counters are what the machine's profile counted
	// since it booted, growing a second as the profile does now.
	if s.Boot.IsZero() {
		return &playMeter{Sample: s}
	}
	counted := p.Points(storyNow, storyNow.Sub(s.Boot).Truncate(activity.CoarseEvery), activity.CoarseEvery)
	end := counted[len(counted)-1]
	s.CPUBusy, s.CPUTotal, s.NetIn, s.NetOut = end.CPUBusy, end.CPUTotal, end.NetIn, end.NetOut
	step := p.Points(storyNow, activity.FineEvery, activity.FineEvery)
	every := uint64(activity.FineEvery / time.Second)
	per := func(a, b uint64) uint64 { return (b - a) / every }
	return &playMeter{Sample: s, PerSecond: &activity.Sample{
		CPUBusy: per(step[0].CPUBusy, step[1].CPUBusy), CPUTotal: per(step[0].CPUTotal, step[1].CPUTotal),
		NetIn: per(step[0].NetIn, step[1].NetIn), NetOut: per(step[0].NetOut, step[1].NetOut),
	}}
}

func meterSample(share float64, cpus int, memory int64, boot time.Time) activity.Sample {
	return activity.Sample{At: storyNow, CPUBusy: uint64(share * 1e7), CPUTotal: 1e7, CPUs: cpus, Load: [3]float64{share * float64(cpus), share * float64(cpus), share * float64(cpus)},
		MemUsed: memory * 2 / 3, MemTotal: memory, Pressure: "normal", Boot: boot}
}

func macFacts(thisMac bool) map[catalog.Fact]settings.Fact {
	ok := func(v string) settings.Fact { return settings.Fact{State: catalog.StateOK, Value: v} }
	facts := map[catalog.Fact]settings.Fact{
		"filevault": ok("on"), "firewall": ok("on"), "sip": ok("on"), "gatekeeper": ok("on"), "automatic-updates": ok("on"),
		"screen-lock": ok("0"), "display-sleep": ok("2"), "time-machine": ok(storyNow.Add(-24 * time.Hour).Format(time.RFC3339)), "system-updates": ok("up to date"),
	}
	if thisMac {
		facts["firewall"] = settings.Fact{State: catalog.StateAbsent, Value: "off"}
	}
	return facts
}

func playAccounts(f Scene, mac string) (ids map[string]accounts.Identity, secrets map[string]string) {
	ids, secrets = map[string]accounts.Identity{}, map[string]string{}
	for _, m := range f.Accounts.Machines {
		if m.Name != mac {
			continue
		}
		for _, a := range m.Accounts {
			ids[a.Provider] = accounts.Identity{Provider: a.Provider, State: a.State, Reason: a.Reason, Who: a.Identity, Scopes: a.Scopes, Detail: a.Detail}
		}
	}
	if mac != proName {
		ids[catalog.ProviderGitHub] = accounts.Identity{Provider: catalog.ProviderGitHub, State: catalog.StateOK, Who: []string{"jterrazz-agent"}, Scopes: []string{"repo"}}
		ids[catalog.ProviderClaude] = accounts.Identity{Provider: catalog.ProviderClaude, State: catalog.StateOK, Who: []string{"admin@jterrazz.com"}, Scopes: []string{"max"}}
	}
	for _, s := range loansOf(f, model.MacView{Name: mac, VMs: vmsOf(f, mac)}) {
		for _, l := range s {
			secrets[l.Name] = "fake-" + l.Name + "-of-" + mac
		}
	}
	return ids, secrets
}

func vmsOf(f Scene, mac string) []model.VM {
	for _, h := range f.All.Macs {
		if h.Name == mac {
			return h.VMs
		}
	}
	return nil
}

func loansOf(f Scene, h model.MacView) map[string][]accounts.Share {
	out := map[string][]accounts.Share{}
	for _, read := range f.Accounts.Machines {
		if !slices.ContainsFunc(h.VMs, func(vm model.VM) bool { return vm.Name == read.Name }) {
			continue
		}
		for _, a := range read.Accounts {
			if a.Shared != nil {
				out[read.Name] = append(out[read.Name], accounts.Share{Name: a.Shared.Name, At: a.Shared.At, Until: a.Shared.Until})
			}
		}
	}
	return out
}

func playMachine(f Scene, mac string, vm model.VM, pid int) *playVM {
	want := "stopped"
	if vm.State == model.Running {
		want = "running"
	}
	v := &playVM{
		Kept: &playKept{Image: vm.Image, Version: vm.Version, Lifetime: vm.Lifetime, Want: want, Check: vm.Check, CreatedAt: vm.CreatedAt},
		Disk: true, Size: model.Size{CPUs: vm.Resources.CPUs, Memory: vm.Resources.MemoryBytes, Disk: int64(vm.SizeGB) * 1e9},
	}
	if _, ok := devApps(vm.Name); ok {
		v.Kept.Template = devFile
	}
	if vm.State != model.Running {
		return v
	}
	v.PID, v.UpAt, v.Guest, v.Console = pid, vm.BootAt, &struct{}{}, consoleOf(vm.Name)
	for _, p := range f.portsOf(mac).AllMachines.Macs {
		for _, port := range p.Ports {
			if port.Machine.Name != vm.Name {
				continue
			}
			v.Listening = append(v.Listening, ports.Listener{Port: port.Port, Bind: port.Binds[0], Process: port.Process, PID: pid*10 + len(v.Listening)})
			if port.Declared != nil && port.Declared.Via == catalog.RouteCloudflareTunnel && port.Verified != nil && port.Verified.Seen {
				v.Listening = append(v.Listening, ports.Listener{Port: 20241, Bind: "127.0.0.1", Process: "cloudflared", PID: pid*10 + len(v.Listening)})
			}
		}
		for _, s := range p.Shared {
			if s.VM == vm.Name {
				v.Listening = append(v.Listening, ports.Listener{Port: s.GuestPort, Bind: "0.0.0.0", Process: s.Process, PID: pid*10 + len(v.Listening)})
			}
		}
	}
	for _, a := range f.Agents {
		if a.VM != vm.Name {
			continue
		}
		if v.Agents == nil {
			v.Agents = &playAgents{Journal: map[string]agents.Agent{}, Hooks: map[agents.App]bool{a.App: true}}
		}
		v.Agents.Journal[a.Session] = a
		if a.Terminal != "" {
			v.Agents.Terminals = append(v.Agents.Terminals, agents.Terminal{Address: a.Terminal, App: a.App, Folder: "/home/admin/app"})
		}
		if a.PR != nil {
			v.Agents.PRs = map[int]string{a.PR.Number: "open"}
		}
	}
	return v
}

func inside(f Scene, vm model.VM, share float64, p activity.Profile) *playMac {
	kind := coremachine.Workspace
	if vm.Lifetime == model.AlwaysOn {
		kind = coremachine.Server
	}
	m := &playMac{Self: &playSelf{Kind: kind, System: coremachine.System{OS: "linux"}}, Disk: &playDisk{Total: int64(vm.SizeGB) * 1e9, Free: int64(vm.SizeGB) * 1e9 * 3 / 4}}
	if vm.State == model.Running {
		m.Meter = playMeterOf(p, meterSample(share, vm.Resources.CPUs, vm.Resources.MemoryBytes, vm.BootAt))
	}
	if apt, ok := devApps(vm.Name); ok {
		m.Files = devFiles()
		m.Preferences = map[string]any{"machinefile": map[string]string{"name": devFile}}
		m.Apps = map[catalog.Method]*playApps{catalog.MethodApt: apt}
	} else if s, ok := f.Insides[vm.Name]; ok && len(s.Needs) > 0 {
		m.Apps = map[catalog.Method]*playApps{catalog.MethodBrew: {Installed: map[string]string{"jq": "1.8.1", "gh": "2.97.0"}, Behind: map[string]string{"gh": "2.98.0", "jq": "1.8.2"}}}
	}
	ids := map[string]accounts.Identity{}
	for _, read := range f.Accounts.Machines {
		if read.Name == vm.Name {
			for _, a := range read.Accounts {
				ids[a.Provider] = accounts.Identity{Provider: a.Provider, State: catalog.StateOK, Who: a.Identity}
			}
		}
	}
	if len(ids) > 0 {
		m.Identities = ids
	}
	return m
}
