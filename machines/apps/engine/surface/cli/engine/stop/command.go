package enginestop

import (
	"context"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	enginecalls "github.com/jterrazz/studio-station/apps/engine/engine/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageengine "github.com/jterrazz/studio-station/apps/engine/pages/engine"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
}

type Input struct {
	remove bool
	// value: what the call said it would end, when it asked.
	asked enginecalls.StopOut
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "stop",
		Short:   "Stop everything on this Mac: its VMs (every disk kept), Awake, the sharing, then the engine",
		Example: "  station engine stop\n  station engine stop --yes\n  station engine stop --yes --remove",
		Asks:    true,
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.BoolVar(&in.remove, "remove", false, "After the stop, remove the engine's two jobs from this Mac, so the engine reads not installed")
		},
		Ask: func(in *Input) (bool, error) {
			return env.Prompt.Confirm(pageengine.StopConfirm(in.asked.Work, d.ThisMachine.Machine))
		},
		Refuse: func(in *Input) error { return kit.Refuse(unconfirmed(in.asked), "station engine stop --yes") },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			var out enginecalls.StopOut
			err := io.Confirmed(func(answer string) (err error) {
				out, err = call.Invoke(ctx, env.Calls, env.Target(), enginecalls.Stop, enginecalls.StopIn{Confirm: answer, Remove: in.remove}, nil)
				if _, asks := confirm.Answer(err); asks {
					in.asked = out
				}
				return err
			})
			if out.Report == nil {
				return err
			}
			if pageErr := io.Page(pageengine.Stop(*out.Report, d.ThisMachine), nil); pageErr != nil {
				return pageErr
			}
			return err
		},
	}, env)
}

func unconfirmed(out enginecalls.StopOut) string {
	names := make([]string, 0, len(out.Work.VMs))
	for _, v := range out.Work.VMs {
		names = append(names, v.Name)
	}
	return "station engine stop stops " + strings.Join(names, ", ")
}
