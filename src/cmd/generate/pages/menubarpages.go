package pages

import (
	"time"

	"github.com/jterrazz/studio-station/src/activity"
	"github.com/jterrazz/studio-station/src/awake"
	"github.com/jterrazz/studio-station/src/catalog"
	coremachine "github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/page"
	pageactivity "github.com/jterrazz/studio-station/src/pages/activity"
	pageawake "github.com/jterrazz/studio-station/src/pages/awake"
	pagemenubar "github.com/jterrazz/studio-station/src/pages/menubar"
	"github.com/jterrazz/studio-station/src/ports"
)

// value: mac-mini's ports as vision panel g draws them: two reachable
// outside Tailscale undeclared, infra's 443 public through its tunnel, the
// mini's Screen Sharing and infra's API private.
func miniPorts() ports.Listing {
	mini, infra := coremachine.Ref{Name: "mac-mini"}, coremachine.Ref{Name: "infra", On: "mac-mini"}
	mac := ports.MacPorts{Name: "mac-mini", Reachable: true, Engine: ports.EngineInfo{Running: true}, Shared: []ports.SharedPort{},
		Ports: []ports.Port{
			{Machine: mini, Port: 5000, Process: "ControlCenter", Binds: []string{"*"}, Reach: ports.Outside},
			{Machine: infra, Port: 10250, Process: "kubelet", Binds: []string{"0.0.0.0"}, Reach: ports.Outside},
			{Machine: infra, Port: 443, Process: "nginx", Binds: []string{"0.0.0.0"}, Reach: ports.Outside,
				Declared: &ports.Declaration{Machine: infra.Name, Port: 443, Via: catalog.RouteCloudflareTunnel, URL: "https://jterrazz.com"}, Verified: &ports.Verified{Seen: true}},
			{Machine: mini, Port: 5900, Process: "screensharingd", Binds: []string{"100.98.133.20"}, Reach: ports.Tailnet},
			{Machine: infra, Port: 6443, Process: "kube-apiserver", Binds: []string{"0.0.0.0"}, Reach: ports.Tailnet, Device: true},
		}}
	return ports.Listing{AllMachines: ports.AllMachines{Schema: ports.AllMachinesSchema, Macs: []ports.MacPorts{mac}}}
}

// value: the ports of This Mac's VMs, as linux-1's Ports reads them: Vite
// shared with its link, a TCP port shared, as linux-1's page lists them.
func linuxPorts() ports.Listing {
	mac := twoMacsPorts().Macs[0]
	mac.Ports = nil
	return ports.Listing{AllMachines: ports.AllMachines{Schema: ports.AllMachinesSchema, Macs: []ports.MacPorts{mac}}}
}

func twoMacsAwake() pageawake.AllMachines {
	f := awakeAllMachines(awake.EngineRunning, true)
	f.Reports[0].Session = thisMacAwake().Session
	f.Label = func(mac string) string { return page.MacName(mac, "jterrazz-macbook-pro") }
	return f
}

// value: This Mac's last hour, a point a minute: busy at the half hour,
// memory climbing, a download at the end.
func hourOfActivity() pageactivity.View {
	v := thisMacActivity()
	last := v.Report.Last.Sample
	points := make([]activity.Sample, 0, 61)
	var busy, total, read, written, in, out uint64
	for i := range 61 {
		load := 20 + uint64(30*(i%20))/20
		if i >= 28 && i <= 34 {
			load = 72
		}
		busy, total = busy+load*600, total+60000
		read, written = read+uint64(40e6+float64(i%7)*20e6), written+uint64(10e6+float64(i%5)*8e6)
		in, out = in+uint64(30e6+float64(i%9)*12e6), out+uint64(3e6+float64(i%4)*2e6)
		if i >= 52 {
			in += 400e6
		}
		points = append(points, activity.Sample{
			At: last.At.Add(time.Duration(i-60) * time.Minute), CPUs: last.CPUs, CPUBusy: busy, CPUTotal: total,
			MemUsed: last.MemUsed - int64(60-i)*20e6, MemTotal: last.MemTotal, SwapUsed: last.SwapUsed,
			DiskRead: read, DiskWritten: written, NetIn: in, NetOut: out, DiskFree: last.DiskFree, DiskTotal: last.DiskTotal,
		})
	}
	v.Report.Graph = points
	v.Graph = time.Hour
	return v
}

// value: the menu bar's own pages of SPEC §10.2 in the two-macs world,
// named menubar.<page>.<world>.
func menubarOwnPages() []Fixture {
	thisMac := menubarHere()
	return []Fixture{
		{Name: "menubar.ports.mini", Page: pagemenubar.Ports(miniPorts(), pagemenubar.PortsOf{Machine: "mac-mini"}, thisMac)},
		{Name: "menubar.ports.linux-1", Page: pagemenubar.Ports(linuxPorts(), pagemenubar.PortsOf{Machine: "linux-1"}, thisMac)},
		{Name: "menubar.awake.two-macs", Page: pagemenubar.Awake(twoMacsAwake(), thisMac)},
		{Name: "menubar.activity.mac", Page: pagemenubar.Activity(hourOfActivity(), thisMac)},
	}
}
