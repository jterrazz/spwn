package machines

import (
	"time"

	"github.com/jterrazz/studio-station/src/core/finding"
)

const ListSchema = "station.machines.list/v1"

type ListDoc struct {
	Schema   string         `json:"schema"`
	Macs     []ListMac      `json:"macs"`
	Findings []finding.JSON `json:"findings"`
}

type ListMac struct {
	Name      string     `json:"name"`
	Label     string     `json:"label"`
	ThisMac   bool       `json:"this_mac"`
	Reachable bool       `json:"reachable"`
	Answered  time.Time  `json:"last_answer,omitzero"`
	IP        string     `json:"ip"`
	User      string     `json:"user"`
	Engine    ListEngine `json:"engine"`
	Station   string     `json:"station_version"`
	VMs       []ListVM   `json:"vms"`
}

type ListEngine struct {
	State   string `json:"state"`
	Version string `json:"version"`
}

type ListVM struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Lifetime string `json:"lifetime"`
	State    string `json:"state"`
	Wanted   string `json:"wanted"`
	// value: SPEC §7.2's steps a workspace has, each passed or not.
	Health   ListHealth   `json:"health"`
	Restarts ListRestarts `json:"restarts"`
	IP       string       `json:"ip"`
	// value: on its Mac's network; false is behind its Mac.
	Bridged bool      `json:"bridged"`
	Size    ListSize  `json:"size"`
	Image   ListImage `json:"image"`
	Detail  string    `json:"detail,omitempty"`
	Created time.Time `json:"created_at"`
	// value: a server's last 30 days (SPEC B5.2), its check's latency
	// (B5.4) and the step its restart runs (B5.11), each absent until known.
	Uptime  *ListUptime  `json:"uptime,omitempty"`
	Latency *ListLatency `json:"latency,omitempty"`
	Restart *ListRestart `json:"restart,omitempty"`
	// value: why a server is down, and its console's last lines then
	// (SPEC B5.5); absent while it is up.
	Cause     string   `json:"cause,omitempty"`
	LastLines []string `json:"last_lines,omitempty"`
}

type ListUptime struct {
	Since time.Time `json:"since"`
	// value: percent, two decimals.
	Availability float64         `json:"availability"`
	Days         []ListUptimeDay `json:"days"`
}

type ListUptimeDay struct {
	Date     string         `json:"date"`
	Mark     string         `json:"mark"`
	Down     []ListIncident `json:"down,omitempty"`
	Restarts int            `json:"restarts,omitempty"`
}

type ListIncident struct {
	From time.Time `json:"from"`
	// value: absent while it is still down.
	To    *time.Time `json:"to,omitempty"`
	Cause string     `json:"cause,omitempty"`
}

type ListLatency struct {
	NowMS    int64     `json:"now_ms"`
	P50MS    int64     `json:"p50_ms"`
	PeakMS   int64     `json:"peak_ms"`
	PeakAt   time.Time `json:"peak_at"`
	PointsMS []int64   `json:"points_ms"`
}

type ListRestart struct {
	Step    int       `json:"step"`
	Of      int       `json:"of"`
	Name    string    `json:"name"`
	Counted string    `json:"counted"`
	Since   time.Time `json:"since"`
}

func uptimeOf(vm VM, now time.Time) *ListUptime {
	view, ok := vm.Uptime.View(now)
	if !ok {
		return nil
	}
	out := &ListUptime{Since: view.Since.UTC(), Availability: view.Availability, Days: make([]ListUptimeDay, 0, len(view.Cells))}
	for _, c := range view.Cells {
		day := ListUptimeDay{Date: c.Date, Mark: string(c.Mark), Restarts: c.Restarts}
		for _, in := range c.Down {
			incident := ListIncident{From: in.From.UTC(), Cause: in.Cause}
			if !in.To.IsZero() {
				to := in.To.UTC()
				incident.To = &to
			}
			day.Down = append(day.Down, incident)
		}
		out.Days = append(out.Days, day)
	}
	return out
}

