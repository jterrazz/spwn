package diskcmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/disk"
	diskcalls "github.com/jterrazz/studio-station/apps/engine/disk/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagedisk "github.com/jterrazz/studio-station/apps/engine/pages/disk"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	// value: the apps with a Disk row here, for completion.
	Names       func() []string
	ThisMachine page.Context
}

type Input struct {
	All bool
}

func Command(d Deps, env kit.Env, children ...*cobra.Command) *cobra.Command {
	cmd := kit.Command(kit.Leaf[Input]{
		Use:     "disk",
		Short:   "What each app keeps on this machine: misplaced, can rebuild, your data, never touched",
		Example: "  station disk\n  station disk --all\n  station disk --json",
		Group:   kit.OnAMachine,
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.BoolVar(&in.All, "all", false, "unfold every group, every row")
		},
		Read:  func(*Input) bool { return true },
		Live:  true,
		Pages: true,
		Entry: kit.Entry("station disk free", "disk row", d.Names),
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			thisMac := d.ThisMachine
			if env.Far() {
				thisMac = page.Context{Machine: env.Target().Name}
			}
			if io.CanLive() {
				updates, err := call.Follow(ctx, env.Calls, env.Target(), diskcalls.Read, diskcalls.ReadIn{}, nil)
				if err != nil {
					return err
				}
				if err := io.LivePage(kit.Updates(updates, func(r disk.Reading) page.Page {
					return pagedisk.Read(r, thisMac, in.All)
				})); err != nil {
					return err
				}
				return ctx.Err()
			}
			reading, err := call.Read(ctx, env.Calls, env.Target(), diskcalls.Read, diskcalls.ReadIn{}, nil)
			if err != nil {
				return err
			}
			return io.Page(pagedisk.Read(reading, thisMac, in.All), pagedisk.ReadContract(reading))
		},
	}, env)
	cmd.AddCommand(children...)
	return cmd
}
