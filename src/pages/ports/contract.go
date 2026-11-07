package ports

import (
	"strings"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/finding"
	"github.com/jterrazz/studio-station/src/ports"
)

type PortJSON struct {
	Machine string `json:"machine"`
	Port    int    `json:"port"`
	Proto   string `json:"proto"`
	Process string `json:"process"`
	What    string `json:"what,omitempty"`
	// value: the name it is known by, and the logo the menu bar draws.
	Name string `json:"name"`
	Logo string `json:"logo,omitempty"`
	// value: the app its process is, and its pid, for a port this machine
	// lists itself.
	App      string        `json:"app,omitempty"`
	PID      int           `json:"pid,omitempty"`
	Bind     []string      `json:"bind"`
	Class    string        `json:"class"`
	Declared *DeclaredJSON `json:"declared"`
	State    string        `json:"state,omitempty"`
	Reserved int           `json:"reserved,omitempty"`
	URL      string        `json:"url,omitempty"`
	Detail   string        `json:"detail,omitempty"`
	// value: unreachable while its VM is down, starting while it boots
	// again; absent while it answers.
	Answer string `json:"answer,omitempty"`
}

type DeclaredJSON struct {
	Via string `json:"via"`
	// value: the setting that declared it; absent when the owner did.
	By string `json:"by,omitempty"`
	// value: its route is seen; what is missing when it is not.
	Seen    bool   `json:"seen"`
	Missing string `json:"missing,omitempty"`
}

type RangeJSON struct {
	Mac      string `json:"mac"`
	From     int    `json:"from"`
	To       int    `json:"to"`
	Reserved int    `json:"reserved"`
	Free     int    `json:"free"`
}

type NodeJSON struct {
	State   string `json:"state"`
	Name    string `json:"name"`
	DNSName string `json:"dns_name"`
	IP      string `json:"ip"`
	Serve   string `json:"serve"`
	Variant string `json:"variant"`
	// value: the client station found, by its absolute path.
	CLI string `json:"cli"`
}

type BodyJSON struct {
	Schema    string         `json:"schema"`
	Findings  []finding.JSON `json:"findings"`
	Ports     []PortJSON     `json:"ports"`
	LocalOnly int            `json:"local_only"`
	Range     []RangeJSON    `json:"range"`
}

type ListJSON map[string]any // value: station.ports.list/v1, its node under the key of the tailnet's product, a word the catalogue owns.

// value: the words of a port's state in station.ports.list/v1.
func stateWord(s ports.SharedPort) string {
	switch {
	case s.State == ports.PhaseRefused || s.Reserved == 0:
		return "waiting"
	case s.State == ports.PhaseUnknown:
		return "engine-stopped"
	case s.State == ports.PhaseWaiting:
		return "not-listening"
	case s.Detail != "":
		return "local-only"
	}
	return "shared"
}

func PortObject(s ports.SharedPort) PortJSON {
	bind := []string{}
	if s.Bind != "" {
		bind = append(bind, s.Bind)
	}
	proto := "tcp"
	if s.HTTP {
		proto = "http"
	}
	known := s.Named()
	return PortJSON{Machine: s.VM, Port: s.GuestPort, Proto: proto, Process: s.Process, Name: known.Name, Logo: known.Logo, Bind: bind, Class: "private",
		State: stateWord(s), Reserved: s.Reserved, URL: s.URLs.Public, Detail: s.Detail}
}

func ListenerObject(p ports.Port) PortJSON {
	known := p.Named()
	o := PortJSON{Machine: p.Machine.Name, Port: p.Port, Proto: "tcp", Process: p.Process, Name: known.Name, Logo: known.Logo, PID: p.PID, Bind: p.Binds, Class: string(p.Class()), Answer: string(p.Answer)}
	if app := p.App(); app != p.Process {
		o.App = app
	}
	if w := p.What(); w != p.Process {
		o.What = w
	}
	if p.Declared != nil {
		o.Declared = &DeclaredJSON{Via: p.Declared.Via, By: p.Declared.By}
		if v := p.Verified; v != nil {
			o.Declared.Seen, o.Declared.Missing = v.Seen, v.Missing
		}
	}
	return o
}

func ListContract(l ports.Listing) ListJSON {
	// invariant: one object per port of a machine, red first, then public, then
	// private; a shared port is its share, told its class by what listens there.
	body := BodyJSON{Schema: "station.ports.list/v1", Ports: []PortJSON{}, Range: []RangeJSON{}}
	var red, public, private []PortJSON
	var all []ports.Port
	for _, h := range l.AllMachines.Macs {
		heard := map[string]ports.Port{}
		for _, p := range h.Ports {
			heard[p.Key()] = p
		}
		shared := map[string]bool{}
		for _, s := range h.Shared {
			o := PortObject(s)
			key := rowID(s.VM, s.GuestPort)
			shared[key] = true
			if p, ok := heard[key]; ok {
				l := ListenerObject(p)
				o.Bind, o.Class, o.Declared, o.What, o.PID, o.Answer = l.Bind, l.Class, l.Declared, l.What, l.PID, l.Answer
			}
			private = append(private, o)
		}
		for _, p := range h.Ports {
			all = append(all, p)
			switch {
			case shared[p.Key()]:
			case p.Red():
				red = append(red, ListenerObject(p))
			case p.Declared != nil:
				public = append(public, ListenerObject(p))
			case p.Reach == ports.ThisMachine:
				body.LocalOnly++
			default:
				private = append(private, ListenerObject(p))
			}
		}
		if r := h.Range; r != nil {
			body.Range = append(body.Range, RangeJSON{Mac: h.Name, From: r.Low, To: r.High, Reserved: r.Reserved, Free: r.Free})
		}
	}
	body.Ports = append(append(append(body.Ports, red...), public...), private...)
	body.Findings = finding.AllMachinesDoc(ports.Exposed(all))
	var node *NodeJSON
	if t := l.Tailnet; t.State != "" {
		state := "on"
		switch {
		case t.State == catalog.StateAbsent:
			state = "not-installed"
		case !t.Connected:
			state = "signed-out"
		case !t.ServeOn:
			state = "serve-off"
		}
		serve := "off"
		if t.ServeOn {
			serve = "on"
		}
		node = &NodeJSON{State: state, Name: t.Name, DNSName: t.DNSName, IP: t.IP, Serve: serve, Variant: t.Variant, CLI: t.CLI}
	}
	return ListJSON{"schema": body.Schema, "findings": body.Findings, "ports": body.Ports, "local_only": body.LocalOnly, "range": body.Range,
		strings.ToLower(catalog.TailnetProduct): node}
}
