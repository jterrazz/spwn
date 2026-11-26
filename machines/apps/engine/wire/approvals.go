package wire

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/approvals"
	approvalsmacos "github.com/jterrazz/studio-station/apps/engine/approvals/adapter/macos"
	approvalscalls "github.com/jterrazz/studio-station/apps/engine/approvals/calls"
	enginecalls "github.com/jterrazz/studio-station/apps/engine/engine/calls"
	"github.com/jterrazz/studio-station/apps/engine/engine/feeds"
)

// invariant: the engine's Local Network read is the approvals family's
// probe: the world's under one, macOS's on a Mac, none off it.
func (m machine) localNetwork() func(ctx context.Context) approvals.State {
	if m.world != nil {
		return m.world.localNetwork()
	}
	if m.sys.Env.GOOS() != "darwin" {
		return nil
	}
	return approvalsmacos.LocalNetwork{Sockets: m.sys.Sockets, GOOS: m.sys.Env.GOOS()}.Read
}

// invariant: approvals.list inside the engine is the view of its store;
// a door that is not the engine relays the engine's answer.
func (m machine) approvalsDeps() approvalscalls.Deps {
	views := enginecalls.Projected(feeds.Pages{ThisMachine: m.where})
	return approvalscalls.Deps{Listed: func(ctx context.Context) (approvals.Listing, error) {
		if v, served := enginecalls.Viewed(ctx, views, feeds.ViewApprovals); served {
			listed, _ := v.(approvals.Listing)
			return listed, nil
		}
		var out approvals.Listing
		err := m.engineAPI().do(ctx, approvalscalls.List.Name, struct{}{}, &out)
		return out, err
	}}
}
