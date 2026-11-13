package fake

import (
	"context"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/ports"
	"github.com/jterrazz/studio-station/apps/engine/ports/contract"
)

type Route struct{ name string }

var _ contract.PublicRoute = Route{}

func NewRoute(name string) Route { return Route{name: name} }

func (r Route) Name() string { return r.name }

func (r Route) Verify(_ context.Context, d ports.Declaration, on ports.Machine) (ports.Verified, error) {
	// invariant: a stand-in route sees the port wherever it listens.
	if slices.ContainsFunc(on.Listening, func(l ports.Listener) bool { return l.Port == d.Port }) {
		return ports.Verified{Seen: true}, nil
	}
	return ports.Verified{Missing: "nothing listens on " + on.Ref.Name}, nil
}
