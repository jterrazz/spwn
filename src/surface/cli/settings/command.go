package settingscmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/page"
	pagesettings "github.com/jterrazz/studio-station/src/pages/settings"
	"github.com/jterrazz/studio-station/src/settings"
	settingscalls "github.com/jterrazz/studio-station/src/settings/calls"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type Deps struct {
	// value: every setting of the catalogue, for completion.
	Names       func() []string
	ThisMachine page.Context
	Now         func() time.Time
	// value: the name a machine's column is headed by (`mac-mini`).
	Label func(string) string
	// value: this station's catalog entry of a setting, for a row a far
	// station sent without one.
	Setting func(string) (catalog.Setting, bool)
}

type Input struct {
	Name        string
	All         bool
	AllMachines bool
}

func Command(d Deps, env kit.Env, children ...*cobra.Command) *cobra.Command {
	cmd := kit.Command(kit.Leaf[Input]{
		Use:     "settings [<setting>]",
		Short:   "What station sets on this machine: applied, changed since, off here",
		Example: "  station settings\n  station settings starship-dotfiles\n  station settings --all-machines\n  station settings --json",
		Group:   kit.OnAMachine,
		Args:    cobra.MaximumNArgs(1),
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.BoolVar(&in.All, "all", false, "unfold every group")
			f.BoolVar(&in.AllMachines, "all-machines", false, "one column per Mac and workspace; servers are not checked")
		},
		Bind: func(in *Input, args []string) error {
			in.Name = ""
			if len(args) == 1 {
				in.Name = args[0]
			}
			return nil
		},
		Read:     func(*Input) bool { return true },
		Pages:    true,
		Complete: func([]string, string) []string { return d.Names() },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			if in.AllMachines {
				m, err := call.Invoke(ctx, env.Calls, env.Target(), settingscalls.Matrix, settingscalls.MatrixIn{}, nil)
				if err != nil {
					return err
				}
				return io.Page(pagesettings.Matrix(m, d.Now(), in.All, d.Label, d.ThisMachine), pagesettings.MatrixContract(m, d.Now(), d.ThisMachine))
			}
			if in.Name != "" {
				row, err := call.Invoke(ctx, env.Calls, env.Target(), settingscalls.Show, settingscalls.ShowIn{Name: in.Name}, nil)
				if err != nil {
					return err
				}
				row = settings.Known([]settings.Row{row}, d.Setting)[0]
				return io.Page(pagesettings.Show(row, d.Now(), d.ThisMachine), pagesettings.ShowContract(row))
			}
			rows, err := call.Read(ctx, env.Calls, env.Target(), settingscalls.List, settingscalls.ListIn{}, nil)
			if err != nil {
				return err
			}
			rows = settings.Known(rows, d.Setting)
			thisMac := d.ThisMachine
			if env.Far() {
				thisMac = page.Context{Machine: env.Target().Name}
			}
			return io.Page(pagesettings.List(rows, d.Now(), in.All, thisMac), pagesettings.ListContract(rows))
		},
	}, env)
	cmd.AddCommand(children...)
	return cmd
}
