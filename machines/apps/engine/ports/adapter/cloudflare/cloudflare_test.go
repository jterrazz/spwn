package cloudflare_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/ports"
	"github.com/jterrazz/studio-station/apps/engine/ports/adapter/cloudflare"
	"github.com/jterrazz/studio-station/apps/engine/ports/contract/contracttest"
)

func TestTheCloudflareTunnelRouteKeepsTheContract(t *testing.T) {
	traefik := ports.Listener{Port: 443, Bind: "*", Process: "traefik"}
	contracttest.PublicRoute(t, func(*testing.T) contracttest.RouteScene {
		return contracttest.RouteScene{
			Route:    cloudflare.Route{},
			Declared: ports.Declaration{Machine: "infra", Port: 443, Via: cloudflare.Name},
			Seen:     ports.VMOf("infra", "mini", false, []ports.Listener{traefik, {Port: 20241, Bind: "127.0.0.1", Process: "cloudflared"}}),
			Unseen:   ports.VMOf("infra", "mini", false, []ports.Listener{traefik}),
		}
	})
}
