package diskfree

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/disk"
	diskcalls "github.com/jterrazz/studio-station/apps/engine/disk/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagedisk "github.com/jterrazz/studio-station/apps/engine/pages/disk"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	Names       func() []string
	ThisMachine page.Context
}

type Input struct{ Apps []string }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:      "free [<app...>]",
		Short:    "Empty what the named apps can rebuild, or every Can rebuild row, with each app's own command; nothing is lost",
		Example:  "  station disk free\n  station disk free go xcode",
		Args:     cobra.ArbitraryArgs,
		Bind:     func(in *Input, args []string) error { in.Apps = args; return nil },
		Complete: func([]string, string) []string { return d.Names() },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			outcomes, err := call.Invoke(ctx, env.Calls, env.Target(), diskcalls.Free, diskcalls.FreeIn{Apps: in.Apps}, nil)
			if err != nil {
				return err
			}
			if err := io.Page(pagedisk.Freed(outcomes, d.ThisMachine), nil); err != nil {
				return err
			}
			failed, first := disk.Failures(outcomes)
			if failed == 0 {
				return nil
			}
			return kit.PartialFailure{Failed: failed, Total: len(outcomes), Verb: "could not be freed", FirstHint: "re-run it: station disk free " + first.Name}
		},
	}, env)
}
