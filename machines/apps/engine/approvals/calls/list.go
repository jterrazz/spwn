package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/approvals"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
)

type Deps struct {
	// value: This Mac's approvals as its engine holds them: inside the
	// engine, the view of the same name; elsewhere, the engine's answer.
	Listed func(ctx context.Context) (approvals.Listing, error)
}

var List = call.Call[Deps, struct{}, approvals.Listing]{
	// invariant: ADR-105: the approvals are the engine's to read, so the
	// call is answered from the engine's store, the view approvals.list,
	// and never reads macOS itself.
	Name: "approvals.list", Where: call.InEngine, Applies: approvals.OnAMac,
	Run: func(ctx context.Context, d Deps, _ struct{}, _ call.Progress) (approvals.Listing, error) {
		return d.Listed(ctx)
	},
}

func All(d Deps) []call.Handler {
	return []call.Handler{call.Bind(List, d)}
}
