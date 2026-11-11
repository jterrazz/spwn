package fake

import (
	"testing"

	"github.com/jterrazz/studio-station/src/ports"
	"github.com/jterrazz/studio-station/src/ports/contract"
	"github.com/jterrazz/studio-station/src/ports/contract/contracttest"
)

func signedIn() *Tailnet {
	host := "jterrazz-macbook-pro.tail1234.ts.net"
	return NewTailnet(ports.Self{
		Node:    ports.Node{Name: "jterrazz-macbook-pro", DNSName: host, IP: "100.98.133.14", Online: true},
		Sharing: ports.SharingState{Backend: ports.Running, On: true, DNSName: host},
	})
}

func TestTheFakeTailnetKeepsTheContract(t *testing.T) {
	contracttest.Tailnet(t, func(*testing.T) contract.Tailnet { return signedIn() })
}

func TestTheFakeAudienceKeepsTheContract(t *testing.T) {
	contracttest.Audience(t, func(*testing.T) (contract.Audience, contract.Tailnet) {
		tn := signedIn()
		return tn, tn
	})
}

func TestTheFakeListenersKeepTheContract(t *testing.T) {
	contracttest.Listeners(t, func(*testing.T) contract.Listeners { return NewListeners(contracttest.Scene()...) })
}

func TestTheFakeReservationStoreKeepsTheContract(t *testing.T) {
	contracttest.ReservationStore(t, func(*testing.T) contract.ReservationStore { return NewReservations() })
}

func TestTheFakeDeclarationStoreKeepsTheContract(t *testing.T) {
	contracttest.DeclarationStore(t, func(*testing.T) contract.DeclarationStore { return NewDeclarations() })
}

func TestTheFakeRouteKeepsTheContract(t *testing.T) {
	contracttest.PublicRoute(t, func(*testing.T) contracttest.RouteScene {
		return contracttest.RouteScene{
			Route:    NewRoute("stand-in"),
			Declared: ports.Declaration{Machine: "mini", Port: 5000, Via: "stand-in"},
			Seen:     ports.MacOf("mini", []ports.Listener{{Port: 5000, Bind: "*", Process: "node"}}),
			Unseen:   ports.MacOf("mini", nil),
		}
	})
}
