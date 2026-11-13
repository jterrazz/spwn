package home

import (
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/background"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/overview"
	"github.com/jterrazz/studio-station/apps/engine/pages"
	pageapps "github.com/jterrazz/studio-station/apps/engine/pages/apps"
	pagedisk "github.com/jterrazz/studio-station/apps/engine/pages/disk"
	pageports "github.com/jterrazz/studio-station/apps/engine/pages/ports"
	pagesettings "github.com/jterrazz/studio-station/apps/engine/pages/settings"
)

const schema = "station.status/v1"

type SnapshotJSON struct {
	// value: alias of station.status/v1, the OS connector's, kept byte for
	// byte one release.
	Schema      string                   `json:"schema"`
	GeneratedAt string                   `json:"generated_at"`
	Station     stationJSON              `json:"station"`
	Machine     machineJSON              `json:"machine"`
	System      systemJSON               `json:"system"`
	Environment environmentJSON          `json:"environment"`
	Workspace   workspaceJSON            `json:"workspace"`
	Apps        []pageapps.AppStatusJSON `json:"tools"`
	Settings    []pagesettings.ItemJSON  `json:"config"`
	Setup       setupJSON                `json:"setup"`
	Identity    []checkJSON              `json:"identity"`
	Background  []jobJSON                `json:"background"`
	Gateways    []checkJSON              `json:"gateways"`
	Tailnet     pageports.TailnetJSON    `json:"tailnet"`
	Power       *awakeJSON               `json:"power,omitempty"`
}

// value: alias of station.status/v1's power, byte for byte one release.
type awakeJSON struct {
	Held  bool   `json:"held"`
	Lid   bool   `json:"lid,omitempty"`
	Until string `json:"until,omitempty"`
}

type jobJSON struct {
	Label   string `json:"label"`
	Group   string `json:"group"`
	State   string `json:"state"`
	Mark    string `json:"mark"`
	PID     int    `json:"pid"`
	Exit    int    `json:"last_exit"`
	Scope   string `json:"scope"`
	Owner   string `json:"owner"`
	Plist   string `json:"plist"`
	Program string `json:"program"`
	Removal string `json:"removal"`
}

func jobs(in []background.Task) []jobJSON {
	out := make([]jobJSON, 0, len(in))
	for _, j := range in {
		out = append(out, jobJSON{Label: j.Label, Group: string(j.Group), State: strings.ReplaceAll(string(j.Run), " ", "-"), Mark: string(pages.MarkOf(j.State)),
			PID: j.PID, Exit: j.Exit, Scope: j.Scope, Owner: j.Owner, Plist: j.Plist, Program: j.Program, Removal: j.Removal})
	}
	return out
}

// value: alias of station.status/v1's station block, byte for byte.
type stationJSON struct {
	Version string     `json:"version"`
	Commit  string     `json:"commit"`
	Engine  engineJSON `json:"daemon"`
}

type engineJSON struct {
	State   string `json:"state"`
	Running bool   `json:"running"`
	Version string `json:"version"`
}

type machineJSON struct {
	Name     *string `json:"name"`
	Kind     string  `json:"kind"`
	Hostname string  `json:"hostname"`
	OS       string  `json:"os"`
	Arch     string  `json:"arch"`
	User     string  `json:"user"`
	Shell    string  `json:"shell"`
}

