package appsmove

import (
	"context"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/src/apps"
	appscalls "github.com/jterrazz/studio-station/src/apps/calls"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/page"
	pageapps "github.com/jterrazz/studio-station/src/pages/apps"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type Deps struct {
	Names       func() []string
	ThisMachine page.Context
	Now         func() time.Time
	// value: the keys the catalog's managers install by, in its order.
	Managers []string
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "move <app> --to <manager>",
		Short:   "Switch an installed app to another of its managers: installed by that one, then the old copy uninstalled",
		Example: "  station apps move ripgrep --to " + first(d.Managers),
		Args:    cobra.ExactArgs(1),
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringVar(&in.To, "to", "", "the manager it moves to, as a Machinefile names it: "+strings.Join(d.Managers, ", "))
		},
		Bind:     binder(d.Managers),
		Complete: func([]string, string) []string { return d.Names() },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			start := d.Now()
			steps := kit.Steps(io)
			outcomes, err := call.Invoke(ctx, env.Calls, env.Target(), appscalls.Move, appscalls.MoveIn{Name: in.Name, To: in.To}, steps)
			steps.Stop()
			if err != nil {
				return err
			}
			run := pageapps.Run{Took: d.Now().Sub(start)}
			if env.Far() {
				run.On = env.Target().Name
			}
			if err := io.Page(pageapps.Outcomes("move", kit.Worded(outcomes), run, d.ThisMachine), nil); err != nil {
				return err
			}
			n, first := apps.FailedOf(outcomes)
			if n == 0 {
				return nil
			}
			return kit.PartialFailure{Failed: n, Total: len(outcomes), Noun: "apps", FirstHint: "station apps move " + first + " --to " + in.To}
		},
	}, env)
}

func first(list []string) string {
	if len(list) == 0 {
		return ""
	}
	return list[0]
}
