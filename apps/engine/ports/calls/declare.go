package calls

import (
	"context"
	"errors"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

type DeclareIn struct {
	// value: the machine whose port it is; empty is this Mac.
	Machine string
	Port    string
	Via     string
	// value: the link it is reached by from outside; empty when it has none.
	URL string
	// value: the setting that declares it; empty when the owner does.
	By string
}

type DeclareOut struct {
	Declaration ports.Declaration
}

var Declare = call.Call[Deps, DeclareIn, DeclareOut]{
	// invariant: a declaration names its route before any effect, and is
	// kept by the Mac the machine is on, whose list verifies it.
	Name: "ports.declare", Section: section.Ports, Where: call.AtTarget, Applies: ports.DeclaredOnAMac,
	Run: func(ctx context.Context, d Deps, in DeclareIn, _ call.Progress) (DeclareOut, error) {
		port, err := d.Ports.ParsePort(in.Port)
		if err != nil {
			return DeclareOut{}, err
		}
		switch _, known := d.Routes.Named(in.Via); {
		case in.Via == "":
			return DeclareOut{}, ports.NeedsRoute(port, d.Routes.Names())
		case !known:
			return DeclareOut{}, ports.UnknownRoute(in.Via, d.Routes.Names())
		}
		name, mac, err := d.Ports.Keeper(ctx, in.Machine)
		if err != nil {
			return DeclareOut{}, err
		}
		if mac != "" {
			in.Machine = name
			return d.Far.Declare(ctx, mac, in)
		}
		decl := ports.Declaration{Machine: name, Port: port, Via: in.Via, URL: in.URL, By: in.By}
		return DeclareOut{Declaration: decl}, d.Declarations.Put(ctx, decl)
	},
}

type UndeclareIn struct {
	Machine string
	Port    string
	Confirm string
}

type UndeclareOut struct {
	Declaration ports.Declaration
}

var Undeclare = call.Call[Deps, UndeclareIn, UndeclareOut]{
	// invariant: an undeclare asks yes or no, then forgets the declaration
	// on the Mac that keeps it; the port is red again while it still listens
	// where something outside the tailnet reaches it.
	Name: "ports.undeclare", Section: section.Ports, Where: call.AtTarget, Applies: ports.DeclaredOnAMac, Confirm: confirm.Undeclare,
	Errors: []fail.Code{confirm.YesNeeded},
	Run: func(ctx context.Context, d Deps, in UndeclareIn, _ call.Progress) (UndeclareOut, error) {
		port, err := d.Ports.ParsePort(in.Port)
		if err != nil {
			return UndeclareOut{}, err
		}
		name, mac, err := d.Ports.Keeper(ctx, in.Machine)
		if err != nil {
			return UndeclareOut{}, err
		}
		if mac != "" {
			in.Machine = name
			return d.Far.Undeclare(ctx, mac, in)
		}
		decl, err := d.Declarations.Get(ctx, name, port)
		if errors.Is(err, store.ErrNotFound) {
			return UndeclareOut{}, ports.NotDeclared(name, port)
		}
		if err != nil {
			return UndeclareOut{}, err
		}
		if err := confirm.Require(confirm.For(confirm.Undeclare, d.Kind), strings.TrimPrefix(decl.Key(), ":"), in.Confirm); err != nil {
			return UndeclareOut{}, err
		}
		return UndeclareOut{Declaration: decl}, d.Declarations.Delete(ctx, name, port)
	},
}
