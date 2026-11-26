package wire

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/approvals"
	approvalsmacos "github.com/jterrazz/studio-station/apps/engine/approvals/adapter/macos"
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
