package appscmd

import (
	"context"
	"errors"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/src/apps"
	appscalls "github.com/jterrazz/studio-station/src/apps/calls"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/page"
	pageapps "github.com/jterrazz/studio-station/src/pages/apps"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type Deps struct {
	// value: the catalogue apps placed on this machine, for completion.
	Names       func() []string
	ThisMachine page.Context
	// value: the catalog's apps, those of other machines among them.
	Apps []catalog.App
}

func Command(d Deps, env kit.Env, children ...*cobra.Command) *cobra.Command {
	cmd := kit.Command(kit.Leaf[Input]{
		Use:     "apps [<app>]",
		Short:   "The catalogue's apps on this machine, agent skills included; one app's detail",
		Example: "  station apps\n  station apps --all\n  station apps gh\n  station apps --json",
		Group:   kit.OnAMachine,
		Args: func(c *cobra.Command, args []string) error {
			if len(args) > 1 {
				return errors.New("unknown command \"" + args[0] + "\" for \"" + c.CommandPath() + "\"")
			}
			return nil
		},
		Bind: bind,
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.BoolVar(&in.All, "all", false, "unfold every group, the apps of other machines too")
		},
		Read:     func(*Input) bool { return true },
		Live:     true,
		Pages:    true,
		Complete: func([]string, string) []string { return d.Names() },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			if in.Name != "" {
				out, err := call.Invoke(ctx, env.Calls, env.Target(), appscalls.Show, appscalls.ShowIn{Name: in.Name}, nil)
				if err != nil {
					return err
				}
				return io.Page(pageapps.Show(out.Report, out.ThisMachine, d.ThisMachine), pageapps.ListContract(call.Schema(appscalls.Show.Name), []apps.Report{out.Report}, d.Apps))
			}
			thisMac := d.ThisMachine
			if env.Far() {
				thisMac = page.Context{Machine: env.Target().Name}
			}
			if io.CanLive() {
				updates, err := call.Follow(ctx, env.Calls, env.Target(), appscalls.List, appscalls.ListIn{}, nil)
				if err != nil {
					return err
				}
				if err := io.LivePage(kit.Updates(updates, func(r []apps.Report) page.Page {
					return pageapps.List(r, d.Apps, thisMac, in.All)
				})); err != nil {
					return err
				}
				return ctx.Err()
			}
			reports, err := call.Read(ctx, env.Calls, env.Target(), appscalls.List, appscalls.ListIn{}, nil)
			if err != nil {
				return err
			}
			return io.Page(pageapps.List(reports, d.Apps, thisMac, in.All), pageapps.ListContract(call.Schema(appscalls.List.Name), reports, d.Apps))
		},
	}, env)
	cmd.AddCommand(children...)
	return cmd
}
