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
	// value: the Machinefile that says them (`base-ubuntu`,
	// `web.local`); empty when no Machinefile names the machine.
	By string
}

type VMRights struct {
	VM string `json:"vm"`
	// value: what the VM may share and with whom, as its Ports page says it
	// under its head.
	Line string `json:"line"`
	// value: the Machinefile that sets them, to open; empty when none does.
	By string `json:"by,omitempty"`
}

func (r Rights) Line() string {
	// value: the rights in the owner's words: `It may share ports 1024 to
	// 65535 with your devices`, its spans, what they except and their
	// audience; a machine that opens nothing shares only what the owner
	// shares.
	switch {
	case r.Refused != "":
		return "It shares nothing: " + r.Refused
	case !r.Opens():
		return "It shares nothing by itself; you share a port when you need it"
	}
	var parts []string
	if len(r.Open) > 0 {
		spans := make([]string, 0, len(r.Open))
		for _, s := range r.Open {
			if s.From == s.To {
				spans = append(spans, strconv.Itoa(s.From))
			} else {
				spans = append(spans, strconv.Itoa(s.From)+" to "+strconv.Itoa(s.To))
			}
		}
		part := portNoun(len(r.Open) == 1 && r.Open[0].From == r.Open[0].To) + joinAnd(spans)
		if len(r.Except) > 0 {
			except := make([]string, 0, len(r.Except))
			for _, e := range r.Except {
				except = append(except, strconv.Itoa(e))
			}
			part += " except " + joinAnd(except)
		}
		parts = append(parts, part+" "+audienceWords(r.To))
	}
	for _, a := range []catalog.Audience{catalog.AudienceLocal, catalog.AudiencePrivate, catalog.AudiencePublic} {
		var own []string
		for _, port := range r.Audienced() {
			if r.Audiences[port] == a {
				own = append(own, strconv.Itoa(port))
			}
		}
		if len(own) > 0 {
			parts = append(parts, portNoun(len(own) == 1)+joinAnd(own)+" "+audienceWords(a))
		}
	}
	return "It may share " + joinAnd(parts)
}

func portNoun(one bool) string {
	if one {
		return "port "
	}
	return "ports "
}

func audienceWords(a catalog.Audience) string {
	switch a {
	case catalog.AudienceLocal:
		return "with this Mac only"
	case catalog.AudiencePublic:
		return "with the internet"
	case catalog.AudiencePrivate:
	}
	return "with your devices"
}

