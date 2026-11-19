package journalcmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	journalcalls "github.com/jterrazz/studio-station/apps/engine/journal/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagejournal "github.com/jterrazz/studio-station/apps/engine/pages/journal"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[struct{}]{
		Use:     "journal",
		Short:   "What was done on every Mac and its VMs, by day, kept thirty days",
		Example: "  station journal\n  station journal --on mac-mini\n  station journal --json",
		Group:   kit.Machines,
		Read:    func(*struct{}) bool { return true },
		Pages:   true,
		Run: func(ctx context.Context, _ *struct{}, io kit.IO) error {
			// invariant: the journal is read from This Mac, which asks each Mac;
			// --on narrows it to that Mac and its VMs.
			in := journalcalls.ReadIn{}
			if env.Far() {
				in.Mac = env.Target().Name
			}
			out, err := call.Invoke(ctx, env.Calls, env.Self, journalcalls.Read, in, nil)
			if err != nil {
				return err
			}
			return io.Page(pagejournal.Read(out, d.ThisMachine), out)
		},
	}, env)
}
