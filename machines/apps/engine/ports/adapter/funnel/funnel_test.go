package funnel_test

import (
	"context"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/ports"
	"github.com/jterrazz/studio-station/apps/engine/ports/adapter/funnel"
	"github.com/jterrazz/studio-station/apps/engine/ports/contract/contracttest"
)

type config []int

func (c config) Funneled(context.Context) ([]int, error) { return c, nil }

func TestTheTailscaleFunnelRouteKeepsTheContract(t *testing.T) {
	caddy := []ports.Listener{{Port: 8443, Bind: "127.0.0.1", Process: "caddy"}}
	contracttest.PublicRoute(t, func(*testing.T) contracttest.RouteScene {
		return contracttest.RouteScene{
			Route:    funnel.New(config{443, 8443}),
			Declared: ports.Declaration{Machine: "mini", Port: 8443, Via: funnel.Name},
			Seen:     ports.MacOf("mini", caddy),
			Unseen:   ports.VMOf("infra", "mini", false, caddy),
		}
	})
}