func joinAnd(words []string) string {
	if len(words) < 2 {
		return strings.Join(words, "")
	}
	return strings.Join(words[:len(words)-1], ", ") + " and " + words[len(words)-1]
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

func (r Rights) Opens() bool { return len(r.Open) > 0 || len(r.Audiences) > 0 || r.Refused != "" }

func (r Rights) Allows(port int) bool {
	// invariant: a port audiences names is allowed whatever open and
	// except say: the most specific rule wins.
	if r.Refused != "" {
		return false
	}
	if _, own := r.Audiences[port]; own {
		return true
	}
	if slices.Contains(r.Except, port) {
		return false
	}
	return slices.ContainsFunc(r.Open, func(s catalog.PortSpan) bool { return port >= s.From && port <= s.To })
}

func (r Rights) AudienceOf(port int) catalog.Audience {
	// value: the audience the rights share a port with: its own in
	// audiences, else theirs.
	if a, own := r.Audiences[port]; own {
		return a
	}
	return r.To
}

func (r Rights) Floor() int {
	// invariant: a port below catalog.SystemPorts is reported only when the
	// rights open one below it, and the VM's own ssh port never is: it
	// carries station itself.
	floor := catalog.SystemPorts
	for _, s := range r.Open {
		floor = min(floor, s.From)
	}
	for port := range r.Audiences {
		floor = min(floor, port)
	}
	return floor
}

func (r Rights) Reported(port, ssh int) bool { return port != ssh && port >= r.Floor() }

func (r Rights) PublicPort(port int, httpsMode string) int {
	// value: the Funnel port a public port of the rights is shared on, its
	// rank among their public ports (catalog.PortRights.Public) mapped onto
	// FunnelPortsIn(httpsMode); 0 for any other, and for one ranked past
	// the Funnel ports the mode leaves.
	funnel := FunnelPortsIn(httpsMode)
	for i, p := range r.Public() {
		if p == port && i < len(funnel) {
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
	// workspace. One the owner made with an audience chosen keeps it, inside
	// the rights or outside them: a public one keeps its Funnel port, which
	// its engine judges free (Engine.chosenFunnelLocked).
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
	if l.Owned() && l.Chosen != "" {
		l.To = l.Chosen
		if l.To == catalog.AudiencePrivate {
			l.To = ""
		}
		if l.To != catalog.AudiencePublic {
			l.PublicPort = 0
		}
		return l, true
	}
	l.To, l.PublicPort = "", 0
	if !allowed || r.AudienceOf(l.GuestPort) == catalog.AudiencePrivate {
		return l, true
	}
	l.To = r.AudienceOf(l.GuestPort)
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
	if !r.Opens() {
		return "opens nothing"
	}
	var said []string
	if len(r.Open) > 0 {
		spans := make([]string, 0, len(r.Open))
		for _, s := range r.Open {
			spans = append(spans, s.String())
		}
		said = append(said, "open: "+strings.Join(spans, ", "))
	}
	if len(r.Except) > 0 {
		except := make([]string, 0, len(r.Except))
		for _, e := range r.Except {
			except = append(except, strconv.Itoa(e))
		}
		said = append(said, "except: "+strings.Join(except, ", "))
	}
	if len(r.Audiences) > 0 {
		own := make([]string, 0, len(r.Audiences))
		for _, port := range r.Audienced() {
			own = append(own, strconv.Itoa(port)+": "+string(r.Audiences[port]))
		}
		said = append(said, "audiences: "+strings.Join(own, ", "))
	}
	return strings.Join(said, "; ")
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

type FunnelsHeld struct {
	Port int
	// value: the Funnel ports the HTTPS mode offers, each held by another
	// share.
	Held []int
}

func (e *FunnelsHeld) Error() string {
	// value: why a port the owner shares public is not: every Funnel port
	// already carries another share.
	held := make([]string, 0, len(e.Held))
	for _, p := range e.Held {
		held = append(held, strconv.Itoa(p))
	}
	return "no Funnel port is left for " + strconv.Itoa(e.Port) + ": " + strings.Join(held, ", ") + " each carry another share"
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
		return "port " + strconv.Itoa(e.Port) + " is funneled by an entry Machines did not make: a private share on it would be public"
	case e.Paths:
		return "Funnel port " + strconv.Itoa(e.Port) + " still serves Machines' private paths: shared once they leave it"
	case e.From != 0:
		return "Funnel port " + strconv.Itoa(e.Port) + " still serves Machines' entry from port " + strconv.Itoa(e.From) + " of this Mac"
	}
	return "Funnel port " + strconv.Itoa(e.Port) + " already serves an entry Machines did not make"
}

func HintOf(hints []catalog.PackagePort, port int) (catalog.PackagePort, bool) {
	// value: the first package that serves on this port, the label a row of it
	// is shown with.
	for _, h := range hints {
		if h.Port == port {
			return h, true
		}
	}
	return catalog.PackagePort{}, false
}

func Hinted(v SharedPort, hints []catalog.PackagePort) SharedPort {
	// value: a shared port with what the catalog's packages say of it: its
	// label, and the package whose env sharing it sets (`profile`, v1's word).
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
	// value: what sharing a port sets in its VM's environment, its package's
	// templates rendered: `{url}` its shared URL, `{host}` the name it is
	// shared at.
	env := map[string]string{}
	for key, value := range h.Env {
		env[key] = strings.NewReplacer("{url}", url, "{host}", name).Replace(value)
	}
	return env
}
