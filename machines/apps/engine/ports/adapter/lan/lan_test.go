package lan_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/ports"
	"github.com/jterrazz/studio-station/apps/engine/ports/adapter/lan"
	"github.com/jterrazz/studio-station/apps/engine/ports/contract/contracttest"
)

func TestTheLANRouteKeepsTheContract(t *testing.T) {
	contracttest.PublicRoute(t, func(*testing.T) contracttest.RouteScene {
		return contracttest.RouteScene{
			Route:    lan.Route{},
			Declared: ports.Declaration{Machine: "mini", Port: 5000, Via: lan.Name},
			Seen:     ports.MacOf("mini", []ports.Listener{{Port: 5000, Bind: "192.168.1.20", Process: "ControlCenter"}}),
			Unseen:   ports.VMOf("infra", "mini", false, []ports.Listener{{Port: 5000, Bind: "*", Process: "traefik"}}),
		}
	})
}