func latencyOf(vm VM) *ListLatency {
	l, ok := LatencyOf(vm.Latency)
	if !ok {
		return nil
	}
	out := &ListLatency{NowMS: l.Now.Milliseconds(), P50MS: l.P50.Milliseconds(), PeakMS: l.Peak.Milliseconds(), PeakAt: l.PeakAt.UTC(), PointsMS: make([]int64, 0, len(vm.Latency))}
	for _, p := range vm.Latency {
		out.PointsMS = append(out.PointsMS, p.Took.Milliseconds())
	}
	return out
}

func restartOf(vm VM) *ListRestart {
	if vm.Restart == nil {
		return nil
	}
	w := *vm.Restart
	return &ListRestart{Step: w.Step, Of: RestartSteps, Name: w.Name(), Counted: w.Counted(), Since: w.At.UTC()}
}

type ListHealth struct {
	Running bool `json:"running"`
	Address bool `json:"address"`
	SSH     bool `json:"ssh"`
}

type ListSize struct {
	CPUs        int   `json:"cpus"`
	MemoryBytes int64 `json:"memory_bytes"`
	DiskBytes   int64 `json:"disk_bytes"`
}

type ListImage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ListRestarts struct {
	Count   int        `json:"count"`
	LastAt  *time.Time `json:"last_at"`
	Looping bool       `json:"looping"`
	WaitS   int64      `json:"wait_s"`
}

func restartsOf(r Restarts) ListRestarts {
	out := ListRestarts{Count: r.Count, Looping: r.Looping, WaitS: int64(r.Wait.Seconds())}
	if !r.Last.IsZero() {
		last := r.Last.UTC()
		out.LastAt = &last
	}
	return out
}

func ToList(f AllMachines) ListDoc {
	out := ListDoc{Schema: ListSchema, Macs: make([]ListMac, 0, len(f.Macs)), Findings: finding.AllMachinesDoc(Findings(f))}
	for _, h := range f.Macs {
		m := ListMac{
			Name: h.Name, Label: Label(f, h), ThisMac: h.ThisMac, Reachable: h.Reachable, Answered: h.Answered, IP: h.Node.IP, User: h.User,
			Engine: ListEngine{State: engineState(h), Version: h.Engine.Version}, Station: h.StationVersion, VMs: []ListVM{},
		}
		for _, vm := range Ordered(h.VMs) {
			m.VMs = append(m.VMs, ListVM{
				Name: vm.Name, Kind: string(vm.Lifetime.Kind()), Lifetime: string(vm.Lifetime), State: string(vm.State),
				Wanted: Wanted(vm), Health: HealthOf(vm), Restarts: restartsOf(vm.Restarts), IP: vm.IP, Bridged: vm.Network == "bridged",
				Size:  ListSize{CPUs: vm.Resources.CPUs, MemoryBytes: vm.Resources.MemoryBytes, DiskBytes: vm.Resources.DiskBytes},
				Image: ListImage{Name: vm.Image, Version: vm.Version}, Detail: vm.Detail, Created: vm.CreatedAt,
				Uptime: uptimeOf(vm, f.Read), Latency: latencyOf(vm), Restart: restartOf(vm),
				Cause: vm.Cause, LastLines: vm.LastLines,
			})
		}
		out.Macs = append(out.Macs, m)
	}
	return out
}

func engineState(h MacView) string {
	switch {
	case !h.Reachable:
		return "unknown"
	case !h.Engine.Known:
		return "none"
	case h.Engine.Running:
		return "running"
	case !h.Engine.Installed && h.ThisMac:
		return "not-set-up"
	}
	return "stopped"
}

func Wanted(vm VM) string {
	if vm.Want == "running" {
		return "running"
	}
	return "stopped"
}

func HealthOf(vm VM) ListHealth {
	// value: a VM that does not run has passed no step.
	running := vm.State == Running
	return ListHealth{Running: running, Address: running && vm.IP != "", SSH: running && (vm.Guest == "connected" || vm.SSH)}
}
