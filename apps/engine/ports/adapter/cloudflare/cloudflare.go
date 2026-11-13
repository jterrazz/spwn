package cloudflare

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/catalog"

	"github.com/jterrazz/studio-station/apps/engine/ports"
	"github.com/jterrazz/studio-station/apps/engine/ports/contract"
)

const (
	Name      = catalog.RouteCloudflareTunnel
	connector = "cloudflared"
)

type Route struct{}

var _ contract.PublicRoute = Route{}

func (Route) Name() string { return Name }

func (Route) Verify(_ context.Context, _ ports.Declaration, on ports.Machine) (ports.Verified, error) {
	// outside: a cloudflared connector serves its metrics on a loopback port
	// (127.0.0.1:20241 and up) while it runs, so what listens on the machine,
	// read by its guest inside a VM, shows the connector.
	for _, l := range on.Listening {
		if l.Process == connector {
			return ports.Verified{Seen: true}, nil
		}
	}
	return ports.Verified{Missing: "no tunnel seen on " + on.Ref.Name}, nil
}