type processJSON struct {
	PID   int     `json:"pid"`
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

type memoryJSON struct {
	TotalBytes int64         `json:"total_bytes"`
	Top        []processJSON `json:"top"`
}

type systemJSON struct {
	Uptime  string        `json:"uptime"`
	Battery string        `json:"battery"`
	CPU     []processJSON `json:"cpu"`
	Memory  memoryJSON    `json:"memory"`
}

type checkJSON struct {
	Name   string `json:"name"`
	State  string `json:"state"`
	Value  string `json:"value"`
	Detail string `json:"detail"`
}

type portJSON struct {
	PID  int    `json:"pid"`
	Name string `json:"name"`
	Port string `json:"port"`
}

type environmentJSON struct {
	Network  []checkJSON `json:"network"`
	Ports    []portJSON  `json:"ports"`
	Security []checkJSON `json:"security"`
	Updates  []checkJSON `json:"updates"`
}

type repositoryJSON struct {
	Path   string `json:"path"`
	Branch string `json:"branch"`
	Dirty  int    `json:"dirty"`
}

type dependencyJSON struct {
	Path     string `json:"path"`
	Manager  string `json:"manager"`
	Outdated int    `json:"outdated"`
}

type dockerJSON struct {
	State         string `json:"state"`
	Containers    int    `json:"containers"`
	Images        int    `json:"images"`
	Volumes       int    `json:"volumes"`
	DanglingBytes int64  `json:"dangling_bytes"`
	TotalBytes    int64  `json:"total_bytes"`
}

type diskJSON struct {
	Name  string `json:"name"`
	Level string `json:"level"`
	Bytes int64  `json:"bytes"`
}

// value: alias of station.status/v1's workspace block, byte for byte.
type workspaceJSON struct {
	// invariant: v1 keeps every key it ever had; the workspace is no longer
	// scanned, so repositories, dependencies and docker read as nothing seen.
	Repositories []repositoryJSON         `json:"repositories"`
	Dependencies []dependencyJSON         `json:"dependencies"`
	Docker       dockerJSON               `json:"docker"`
	Disk         []diskJSON               `json:"disk"`
	Misplaced    []pagedisk.MisplacedJSON `json:"strays"`
}

type stepJSON struct {
	Name   string `json:"name"`
	State  string `json:"state"`
	Detail string `json:"detail"`
}

type setupJSON struct {
	State string     `json:"state"`
	Steps []stepJSON `json:"steps"`
}

func SnapshotContract(s Overview) SnapshotJSON {
	// invariant: every array is built with make, so an empty one encodes as [], never null.
	body := SnapshotJSON{
		Schema:      schema,
		GeneratedAt: s.GeneratedAt.UTC().Format(time.RFC3339),
		Station:     stationJSON{Version: s.Station.Version, Commit: s.Station.Commit, Engine: engineOf(s.Station.Engine)},
		Machine: machineJSON{
			Kind: catalog.StatusV1Kind(s.Machine.Kind), Hostname: s.Machine.Hostname, OS: s.Machine.OS,
			Arch: s.Machine.Arch, User: s.Machine.User, Shell: s.Machine.Shell,
		},
		System: systemJSON{
			Uptime: s.System.Uptime, Battery: s.System.Battery,
			CPU:    processes(s.System.Reading.Top.CPU),
			Memory: memoryJSON{TotalBytes: s.System.MemoryTotal, Top: processes(s.System.Reading.Top.Memory)},
		},
		Environment: environmentJSON{
			Network:  checks(s.Environment.Network),
			Ports:    make([]portJSON, 0, len(s.Environment.Ports)),
			Security: checks(s.Environment.Protections),
			Updates:  checks(s.Environment.Updates),
		},
		Workspace: workspaceJSON{
			Repositories: []repositoryJSON{},
			Dependencies: []dependencyJSON{},
			Docker:       dockerJSON{State: string(catalog.StateUnknown)},
			Disk:         make([]diskJSON, 0, len(s.Workspace.Disk)),
			Misplaced:    pagedisk.MisplacedContract(s.Workspace.Misplaced),
		},
		Apps:       make([]pageapps.AppStatusJSON, 0, len(s.Apps)),
		Settings:   make([]pagesettings.ItemJSON, 0, len(s.Settings)),
		Setup:      setupJSON{State: string(s.Setup.State), Steps: make([]stepJSON, 0, len(s.Setup.Steps))},
		Identity:   checks(s.Identity),
		Background: jobs(s.Background),
		Gateways:   checks(s.Gateways),
		Tailnet:    pageports.TailnetObject(s.Tailnet),
	}
	if s.Machine.Registered {
		name := s.Machine.Name
		body.Machine.Name = &name
	}
	for _, p := range s.Environment.Ports {
		body.Environment.Ports = append(body.Environment.Ports, portJSON{PID: p.PID, Name: p.Name, Port: p.Port})
	}
	for _, d := range s.Workspace.Disk {
		body.Workspace.Disk = append(body.Workspace.Disk, diskJSON{Name: d.Name, Level: string(d.Level), Bytes: d.Bytes})
	}
	for _, t := range s.Apps {
		body.Apps = append(body.Apps, pageapps.AppObject(t))
	}
	for _, c := range s.Settings {
		body.Settings = append(body.Settings, pagesettings.ItemObject(c))
	}
	for _, step := range s.Setup.Steps {
		body.Setup.Steps = append(body.Setup.Steps, stepJSON{Name: step.Name, State: string(step.State), Detail: step.Detail})
	}
	if s.AwakeKnown {
		body.Power = &awakeJSON{Held: s.Awake.Held, Lid: s.Awake.Lid}
		if s.Awake.Held {
			body.Power.Until = s.Awake.Until.UTC().Format(time.RFC3339)
		}
	}
	return body
}

func checks(in []overview.Check) []checkJSON {
	out := make([]checkJSON, 0, len(in))
	for _, c := range in {
		out = append(out, checkJSON{Name: c.Name, State: string(c.State), Value: c.Value, Detail: c.Detail})
	}
	return out
}

func processes(in []activity.Process) []processJSON {
	out := make([]processJSON, 0, len(in))
	for _, p := range in {
		out = append(out, processJSON{PID: p.PID, Name: p.Name, Value: p.Value})
	}
	return out
}

func engineOf(c overview.Check) engineJSON {
	d := engineJSON{State: string(c.State), Running: c.State == catalog.StateOK || strings.HasPrefix(c.Value, "runs ")}
	if d.State == "" {
		d.State = string(catalog.StateAbsent)
	}
	if v, ok := strings.CutPrefix(c.Value, "running "); ok {
		d.Version = v
	} else if v, ok := strings.CutPrefix(c.Value, "runs "); ok {
		d.Version, _, _ = strings.Cut(v, ",")
	}
	return d
}
