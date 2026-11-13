package machinescmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagemachines "github.com/jterrazz/studio-station/apps/engine/pages/machines"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
	// value: one machine's Overview, read on the machine env targets.
	Overview func(ctx context.Context, io kit.IO) error
	Names    func() []string
}

type Input struct {
	ThisMacOnly bool
	Machine     string
}

func Command(d Deps, env kit.Env, children ...*cobra.Command) *cobra.Command {
	cmd := kit.Command(kit.Leaf[Input]{
		Use:     "machines [<machine>]",
		Short:   "All machines: every Mac, its VMs under it; one machine's Overview",
		Example: "  station machines\n  station machines linux-1\n  station machines --json",
		Args: func(c *cobra.Command, args []string) error {
			if len(args) > 1 {
				return fmt.Errorf("unknown command %q for %q", args[0], c.CommandPath())
			}
			return nil
		},
		Bind: func(in *Input, args []string) error {
			if len(args) == 1 {
				in.Machine = args[0]
			}
			return nil
		},
		Complete: func([]string, string) []string { return d.Names() },
		Group:    kit.Machines,
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.BoolVar(&in.ThisMacOnly, "this-mac", false, "This Mac only, without asking the other Macs")
		},
		Read:  func(*Input) bool { return true },
		Live:  true,
		Pages: true,
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			if in.Machine != "" {
				if env.On != nil {
					env.On.Machine = in.Machine
				}
				return d.Overview(ctx, io)
			}
			listed := machinescalls.ListIn{ThisMac: in.ThisMacOnly}
			if !io.CanLive() {
				out, err := call.Read(ctx, env.Calls, env.Target(), machinescalls.List, listed, nil)
				if err != nil {
					return err
				}
				return io.Page(listPage(out, d.ThisMachine, env.Far()), machines.ToList(out.All))
			}
			updates, err := call.Follow(ctx, env.Calls, env.Target(), machinescalls.List, listed, nil)
			if err != nil {
				return err
			}
			var failed error
			pages := make(chan page.Page)
			go func() {
				defer close(pages)
				for u := range updates {
					if u.Err != nil {
						failed = u.Err
						continue
					}
					pages <- listPage(u.Value, d.ThisMachine, env.Far())
				}
			}()
			if err := io.LivePage(pages); err != nil {
				return err
			}
			for range pages {
			}
			if failed != nil {
				return failed
			}
			return ctx.Err()
		},
	}, env)
	cmd.AddCommand(children...)
	return cmd
}

func listPage(out machinescalls.ListOut, thisMac page.Context, far bool) page.Page {
	if far {
		out.All = machines.Elsewhere(out.All)
	}
	var warnings []string
	if out.Unrefreshed != nil {
		warnings = append(warnings, kit.SSHConfigNotRefreshed(out.Unrefreshed.Error()))
	}
	return pagemachines.List(out.All, warnings, thisMac)
}
