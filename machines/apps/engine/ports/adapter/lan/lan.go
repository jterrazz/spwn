package lan

import (
	"context"
	"strconv"

	"github.com/jterrazz/studio-station/apps/engine/catalog"

	"github.com/jterrazz/studio-station/apps/engine/ports"
	"github.com/jterrazz/studio-station/apps/engine/ports/contract"
)

const Name = catalog.RouteLAN

type Route struct{}

var _ contract.PublicRoute = Route{}

func (Route) Name() string { return Name }

func (Route) Verify(_ context.Context, d ports.Declaration, on ports.Machine) (ports.Verified, error) {
	// invariant: seen when the port listens where the machine's network
	// reaches it: every interface or a LAN address, never behind its Mac.
	for _, l := range on.Listening {
		if l.Port == d.Port && ports.ReachOf(l.Bind, on.At) == ports.Outside {
			return ports.Verified{Seen: true}, nil
		}
	}
	return ports.Verified{Missing: strconv.Itoa(d.Port) + " does not listen on the LAN of " + on.Ref.Name}, nil
}
