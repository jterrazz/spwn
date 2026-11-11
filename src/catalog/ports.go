package catalog

import (
	"strconv"
	"time"
)

type PortDefaults struct {
	RangeLow, RangeHigh, MaxPerVM int
	HTTPS                         string
	Idle, Grace                   time.Duration
}

var PortPolicy = PortDefaults{
	RangeLow: 40000, RangeHigh: 40999, MaxPerVM: 20, HTTPS: "port", Idle: 7 * 24 * time.Hour, Grace: 5 * time.Second,
}

// value: the lowest port that is not the system's: a VM's guest reports a
// port below it only when the VM's rights open one there, so a VM's own
// daemons (DNS on 53) are never flagged.

const SystemPorts = 1024

// value: the port a VM's sshd answers on, which carries station itself:
// never reported, never shared.

const GuestSSH = 22

// value: the right of a workspace whose Machinefile says no ports: every
// port from SystemPorts up, shared with the owner's devices. A server has
// none: its own tooling chooses its routes.

var WorkspaceRights = PortRights{Open: PortSet{{From: SystemPorts, To: 65535}}, To: AudiencePrivate}

// value: the ports the internet reaches a machine on through Tailscale
// Funnel, in the order a `to: public` right's ports are mapped onto them
// (https://tailscale.com/kb/1223/funnel).

var FunnelPorts = [PublicPorts]int{443, 8443, 10000}

// value: the most rows a list of ports shows: the rest is one line, "N more".

const PortRowsShown = 40

func MoreRows(n int) string { return strconv.Itoa(n) + " more" }

// value: the one line that tells a VM's listeners no reservation holds and
// a refusal kept from sharing.

func NotSharedWords(n int) string {
	if n == 1 {
		return "1 port not shared"
	}
	return strconv.Itoa(n) + " ports not shared"
}

const TailnetProduct = "Tailscale" // value: the product that carries a share, as the owner reads its name.

const ServeRow = "Tailscale Serve" // value: the row that says whether the tailnet serves this host's VMs.

var SystemListeners = []string{"rapportd", "ControlCenter", "IPNExtension", "mDNSResponder", "launchd", "systemd"} // value: the processes whose listening port is the system's own: noise in a developer's list.

func StoppedDetail(vm string) string {
	// value: why a reserved port of a VM that does not run is not shared.
	return "not shared while " + vm + " is stopped"
}

func UnreachableDetail(vm string) string {
	// value: why a port its VM was heard listening on does not answer.
	return "unreachable while " + vm + " is down"
}

func StartingDetail(vm string) string {
	// value: a port of a VM booting again, before its guest reports.
	return "starting: unreachable until " + vm + " is ready"
}

const IssuingDetail = "issuing certificate" // value: a share whose host certificate the tailnet has not issued yet.

const ServeOffDetail = "Serve is off on Tailscale: turn on HTTPS certificates in the Tailscale admin console"

const SignedOutDetail = TailnetProduct + " is signed out" // value: why nothing is shared on a Mac signed out of the tailnet.

const SignedOutLocalOnly = SignedOutDetail + " on This Mac: ports are local only"

// value: the variable every VM whose shared ports set an env is given:
// the name its ports are shared at.

const PublicNameEnv = "STATION_PUBLIC_HOST"

// value: the two Tailscale app variants, measured 2026-11-01: the Standalone
// app's pkg installs the first CLI path, the App Store app only its bundle,
// whose executable answers when run directly (a symlink to it crashes).
const (
	TailscaleApp           = "/Applications/Tailscale.app"
	TailscaleReceipt       = TailscaleApp + "/Contents/_MASReceipt"
	TailscaleBundleCLI     = TailscaleApp + "/Contents/MacOS/Tailscale"
	TailscaleStandaloneCLI = "/usr/local/bin/tailscale"
)

var TailscaleCLIs = []string{TailscaleStandaloneCLI, TailscaleBundleCLI}

// value: the info button of the Private card and what it says (SPEC A4 pop.info).
const (
	AboutPrivateLinks = "About private links"
	PrivateLinksAre   = "Private links open only on devices signed in to your tailnet; Public routes are a Cloudflare Tunnel or a Funnel you declared."
)

const ReachableOutside = "reachable outside " + TailnetProduct // value: the words of a port anything outside the tailnet reaches.

type KnownPort struct {
	Port    int
	Process string
	What    string
	// value: where the owner turns it off, a System Settings path.
	Off  string
	Logo string
}

const airPlayOff = "System Settings > General > AirDrop & Handoff > AirPlay Receiver"

const airPlayReceiver = "AirPlay Receiver"

var KnownPorts = []KnownPort{ // value: the ports whose name the owner knows better than its process's; one with no process is known on any.
	{Port: 5000, Process: "ControlCenter", What: airPlayReceiver, Off: airPlayOff},
	{Port: 7000, Process: "ControlCenter", What: airPlayReceiver, Off: airPlayOff},
	{Port: 22, What: "SSH"},
	{Port: 5900, What: "Screen Sharing"},
	{Port: 6443, What: "Kubernetes API", Logo: "kubernetes"},
}

type PortName struct {
	Port int
	Name string
	// value: the logo a row of this port wears in the menu bar; none wears
	// a quiet chip.
	Logo string
}

var PortNames = []PortName{ // value: the software a developer knows a listener by (SPEC B5.17), whatever process runs it; a port KnownPorts names is not repeated here.
	{Port: 5173, Name: "Vite", Logo: "vite"},
	{Port: 4321, Name: "Astro", Logo: "astro"},
	{Port: 5432, Name: "Postgres", Logo: "postgres"},
	{Port: 443, Name: "Traefik", Logo: "traefik"},
	{Port: 9229, Name: "Node inspector", Logo: "node"},
	{Port: 6006, Name: "Storybook", Logo: "storybook"},
	{Port: 8081, Name: "Expo", Logo: "expo"},
}

func PortNameOf(port int, process string) PortName {
	// invariant: a listener is known by the name KnownPorts gives it, else
	// by the software of its port, else by its number ("Port 8080").
	if k, ok := KnownPortOf(port, process); ok {
		return PortName{Port: port, Name: k.What, Logo: k.Logo}
	}
	for _, s := range PortNames {
		if s.Port == port {
			return s
		}
	}
	return PortName{Port: port, Name: "Port " + strconv.Itoa(port)}
}

var AppNames = map[string]string{ // value: the app the owner knows a system process by, as System Settings names it.
	"ControlCenter": airPlayReceiver,
}

func AppOf(process string) string {
	// value: the app a process is, as the owner reads it; the process's own
	// name when the catalogue knows no better.
	if name, ok := AppNames[process]; ok {
		return name
	}
	return process
}

func KnownPortOf(port int, process string) (KnownPort, bool) {
	for _, k := range KnownPorts {
		if k.Port == port && (k.Process == "" || k.Process == process) {
			return k, true
		}
	}
	return KnownPort{}, false
}
