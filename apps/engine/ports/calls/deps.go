package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/ports"
	"github.com/jterrazz/studio-station/apps/engine/ports/contract"
)

type Deps struct {
	Ports   Ports
	Tailnet Node
	Kind    machine.Kind
	// value: this Mac's ports declared public, and the routes that verify them.
	Declarations contract.DeclarationStore
	Routes       contract.PublicRoutes
	Far          Far
	// value: this Mac's ports group, where its HTTPS links are kept.
	Prefs contract.PortsPrefs
}

type Far interface {
	// value: the same call, run by the Mac that keeps the machine's declarations.
	Declare(ctx context.Context, mac string, in DeclareIn) (DeclareOut, error)
	Undeclare(ctx context.Context, mac string, in UndeclareIn) (UndeclareOut, error)
}

type Node interface {
	// value: this node, its peers and what it shares, each share told
	// station's when the ports policy owns it.
	Status(ctx context.Context) (ports.Status, error)
}

type Ports interface {
	List(ctx context.Context, thisMacOnly bool) (ports.AllMachines, error)
	// value: the VM this binary runs in, inside one: its share and its stop
	// of sharing ride the socket its Mac's engine offers there.
	Placed() (vm string, inside bool)
	CheckPort(text string) (int, error)
	// value: a port number as given, whatever the sharing policy ignores.
	ParsePort(text string) (int, error)
	// value: the other Mac a VM runs on; empty when it is one of this Mac's.
	CheckVM(ctx context.Context, vm string) (mac string, err error)
	Share(ctx context.Context, vm string, port int) (ports.SharedPort, error)
	// value: the other Mac that holds the reservation; empty when this one does.
	CheckRelease(ctx context.Context, t ports.Target) (mac string, err error)
	Release(ctx context.Context, t ports.Target, mac string, yes bool) (ports.Reservation, error)
	// value: the machine's name (this Mac's when empty), and the other Mac
	// that keeps its declarations; empty when this one does.
	Keeper(ctx context.Context, machine string) (name, mac string, err error)
}

func All(d Deps) []call.Handler {
	return []call.Handler{call.Bind(List, d), call.Bind(Share, d), call.Bind(StopSharing, d), call.Bind(Declare, d), call.Bind(Undeclare, d), call.Bind(SetHTTPS, d)}
}
