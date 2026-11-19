package wire

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/engine"
	accountscmd "github.com/jterrazz/studio-station/apps/engine/surface/cli/accounts"
	accountslogin "github.com/jterrazz/studio-station/apps/engine/surface/cli/accounts/login"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

func accountsCommand(env kit.Env, m machine) *cobra.Command {
	return accountscmd.Command(accountscmd.Deps{ThisMachine: m.where}, env, accountslogin.Command(accountslogin.Deps{ThisMachine: m.where}, env))
}

// value: whether a VM of this Mac runs, as its engine says, asked through
// the engine's socket.
func (m machine) vmRunning(ctx context.Context, vm string) (bool, error) {
	live, err := m.macEngine().VMs(ctx)
	if err != nil {
		return false, err
	}
	return live[vm].State == engine.StateRunning, nil
}

// value: a command inside a VM as the engine runs it: the socket's, or
// the engine's own when it is the engine that runs it.
type localGuest interface {
	GuestExec(ctx context.Context, vm, line, stdin string) (engine.GuestRan, error)
}
