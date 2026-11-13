package ports

import (
	"slices"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

// value: what a machine may open and to whom, as its Machinefile's ports
// say it; the zero Rights open nothing. Refused holds why the Machinefile
// that says them could not be read: it opens nothing either.

type Rights struct {
	catalog.PortRights
	Refused string
	// value: the machine is a server, whose reservations made before they
	// said who made them are the owner's.
	Server bool
}

func RightsOf(said *catalog.PortRights, server bool) Rights {
	// invariant: a Machinefile's own ports win; one that says none gives a
	// workspace catalog.WorkspaceRights, today's sharing, and a server none.
	switch {
	case said != nil:
		return Rights{PortRights: *said, Server: server}
	case server:
		return Rights{Server: true}
	}
	return Rights{PortRights: catalog.WorkspaceRights}
}

func (r Rights) Opens() bool { return len(r.Open) > 0 || r.Refused != "" }

func (r Rights) Allows(port int) bool {
	if r.Refused != "" || slices.Contains(r.Except, port) {
		return false
	}
	return slices.ContainsFunc(r.Open, func(s catalog.PortSpan) bool { return port >= s.From && port <= s.To })
}

func (r Rights) Audience() catalog.Audience { return r.To }

func (r Rights) Floor() int {
	// invariant: a port below catalog.SystemPorts is reported only when the
	// rights open one below it, and the VM's own ssh port never is: it
	// carries station itself.
	floor := catalog.SystemPorts
	for _, s := range r.Open {
		floor = min(floor, s.From)
	}
	return floor
}

func (r Rights) Reported(port, ssh int) bool { return port != ssh && port >= r.Floor() }

func (r Rights) PublicPort(port int, httpsMode string) int {
	// value: the Funnel port a `to: public` right's port is shared on, its
	// rank in open mapped onto FunnelPortsIn(httpsMode); 0 for any other,
	// and for one ranked past the Funnel ports the mode leaves.
	if r.To != catalog.AudiencePublic {
		return 0
	}
	funnel := FunnelPortsIn(httpsMode)
	for i, s := range r.Open {
		if s.From == port && i < len(funnel) {
			return funnel[i]
		}
	}
	return 0
}

func FunnelPortsIn(httpsMode string) []int {
	// invariant: HTTPS links by path mount every private share on 443, and
	// Funnel opens a port whole, every path of it: in path mode 443 is never
	// a public port, so a private path is never on the internet.
	if httpsMode == HTTPSPath {
		return slices.DeleteFunc(slices.Clone(catalog.FunnelPorts[:]), func(p int) bool { return p == 443 })
	}
	return slices.Clone(catalog.FunnelPorts[:])
}

func (r Rights) Judge(l Reservation, httpsMode string) (Reservation, bool) {
	// invariant: a reservation is judged by its VM's rights as they are
	// now: one the rights made that they no longer allow is not kept, nor a
	// public one no Funnel port is left for; one the owner made is kept,
	// private when the rights do not give it their audience. A kept one
	// takes the audience and Funnel port the rights give it today. One
	// made before reservations said who made them is the owner's on a
	// server, which only `ports share` shared then, and the rights' on a
	// workspace.
	if l.By == "" {
		l.By = ByRights
		if r.Server {
			l.By = ByOwner
		}
	}
	allowed := r.Allows(l.GuestPort)
	if !allowed && !l.Owned() {
		return l, false
	}
	l.To, l.PublicPort = "", 0
	if !allowed || r.Audience() == catalog.AudiencePrivate {
		return l, true
	}
	l.To = r.Audience()
	if l.To != catalog.AudiencePublic {
		return l, true
	}
	if l.PublicPort = r.PublicPort(l.GuestPort, httpsMode); l.PublicPort == 0 {
		l.To = ""
		return l, l.Owned()
	}
	return l, true
}

func (r Rights) Said() string {
	// value: the rights as a flagged port names them: `open: 3000-9999`,
	// then `; except: 3306` when they except any.
	if r.Refused != "" {
		return r.Refused
	}
	spans := make([]string, 0, len(r.Open))
	for _, s := range r.Open {
		spans = append(spans, s.String())
	}
	if len(spans) == 0 {
		return "opens nothing"
	}
	said := "open: " + strings.Join(spans, ", ")
	if len(r.Except) > 0 {
		except := make([]string, 0, len(r.Except))
		for _, e := range r.Except {
			except = append(except, strconv.Itoa(e))
		}
		said += "; except: " + strings.Join(except, ", ")
	}
	return said
}

func NotAllowed(r Rights) string {
	// value: why a port that listens outside its machine's rights is not
	// shared, as its row says it after its port.
	return "listens, not allowed (" + r.Said() + ")"
}

type FunnelTaken struct {
	Port int
	// value: the share that holds it, as <vm>:<port>.
	By string
}

func (e *FunnelTaken) Error() string {
	return "Funnel port " + strconv.Itoa(e.Port) + " is shared by " + e.By
}

type NoFunnelPort struct{ Port int }

func (e *NoFunnelPort) Error() string {
	// value: why a public port is not shared in path mode: 443 carries the
	// private paths, so two Funnel ports are left.
	return "no Funnel port is left for " + strconv.Itoa(e.Port) + ": HTTPS links by path keep 443 private, so public opens 2 ports at most"
}

type FunnelHeld struct {
	Port int
	// value: the share refused is private, a path or a port, which a
	// funneled port would put on the internet.
	Private bool
	// value: the port of this Mac the entry that holds it is carried from,
	// when it is of station's range: an entry station made, still there.
	From int
	// value: the entry that holds it is a private path station made, of
	// the HTTPS mode the Mac is leaving.
	Paths bool
}

func (e *FunnelHeld) Error() string {
	// value: why a share is not opened: its port serves or funnels an entry
	// station did not make, which station never replaces nor joins, or one
	// station made for another share that still holds it.
	switch {
	case e.Private:
		return "port " + strconv.Itoa(e.Port) + " is funneled by an entry station did not make: a private share on it would be public"
	case e.Paths:
		return "Funnel port " + strconv.Itoa(e.Port) + " still serves station's private paths: shared once they leave it"
	case e.From != 0:
		return "Funnel port " + strconv.Itoa(e.Port) + " still serves station's entry from port " + strconv.Itoa(e.From) + " of this Mac"
	}
	return "Funnel port " + strconv.Itoa(e.Port) + " already serves an entry station did not make"
}

func HintOf(hints []catalog.PackagePort, port int) (catalog.PackagePort, bool) {
	// value: the first item that serves on this port, the label a row of it
	// is shown with.
	for _, h := range hints {
		if h.Port == port {
			return h, true
		}
	}
	return catalog.PackagePort{}, false
}

func Hinted(v SharedPort, hints []catalog.PackagePort) SharedPort {
	// value: a shared port with what the catalog's items say of it: its
	// label, and the item whose env sharing it sets (`profile`, v1's word).
	for _, h := range hints {
		if h.Port != v.GuestPort {
			continue
		}
		if v.Label == "" {
			v.Label = h.Label
		}
		if v.Profile == "" && len(h.Env) > 0 {
			v.Profile = h.Package
		}
	}
	return v
}

func CarriedFrom(s Share) int {
	// value: the port of this Mac's loopback a share is carried from, as the
	// tailnet reports its target (`http://127.0.0.1:40001`); 0 when it names
	// none.
	_, port, ok := strings.Cut(strings.TrimPrefix(strings.TrimPrefix(s.Target, "http://"), "tcp://"), Loopback+":")
	if !ok {
		return 0
	}
	n, _ := strconv.Atoi(port)
	return n
}

func EnvOf(h catalog.PortHint, url, name string) map[string]string {
	// value: what sharing a port sets in its VM's environment, its item's
	// templates rendered: `{url}` its shared URL, `{host}` the name it is
	// shared at.
	env := map[string]string{}
	for key, value := range h.Env {
		env[key] = strings.NewReplacer("{url}", url, "{host}", name).Replace(value)
	}
	return env
}
