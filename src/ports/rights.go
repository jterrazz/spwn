package ports

import (
	"slices"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/src/catalog"
)

// value: what a machine may open and to whom, as its Machinefile's ports
// say it; the zero Rights open nothing. Refused holds why the Machinefile
// that says them could not be read: it opens nothing either.

type Rights struct {
	catalog.PortRights
	Refused string
}

func RightsOf(said *catalog.PortRights, server bool) Rights {
	// invariant: a Machinefile's own ports win; one that says none gives a
	// workspace catalog.WorkspaceRights, today's sharing, and a server none.
	switch {
	case said != nil:
		return Rights{PortRights: *said}
	case server:
		return Rights{}
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

func (r Rights) PublicPort(port int) int {
	// value: the Funnel port a `to: public` right's port is shared on, its
	// rank in open mapped onto catalog.FunnelPorts; 0 for any other.
	if r.To != catalog.AudiencePublic {
		return 0
	}
	for i, s := range r.Open {
		if s.From == port && i < len(catalog.FunnelPorts) {
			return catalog.FunnelPorts[i]
		}
	}
	return 0
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

func HintOf(hints []catalog.ItemPort, port int) (catalog.ItemPort, bool) {
	// value: the first item that serves on this port, the label a row of it
	// is shown with.
	for _, h := range hints {
		if h.Port == port {
			return h, true
		}
	}
	return catalog.ItemPort{}, false
}

func Hinted(v SharedPort, hints []catalog.ItemPort) SharedPort {
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
			v.Profile = h.Item
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
