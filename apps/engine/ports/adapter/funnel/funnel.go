package funnel

import (
	"context"
	"slices"
	"strconv"

	"github.com/jterrazz/studio-station/apps/engine/catalog"

	"github.com/jterrazz/studio-station/apps/engine/ports"
	"github.com/jterrazz/studio-station/apps/engine/ports/contract"
)

const Name = catalog.RouteTailscaleFunnel

type Funnel interface {
	// value: the ports this Mac's Funnel config opens to the internet.
	Funneled(ctx context.Context) ([]int, error)
}

type Route struct{ funnel Funnel }

var _ contract.PublicRoute = Route{}

func New(c Funnel) Route { return Route{funnel: c} }

func (Route) Name() string { return Name }

func (r Route) Verify(ctx context.Context, d ports.Declaration, on ports.Machine) (ports.Verified, error) {
	// invariant: the Funnel config read is this Mac's own node's: a VM's
	// own device keeps its config where this Mac does not read it.
	missing := ports.Verified{Missing: "Funnel does not list " + strconv.Itoa(d.Port) + " on " + on.Ref.Name}
	if on.At != ports.OnMac {
		return missing, nil
	}
	open, err := r.funnel.Funneled(ctx)
	if err != nil {
		return ports.Verified{}, err
	}
	if slices.Contains(open, d.Port) {
		return ports.Verified{Seen: true}, nil
	}
	return missing, nil
}
