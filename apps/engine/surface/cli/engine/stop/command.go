package enginestop

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

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
	// value: what the call said it would end, when it asked.
	asked enginecalls.StopOut
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "stop",
		Short:   "Stop everything on this Mac: its VMs (every disk kept), Awake, the sharing, then the engine",
		Example: "  station engine stop\n  station engine stop --yes",
		Asks:    true,
		Ask: func(in *Input) (bool, error) {
			return env.Prompt.Confirm(pageengine.StopConfirm(in.asked.Work, d.ThisMachine.Machine))
		},
		Refuse: func(in *Input) error { return kit.Refuse(unconfirmed(in.asked), "station engine stop --yes") },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			var out enginecalls.StopOut
			err := io.Confirmed(func(answer string) (err error) {
				out, err = call.Invoke(ctx, env.Calls, env.Target(), enginecalls.Stop, enginecalls.StopIn{Confirm: answer}, nil)
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
