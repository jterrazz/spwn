package wire

import (
	"context"

	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/ports"
	"github.com/jterrazz/studio-station/src/ports/adapter/cloudflare"
	"github.com/jterrazz/studio-station/src/ports/adapter/funnel"
	"github.com/jterrazz/studio-station/src/ports/adapter/lan"
	"github.com/jterrazz/studio-station/src/ports/adapter/tailscale"
	"github.com/jterrazz/studio-station/src/ports/contract"
)

func (m machine) exposure(ctx context.Context) ports.Exposure {
	ready, err := m.tailnet().Ready(ctx)
	if err != nil {
		return ports.ExposureOf(false, "")
	}
	return ports.ExposureOf(ready.On, ready.DNSName)
}

// value: this Mac's node on the tailnet, and the Audience its shares are
// opened to: one adapter, tailscale's, or the world's.
type sharingNode interface {
	contract.Tailnet
	contract.Audience
	funnel.Funnel
}

func (m machine) tailnet() sharingNode {
	if m.world != nil {
		return m.world.tailnet()
	}
	return m.tailnetCLI()
}

func (m machine) routes() contract.PublicRoutes {
	var funneled funnel.Funnel = m.tailnetCLI()
	if m.world != nil {
		funneled = m.world.funnel()
	}
	return contract.PublicRoutes{cloudflare.Route{}, funnel.New(funneled), lan.Route{}}
}

func (m machine) tailnetCLI() *tailscale.CLI {
	return tailscale.New(m.sys.Run, m.sys.FS, m.systemRoot())
}

// value: the tailnet as a page reads it, this node and what it shares in
// one pass, each share told station's when the ports policy owns it.
type tailnetStatus struct{ m machine }

func (m machine) tailnetStatus() tailnetStatus { return tailnetStatus{m: m} }

func (t tailnetStatus) Status(ctx context.Context) (ports.Status, error) {
	g, _ := t.m.stores.ports().Get()
	p := ports.PolicyOf(g)
	owns := func(s ports.Share) bool {
		return p.InRange(s.Port) || (p.HTTPS == ports.HTTPSPath && s.Port == 443 && s.Path != "/") || (s.Public() && p.InRange(ports.CarriedFrom(s)))
	}
	tn := t.m.tailnet()
	self, err := tn.Self(ctx)
	var shared []ports.Share
	if err == nil {
		shared, _ = tn.Shared(ctx)
	}
	return ports.StatusOf(self, err, shared, owns), nil
}

type macNodes struct{ tailnet contract.Tailnet }

func (t macNodes) Peers(ctx context.Context) (map[string]model.Node, error) {
	self, err := t.tailnet.Self(ctx)
	if err != nil {
		return nil, err
	}
	roster := self.Peers
	if self.Name != "" {
		roster = append([]ports.Node{self.Node}, roster...)
	}
	nodes := make(map[string]model.Node, len(roster))
	for _, p := range roster {
		if held, ok := nodes[p.Name]; ok && held.Online {
			continue
		}
		nodes[p.Name] = model.Node{Online: p.Online, IP: p.IP}
	}
	return nodes, nil
}
